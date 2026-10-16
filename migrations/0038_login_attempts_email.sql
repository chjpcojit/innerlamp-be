-- login_attempts keyed by email instead of phone number (ADR-0003).
-- docs/specs/features/onboarding/design.md "Migration 0037" step 5, split out of
-- 0037 so it lands together with the login code that reads it
-- (docs/specs/features/login/tasks.md Phase 1).
--
-- The counter still has NO foreign key to users: the lock must apply to emails
-- with no account, or the lock itself would reveal which emails are registered
-- (login AC5). The reasoning in 0002 stands; only the key changes.
--
-- Depends on 0002. Idempotent (guarded TRUNCATE, IF EXISTS / IF NOT EXISTS,
-- constraint via DO block), like every other file here.

-- Transient data only (ADR-0003: no real users yet). Guarded on the old column
-- so the wipe happens once: `make migrate` re-applies every file and must not
-- clear live locks each time. It also has to run before ADD COLUMN ... NOT NULL,
-- which fails on a non-empty table without a default.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'login_attempts' AND column_name = 'phone_number'
    ) THEN
        TRUNCATE login_attempts;
    END IF;
END $$;

-- Dropping phone_number drops login_attempts_pkey with it.
ALTER TABLE login_attempts DROP COLUMN IF EXISTS phone_number;
ALTER TABLE login_attempts ADD COLUMN IF NOT EXISTS email VARCHAR(254) NOT NULL;

-- The PRIMARY KEY is the target of ON CONFLICT (email) in ReserveLoginAttempt.
-- Named like the PK 0002 created, so tooling sees the same constraint name.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'login_attempts_pkey'
    ) THEN
        ALTER TABLE login_attempts ADD CONSTRAINT login_attempts_pkey PRIMARY KEY (email);
    END IF;
END $$;
