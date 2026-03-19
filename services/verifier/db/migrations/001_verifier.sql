-- INDIS Verifier Service — initial schema.
-- Implements PRD FR-012 (verifier registration) and FR-013 (credential verification).

CREATE TABLE IF NOT EXISTS verifiers (
    id                        TEXT PRIMARY KEY,
    org_name                  TEXT NOT NULL,
    org_type                  TEXT NOT NULL, -- 'government', 'private', 'international'
    authorized_credential_types TEXT[] NOT NULL DEFAULT '{}',
    geographic_scope          TEXT NOT NULL DEFAULT 'nationwide',
    max_verifications_per_day INTEGER NOT NULL DEFAULT 10000,
    status                    TEXT NOT NULL DEFAULT 'active', -- 'active', 'suspended', 'revoked'
    certificate_id            TEXT,
    public_key_hex            TEXT,
    registered_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS verification_events (
    id              TEXT PRIMARY KEY,
    verifier_id     TEXT NOT NULL REFERENCES verifiers(id),
    credential_type TEXT NOT NULL,
    result          BOOLEAN NOT NULL,
    proof_system    TEXT NOT NULL,
    nonce           TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_events_verifier  ON verification_events(verifier_id);
CREATE INDEX IF NOT EXISTS idx_verification_events_occurred  ON verification_events(occurred_at);
