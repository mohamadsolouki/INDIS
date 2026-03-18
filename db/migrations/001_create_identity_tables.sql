-- INDIS Identity Service — Initial Schema
-- Migration: 001
-- Applies to: PostgreSQL 16+

-- Enable pgcrypto for gen_random_uuid() if not already loaded.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- identities stores one row per DID.
-- The DID itself is the primary key (it encodes the subject's public key hash).
CREATE TABLE IF NOT EXISTS identities (
    did               TEXT        NOT NULL PRIMARY KEY,
    public_key_hex    TEXT        NOT NULL,
    document          JSONB       NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated       BOOLEAN     NOT NULL DEFAULT FALSE
);

-- Fast lookup of active DIDs.
CREATE INDEX IF NOT EXISTS idx_identities_deactivated ON identities (deactivated);

-- Partial index: only active DIDs to speed up live queries.
CREATE INDEX IF NOT EXISTS idx_identities_active ON identities (did)
    WHERE deactivated = FALSE;

-- identity_audit_log records every state transition for compliance.
-- No personal data — only the DID, action, and actor DID are stored.
-- Ref: PRD §FR-007 (Audit Trail)
CREATE TABLE IF NOT EXISTS identity_audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    did         TEXT        NOT NULL,
    action      TEXT        NOT NULL,   -- 'register' | 'resolve' | 'deactivate'
    actor_did   TEXT,                   -- DID of the operator (NULL for system actions)
    tx_id       TEXT,                   -- blockchain transaction ID
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_did ON identity_audit_log (did);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON identity_audit_log (created_at);
