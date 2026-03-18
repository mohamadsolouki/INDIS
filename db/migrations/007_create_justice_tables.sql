-- Migration 007: Justice service tables
-- Supports anonymous testimony (ZK citizenship proof) and conditional amnesty.
-- PRD §FR-010 (testimony), §FR-011 (amnesty requires full DID — no anonymity).

CREATE TABLE IF NOT EXISTS testimonies (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id         TEXT        NOT NULL,
    -- receipt_token is the only link back to the anonymous author (given to submitter).
    receipt_token   TEXT        NOT NULL UNIQUE,
    content_hash    TEXT        NOT NULL,   -- SHA-256 of encrypted testimony payload
    zk_proof        BYTEA,                  -- ZK citizenship proof (stub)
    status          TEXT        NOT NULL DEFAULT 'SUBMITTED',  -- SUBMITTED | UNDER_REVIEW | ACCEPTED | REJECTED
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_testimonies_case
    ON testimonies (case_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_testimonies_receipt
    ON testimonies (receipt_token);

-- Amnesty cases require full DID identification (PRD §FR-011).
CREATE TABLE IF NOT EXISTS amnesty_cases (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    applicant_did   TEXT        NOT NULL,   -- NOT NULL: anonymity is prohibited for amnesty
    case_type       TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL DEFAULT 'SUBMITTED',  -- SUBMITTED | UNDER_REVIEW | APPROVED | REJECTED | CLOSED
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_amnesty_applicant
    ON amnesty_cases (applicant_did);
