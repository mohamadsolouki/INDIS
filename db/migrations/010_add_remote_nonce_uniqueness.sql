-- Migration 010: Enforce remote transport nonce uniqueness per election.

CREATE UNIQUE INDEX IF NOT EXISTS idx_ballots_election_nonce_unique
    ON ballots (election_id, transport_nonce_hash)
    WHERE transport_nonce_hash IS NOT NULL;
