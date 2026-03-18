-- Migration 005: Notification service tables

CREATE TABLE IF NOT EXISTS notifications (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_did   TEXT        NOT NULL,
    channel         TEXT        NOT NULL,   -- SMS | PUSH | EMAIL | IN_APP
    notification_type TEXT      NOT NULL,   -- CREDENTIAL_ISSUED | CREDENTIAL_EXPIRING | …
    title           TEXT        NOT NULL,
    body            TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'queued',  -- queued | sent | cancelled | failed
    scheduled_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for looking up pending/scheduled notifications by recipient.
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_status
    ON notifications (recipient_did, status, scheduled_at)
    WHERE status IN ('queued', 'sent');

-- Index for expiry-alert lookups (scheduler polls this).
CREATE INDEX IF NOT EXISTS idx_notifications_scheduled
    ON notifications (scheduled_at)
    WHERE status = 'queued';
