-- Genealogy (gia phả): a clan's family tree and chronicle, owned by one user and
-- readable by the people the owner invites. docs/specs/features/genealogy/ (v5.5).
--
-- Five tables:
--   genealogies            the tree; owner_user_id edits, everyone else only reads
--   genealogy_chronicles   the phả ký, one row per tree, created empty with it
--   genealogy_members      the people: blood members and their in-laws (dâu / rể)
--   genealogy_viewers      users who accepted an invitation (read-only access)
--   genealogy_invitations  pending/answered invitations, each tied to one member
--
-- What the schema enforces so the use case cannot get it wrong:
--   * one founder per tree (uq_genealogy_members_founder);
--   * one clan head at most, and only a living male blood member
--     (uq_genealogy_members_clan_head, chk_genealogy_members_clan_head);
--   * a phone number belongs to one member of a tree (uq_genealogy_members_phone),
--     which is what makes "my position = the member with my number" unambiguous;
--   * a parent / spouse / other parent is always in the SAME tree (composite FKs);
--   * a blood member with children cannot be deleted (parent FK is RESTRICT),
--     whereas its in-laws go with it (spouse FK is CASCADE).
--
-- Dates are stored as SOLAR (chap_*, death_*), with *_entered_as_lunar remembering
-- that the user typed them as lunar; the lunar date is derived at read time.
--
-- ON DELETE SET NULL (other_parent_id) is a column list, which needs PostgreSQL >= 15.
--
-- Idempotent (IF NOT EXISTS) like every other file here.

