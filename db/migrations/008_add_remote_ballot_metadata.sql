-- Migration 008: Remote ballot metadata for anti-replay and auditability.

ALTER TABLE ballots
    ADD COLUMN IF NOT EXISTS receipt_hash TEXT,
    ADD COLUMN IF NOT EXISTS block_height TEXT,
    ADD COLUMN IF NOT EXISTS remote_network TEXT,
    ADD COLUMN IF NOT EXISTS client_attestation_hash TEXT,
    ADD COLUMN IF NOT EXISTS transport_nonce_hash TEXT,
    ADD COLUMN IF NOT EXISTS client_submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS accepted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_ballots_receipt_hash
    ON ballots (receipt_hash);

CREATE INDEX IF NOT EXISTS idx_ballots_remote_network
    ON ballots (remote_network);
