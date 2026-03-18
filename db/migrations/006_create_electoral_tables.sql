-- Migration 006: Electoral service tables
-- Double-vote prevention via SHA-256 nullifier hashes (ZK-STARK design).

CREATE TABLE IF NOT EXISTS elections (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL DEFAULT 'REGISTRATION',  -- REGISTRATION | ACTIVE | CLOSED | TALLYING | FINALIZED
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ballot_count    BIGINT      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ballot table stores nullifiers only, never voter identity.
-- nullifier_hash = SHA-256(election_id || voter_did || secret)
CREATE TABLE IF NOT EXISTS ballots (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    election_id     UUID        NOT NULL REFERENCES elections(id),
    nullifier_hash  TEXT        NOT NULL,   -- prevents double-voting
    encrypted_vote  BYTEA,                  -- placeholder for homomorphic tally
    zk_proof        BYTEA,                  -- STARK proof bytes (from services/zkproof)
    cast_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique constraint on nullifier: one vote per voter per election.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ballots_nullifier
    ON ballots (election_id, nullifier_hash);

CREATE INDEX IF NOT EXISTS idx_ballots_election
    ON ballots (election_id);
