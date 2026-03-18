-- INDIS Credential Service — Schema
-- Migration: 002
-- Applies to: PostgreSQL 16+

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- credentials stores one row per issued Verifiable Credential.
CREATE TABLE IF NOT EXISTS credentials (
    id            TEXT        NOT NULL PRIMARY KEY,
    subject_did   TEXT        NOT NULL,
    issuer_did    TEXT        NOT NULL,
    type          TEXT        NOT NULL,
    data          JSONB       NOT NULL,           -- full VC JSON including proof
    revoked       BOOLEAN     NOT NULL DEFAULT FALSE,
    revoke_reason TEXT        NOT NULL DEFAULT '',
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for per-subject credential lookups (e.g. list all credentials of a citizen).
CREATE INDEX IF NOT EXISTS idx_credentials_subject ON credentials (subject_did);

-- Partial index for active (non-revoked) credentials only.
CREATE INDEX IF NOT EXISTS idx_credentials_active ON credentials (subject_did)
    WHERE revoked = FALSE;

-- credential_audit_log records every issuance and revocation for compliance.
-- Ref: PRD §FR-007
CREATE TABLE IF NOT EXISTS credential_audit_log (
    id           BIGSERIAL   PRIMARY KEY,
    credential_id TEXT       NOT NULL,
    action        TEXT       NOT NULL,   -- 'issue' | 'revoke' | 'verify'
    actor_did     TEXT,
    tx_id         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cred_audit_credential_id ON credential_audit_log (credential_id);
