-- Migration 004: Biometric service tables
-- Encrypted biometric templates with right-to-erasure support (GDPR Article 17).

CREATE TABLE IF NOT EXISTS biometric_templates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id   UUID        NOT NULL,
    modality        TEXT        NOT NULL,          -- FINGERPRINT | FACE | IRIS | VOICE
    encrypted_data  BYTEA       NOT NULL,          -- AES-256-GCM encrypted template
    nonce           BYTEA       NOT NULL,          -- GCM nonce (12 bytes)
    quality_score   FLOAT8      NOT NULL DEFAULT 0,
    deleted         BOOLEAN     NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for enrollment lookups; exclude soft-deleted rows.
CREATE INDEX IF NOT EXISTS idx_biometric_enrollment
    ON biometric_templates (enrollment_id)
    WHERE deleted = FALSE;

-- Separate audit table so delete events are still auditable
-- even after template erasure.
CREATE TABLE IF NOT EXISTS biometric_audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    template_id UUID        NOT NULL,
    action      TEXT        NOT NULL,   -- STORE | DELETE
    actor_did   TEXT        NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
