CREATE TABLE IF NOT EXISTS invites (
    id         UUID        PRIMARY KEY,
    token_hash BYTEA       NOT NULL UNIQUE,
    max_uses   INTEGER     NOT NULL,
    uses       INTEGER     NOT NULL DEFAULT 0,
    created_by UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp
);
