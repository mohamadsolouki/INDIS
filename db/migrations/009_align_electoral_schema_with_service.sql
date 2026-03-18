-- Migration 009: Align electoral schema with service repository contract.

-- elections table alignment
ALTER TABLE elections
    ALTER COLUMN id DROP DEFAULT;

ALTER TABLE elections
    ALTER COLUMN id TYPE TEXT USING id::text;

ALTER TABLE elections
    ADD COLUMN IF NOT EXISTS name TEXT,
    ADD COLUMN IF NOT EXISTS opens_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS closes_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS admin_did TEXT;

UPDATE elections
SET
    name = COALESCE(name, title),
    opens_at = COALESCE(opens_at, start_time),
    closes_at = COALESCE(closes_at, end_time),
    admin_did = COALESCE(admin_did, '');

-- ballots table alignment
ALTER TABLE ballots
    DROP CONSTRAINT IF EXISTS ballots_election_id_fkey;

ALTER TABLE ballots
    ALTER COLUMN id DROP DEFAULT;

ALTER TABLE ballots
    ALTER COLUMN id TYPE TEXT USING id::text,
    ALTER COLUMN election_id TYPE TEXT USING election_id::text;

ALTER TABLE ballots
    ADD CONSTRAINT ballots_election_id_fkey
    FOREIGN KEY (election_id) REFERENCES elections(id);
