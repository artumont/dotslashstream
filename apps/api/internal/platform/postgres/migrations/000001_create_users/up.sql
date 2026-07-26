CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY,
    username      VARCHAR     NOT NULL UNIQUE,
    email         VARCHAR     NOT NULL UNIQUE,
    password_hash BYTEA       NOT NULL,
    salt          BYTEA       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    is_admin      BOOLEAN     NOT NULL DEFAULT false
);
