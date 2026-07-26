CREATE TABLE IF NOT EXISTS settings (
    id                         INTEGER     PRIMARY KEY,
    allow_signup_without_invite BOOLEAN     NOT NULL DEFAULT true,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT current_timestamp
);

INSERT INTO settings (id, allow_signup_without_invite) VALUES (1, true) ON CONFLICT (id) DO NOTHING;
