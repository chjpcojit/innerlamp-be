-- Backoffice: admin accounts.
--
-- Separate from `users` on purpose: a backoffice operator is not an app user,
-- has no phone/OTP flow, and must never be reachable through appService's
-- login. Accounts are created out-of-band (services/backofficeService/cmd/
-- seedadmin), never by self-registration.
--
-- email is stored lower-cased by the application; the UNIQUE constraint is the
-- backstop, not the normalizer.
CREATE TABLE IF NOT EXISTS admins (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    role          VARCHAR(32)  NOT NULL DEFAULT 'admin',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
