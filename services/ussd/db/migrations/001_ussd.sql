CREATE TABLE IF NOT EXISTS ussd_sessions (
    session_id TEXT PRIMARY KEY,
    phone_number_hash TEXT NOT NULL, -- SHA-256 of phone, never plain
    service_code TEXT NOT NULL,
    current_step INTEGER NOT NULL DEFAULT 0,
    flow_type TEXT NOT NULL, -- 'voter', 'pension', 'credential'
    locale TEXT NOT NULL DEFAULT 'fa',
    state_data JSONB NOT NULL DEFAULT '{}', -- temp state, wiped on END
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sms_otps (
    id TEXT PRIMARY KEY,
    phone_number_hash TEXT NOT NULL,
    otp_hash TEXT NOT NULL, -- SHA-256 of OTP
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sms_otps_phone ON sms_otps(phone_number_hash);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_active ON ussd_sessions(last_active_at) WHERE ended_at IS NULL;
