-- Ledger schema v1 — categorization rules (learning loop), clarifications, reports

-- Learned categorization rules (FR-4.2 / FR-5.2). Matched on counterparty and/or
-- keyword contained in counterparty; scoped strictly per user.
CREATE TABLE category_rules (
    id          TEXT PRIMARY KEY,               -- rule_...
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_type  TEXT NOT NULL CHECK (match_type IN ('counterparty','keyword')),
    match_value TEXT NOT NULL,
    category    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, match_type, match_value)
);

CREATE INDEX idx_rules_user ON category_rules(user_id);

-- Clarifying questions queued by the agent orchestrator (FR-5.1).
-- options stored as JSONB (array of short strings) for portable scanning.
CREATE TABLE clarifications (
    id              TEXT PRIMARY KEY,           -- clr_...
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_id  TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    question        TEXT NOT NULL,
    options         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','resolved')),
    answer_category TEXT,
    answered_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_clarifications_pending ON clarifications(user_id) WHERE status = 'pending';

-- Weekly reports (FR-6). Narrative and aggregates stored as generated.
CREATE TABLE weekly_reports (
    id              TEXT PRIMARY KEY,           -- rpt_...
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    totals_by_category JSONB NOT NULL,
    week_over_week     JSONB NOT NULL,
    narrative       TEXT,
    generation_status TEXT NOT NULL DEFAULT 'pending'
                      CHECK (generation_status IN ('pending','llm_skipped','completed','failed')),
    generated_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, period_start)
);
