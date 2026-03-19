-- INDIS Gov Portal Service — initial schema.
-- Provides ministry dashboard user management and bulk credential operations.
-- Implements PRD FR-009, FR-010, FR-011.

CREATE TABLE IF NOT EXISTS portal_users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    ministry      TEXT NOT NULL,
    role          TEXT NOT NULL, -- 'viewer', 'operator', 'senior', 'admin'
    api_key_hash  TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS bulk_operations (
    id             TEXT PRIMARY KEY,
    operation_type TEXT NOT NULL, -- 'issue_credential', 'revoke_credential', 'enroll_batch'
    ministry       TEXT NOT NULL,
    requested_by   TEXT NOT NULL REFERENCES portal_users(id),
    approved_by    TEXT REFERENCES portal_users(id),
    status         TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'executing', 'completed', 'failed'
    target_dids    TEXT[] NOT NULL DEFAULT '{}',
    parameters     JSONB NOT NULL DEFAULT '{}',
    result_summary JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bulk_ops_status   ON bulk_operations(status);
CREATE INDEX IF NOT EXISTS idx_bulk_ops_ministry ON bulk_operations(ministry);
