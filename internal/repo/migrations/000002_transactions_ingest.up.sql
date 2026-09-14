-- Ledger schema v1 — transactions, raw messages, statement uploads, review queue

-- Candidate + final transaction records. All money as numeric(14,2);
-- amount is always positive, direction carries the sign meaning (D-05).
CREATE TABLE transactions (
    id              TEXT PRIMARY KEY,                -- txn_...
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id       TEXT REFERENCES devices(id) ON DELETE SET NULL,
    amount          NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    currency        TEXT NOT NULL DEFAULT 'ETB',
    direction       TEXT NOT NULL CHECK (direction IN ('debit','credit')),
    counterparty    TEXT,
    category        TEXT,
    category_confidence REAL CHECK (category_confidence IS NULL OR (category_confidence >= 0 AND category_confidence <= 1)),
    category_source TEXT CHECK (category_source IN ('system','user')),
    source          TEXT NOT NULL CHECK (source IN ('sms','statement')),
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tx_user_time ON transactions(user_id, occurred_at DESC);
CREATE INDEX idx_tx_user_category ON transactions(user_id, category);

-- Every ingested SMS is retained for idempotency and parser reprocessing (FR-2.4).
-- client_message_id is unique per device → idempotent ingestion at the DB layer (D-10).
CREATE TABLE raw_messages (
    id                  TEXT PRIMARY KEY,            -- msg_...
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id           TEXT REFERENCES devices(id) ON DELETE SET NULL,
    client_message_id   TEXT NOT NULL,
    sender_id           TEXT NOT NULL,
    body                TEXT NOT NULL,
    received_at         TIMESTAMPTZ NOT NULL,
    status              TEXT NOT NULL DEFAULT 'parsed'
                        CHECK (status IN ('parsed','queued_for_review','failed')),
    transaction_id      TEXT REFERENCES transactions(id) ON DELETE SET NULL,
    parse_error         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (device_id, client_message_id)
);

CREATE INDEX idx_raw_messages_user ON raw_messages(user_id, received_at DESC);

-- Statement uploads (FR-7); file bytes retained for reprocessing.
CREATE TABLE statement_uploads (
    id              TEXT PRIMARY KEY,               -- up_...
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_hint       TEXT,
    mime_type       TEXT NOT NULL,
    file_bytes      BYTEA NOT NULL,
    size_bytes      BIGINT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'processing'
                    CHECK (status IN ('processing','completed','failed')),
    error           TEXT,
    transactions_created INT NOT NULL DEFAULT 0,
    transactions_flagged INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Review queue for SMS that matched no parser (FR-3.3; engineer-facing, D-18).
CREATE TABLE review_queue (
    id          BIGSERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    raw_message_id TEXT NOT NULL REFERENCES raw_messages(id) ON DELETE CASCADE,
    reason      TEXT NOT NULL,
    resolved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_review_queue_open ON review_queue(resolved_at) WHERE resolved_at IS NULL;