CREATE TABLE IF NOT EXISTS genealogies (
    id             BIGSERIAL PRIMARY KEY,
    owner_user_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name           VARCHAR(100) NOT NULL,
    ancestral_home VARCHAR(200) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- "My trees, newest first" and the owner-side count for the per-user limit.
CREATE INDEX IF NOT EXISTS idx_genealogies_owner ON genealogies (owner_user_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS genealogy_chronicles (
    genealogy_id          BIGINT PRIMARY KEY REFERENCES genealogies(id) ON DELETE CASCADE,
    surname               VARCHAR(50),
    current_location      VARCHAR(200),
    since_year            SMALLINT CONSTRAINT chk_genealogy_chronicles_since_year CHECK (since_year >= 1000),
    origin                VARCHAR(5000),
    history               VARCHAR(20000),
    hall_name             VARCHAR(200),
    hall_address          VARCHAR(300),
    -- ngày chạp họ: a solar date plus whether the user chose it as lunar.
    chap_year             SMALLINT,
    chap_month            SMALLINT,
    chap_day              SMALLINT,
    chap_entered_as_lunar BOOLEAN NOT NULL DEFAULT false,
    hoanh_phi             VARCHAR(200),
    hoanh_phi_reading     VARCHAR(200),
    hoanh_phi_meaning     VARCHAR(1000),
    couplet_upper         VARCHAR(500),
    couplet_upper_reading VARCHAR(500),
    couplet_upper_meaning VARCHAR(1000),
    couplet_lower         VARCHAR(500),
    couplet_lower_reading VARCHAR(500),
    couplet_lower_meaning VARCHAR(1000),
    teaching              VARCHAR(5000),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_genealogy_chronicles_chap_all_or_none CHECK (
        (chap_year IS NULL AND chap_month IS NULL AND chap_day IS NULL)
        OR
        (chap_year IS NOT NULL AND chap_month IS NOT NULL AND chap_day IS NOT NULL)
    ),
    CONSTRAINT chk_genealogy_chronicles_chap_range CHECK (
        (chap_year IS NULL OR chap_year >= 1000)
        AND (chap_month IS NULL OR chap_month BETWEEN 1 AND 12)
        AND (chap_day IS NULL OR chap_day BETWEEN 1 AND 31)
    ),
    CONSTRAINT chk_genealogy_chronicles_chap_lunar_flag CHECK (
        chap_entered_as_lunar = false OR chap_year IS NOT NULL
    )
);

CREATE TABLE IF NOT EXISTS genealogy_members (
    id                     BIGSERIAL PRIMARY KEY,
    genealogy_id           BIGINT NOT NULL REFERENCES genealogies(id) ON DELETE CASCADE,
    kind                   VARCHAR(6) NOT NULL
                           CONSTRAINT chk_genealogy_members_kind CHECK (kind IN ('blood', 'in_law')),
    gender                 VARCHAR(6) NOT NULL
                           CONSTRAINT chk_genealogy_members_gender CHECK (gender IN ('male', 'female')),
    full_name              VARCHAR(100) NOT NULL,
    birth_year             SMALLINT,
    is_alive               BOOLEAN NOT NULL,
    death_year             SMALLINT,
    residence              VARCHAR(200),
    dharma_name            VARCHAR(100),
    tablet_location        VARCHAR(200),
    burial_place           VARCHAR(200),
    -- death anniversary: solar month/day, the year being death_year.
    death_month            SMALLINT,
    death_day              SMALLINT,
    death_entered_as_lunar BOOLEAN,
    phone_number           VARCHAR(20),
    note                   VARCHAR(1000),
    -- a remembered flag, not a role: at most one per tree.
    is_clan_head           BOOLEAN NOT NULL DEFAULT false,
    parent_id              BIGINT,
    other_parent_id        BIGINT,
    spouse_of_id           BIGINT,
    birth_order            SMALLINT,
    generation             SMALLINT NOT NULL CONSTRAINT chk_genealogy_members_generation CHECK (generation >= 1),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Target of the composite FKs below: it is what keeps every relationship inside one tree.
    CONSTRAINT uq_genealogy_members_id_tree UNIQUE (id, genealogy_id),

    CONSTRAINT fk_genealogy_members_parent FOREIGN KEY (parent_id, genealogy_id)
        REFERENCES genealogy_members (id, genealogy_id) ON DELETE RESTRICT,
    CONSTRAINT fk_genealogy_members_spouse FOREIGN KEY (spouse_of_id, genealogy_id)
        REFERENCES genealogy_members (id, genealogy_id) ON DELETE CASCADE,
    CONSTRAINT fk_genealogy_members_other_parent FOREIGN KEY (other_parent_id, genealogy_id)
        REFERENCES genealogy_members (id, genealogy_id) ON DELETE SET NULL (other_parent_id),

    CONSTRAINT chk_genealogy_members_years CHECK (
        (birth_year IS NULL OR birth_year >= 1000)
        AND (death_year IS NULL OR death_year >= 1000)
        AND (birth_year IS NULL OR death_year IS NULL OR death_year >= birth_year)
    ),
    -- Only a deceased member has these.
    CONSTRAINT chk_genealogy_members_alive CHECK (
        NOT is_alive OR (
            death_year IS NULL AND tablet_location IS NULL AND burial_place IS NULL
            AND death_month IS NULL AND death_day IS NULL AND death_entered_as_lunar IS NULL
        )
    ),
    CONSTRAINT chk_genealogy_members_anniversary CHECK (
        (death_month IS NULL) = (death_day IS NULL)
        AND (death_month IS NULL OR death_month BETWEEN 1 AND 12)
        AND (death_day IS NULL OR death_day BETWEEN 1 AND 31)
        AND (death_day IS NULL OR death_year IS NOT NULL)
        AND (death_entered_as_lunar IS NULL OR death_day IS NOT NULL)
    ),
    CONSTRAINT chk_genealogy_members_blood_shape CHECK (kind <> 'blood' OR spouse_of_id IS NULL),
    CONSTRAINT chk_genealogy_members_in_law_shape CHECK (
        kind <> 'in_law' OR (
            spouse_of_id IS NOT NULL AND parent_id IS NULL AND other_parent_id IS NULL
            AND birth_order IS NULL AND NOT is_clan_head
        )
    ),
    CONSTRAINT chk_genealogy_members_other_parent_blood CHECK (other_parent_id IS NULL OR kind = 'blood'),
    CONSTRAINT chk_genealogy_members_birth_order CHECK (
        birth_order IS NULL OR (birth_order BETWEEN 1 AND 1000 AND parent_id IS NOT NULL)
    ),
    CONSTRAINT chk_genealogy_members_clan_head CHECK (
        NOT is_clan_head OR (kind = 'blood' AND gender = 'male' AND is_alive)
    )
);

-- Exactly one founder (a blood member with no parent) per tree.
CREATE UNIQUE INDEX IF NOT EXISTS uq_genealogy_members_founder
    ON genealogy_members (genealogy_id) WHERE kind = 'blood' AND parent_id IS NULL;
-- At most one clan head per tree.
CREATE UNIQUE INDEX IF NOT EXISTS uq_genealogy_members_clan_head
    ON genealogy_members (genealogy_id) WHERE is_clan_head;
-- A phone number belongs to one member of a tree; the same number may appear in another tree.
CREATE UNIQUE INDEX IF NOT EXISTS uq_genealogy_members_phone
    ON genealogy_members (genealogy_id, phone_number) WHERE phone_number IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_genealogy_members_tree ON genealogy_members (genealogy_id, generation, id);
CREATE INDEX IF NOT EXISTS idx_genealogy_members_parent ON genealogy_members (parent_id);
CREATE INDEX IF NOT EXISTS idx_genealogy_members_spouse ON genealogy_members (spouse_of_id);
CREATE INDEX IF NOT EXISTS idx_genealogy_members_other_parent ON genealogy_members (other_parent_id);

CREATE TABLE IF NOT EXISTS genealogy_viewers (
    genealogy_id BIGINT NOT NULL REFERENCES genealogies(id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_by   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (genealogy_id, user_id)
);

-- "Trees I can view".
CREATE INDEX IF NOT EXISTS idx_genealogy_viewers_user ON genealogy_viewers (user_id);

CREATE TABLE IF NOT EXISTS genealogy_invitations (
    id           BIGSERIAL PRIMARY KEY,
    genealogy_id BIGINT NOT NULL REFERENCES genealogies(id) ON DELETE CASCADE,
    member_id    BIGINT NOT NULL,
    -- Normalised like sign-up. There is deliberately no user_id of the invitee: the
    -- invitation is matched to a person by phone number only, so a person who registers
    -- later still finds it.
    phone_number VARCHAR(20) NOT NULL,
    invited_by   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(9) NOT NULL DEFAULT 'pending'
                 CONSTRAINT chk_genealogy_invitations_status CHECK (
                     status IN ('pending', 'accepted', 'declined', 'cancelled', 'expired')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ,

    CONSTRAINT fk_genealogy_invitations_member FOREIGN KEY (member_id, genealogy_id)
        REFERENCES genealogy_members (id, genealogy_id) ON DELETE CASCADE,
    CONSTRAINT chk_genealogy_invitations_responded CHECK ((status = 'pending') = (responded_at IS NULL))
);

-- One waiting invitation per number per tree.
CREATE UNIQUE INDEX IF NOT EXISTS uq_genealogy_invitations_pending
    ON genealogy_invitations (genealogy_id, phone_number) WHERE status = 'pending';
-- The invitee's inbox: "pending invitations for my number".
CREATE INDEX IF NOT EXISTS idx_genealogy_invitations_phone
    ON genealogy_invitations (phone_number) WHERE status = 'pending';
-- The owner's list.
CREATE INDEX IF NOT EXISTS idx_genealogy_invitations_tree
    ON genealogy_invitations (genealogy_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_genealogy_invitations_member ON genealogy_invitations (member_id);
