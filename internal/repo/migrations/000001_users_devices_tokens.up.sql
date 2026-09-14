-- Ledger schema v1 — users, devices, tokens
CREATE TABLE users (
    id          TEXT PRIMARY KEY,            -- usr_...
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE devices (
    id          TEXT PRIMARY KEY,            -- dev_...
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_model  TEXT NOT NULL,
    os_version    TEXT NOT NULL,
    api_key_hash  TEXT NOT NULL UNIQUE,      -- SHA-256 of device API key; lookup key
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    token_hash  TEXT PRIMARY KEY,            -- SHA-256 of the opaque token
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_devices_user ON devices(user_id);
