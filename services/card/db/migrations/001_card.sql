CREATE TABLE IF NOT EXISTS cards (
    id TEXT PRIMARY KEY,
    did TEXT NOT NULL UNIQUE,
    mrz_line1 TEXT NOT NULL,
    mrz_line2 TEXT NOT NULL,
    chip_data_hex TEXT NOT NULL,
    qr_payload_b64 TEXT NOT NULL,
    issuer_sig TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    invalidated_at TIMESTAMPTZ,
    invalidation_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_cards_did ON cards(did);
