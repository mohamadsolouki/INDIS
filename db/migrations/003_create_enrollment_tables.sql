-- INDIS Enrollment Service — Schema
-- Migration: 003
-- Applies to: PostgreSQL 16+

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- enrollments tracks one enrollment session per row.
-- A session transitions through: pending → biometrics_submitted → (attestation_submitted) → completed | failed
CREATE TABLE IF NOT EXISTS enrollments (
    id                TEXT        NOT NULL PRIMARY KEY,
    pathway           TEXT        NOT NULL CHECK (pathway IN ('standard', 'enhanced', 'social')),
    status            TEXT        NOT NULL DEFAULT 'pending',
    agent_id          TEXT        NOT NULL DEFAULT '',
    locale            TEXT        NOT NULL DEFAULT 'fa',
    biometrics_passed BOOLEAN     NOT NULL DEFAULT FALSE,
    attestor_count    INT         NOT NULL DEFAULT 0,
    assigned_did      TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Partial index on pending enrollments (most queried state).
CREATE INDEX IF NOT EXISTS idx_enrollments_pending ON enrollments (id)
    WHERE status = 'pending';

-- enrollment_audit_log records every status transition.
-- Ref: PRD §FR-007
CREATE TABLE IF NOT EXISTS enrollment_audit_log (
    id            BIGSERIAL   PRIMARY KEY,
    enrollment_id TEXT        NOT NULL,
    from_status   TEXT,
    to_status     TEXT        NOT NULL,
    actor_id      TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_enroll_audit_id ON enrollment_audit_log (enrollment_id);
