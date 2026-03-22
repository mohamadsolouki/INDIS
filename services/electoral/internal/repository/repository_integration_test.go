package repository

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	indismigrate "github.com/mohamadsolouki/INDIS/pkg/migrate"
)

func TestRepositoryCastBallotPersistsRemoteMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	dsn := os.Getenv("ELECTORAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ELECTORAL_TEST_DATABASE_URL to run repository integration tests")
	}

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	_, currentFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../"))
	migrationsDir := filepath.Join(repoRoot, "db", "migrations")

	if err := indismigrate.Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := New(pool)

	electionID := "elc_integration_001"
	if err := repo.CreateElection(ctx, ElectionRecord{
		ID:        electionID,
		Name:      "Integration Election",
		Status:    "scheduled",
		OpensAt:   time.Now().UTC().Add(1 * time.Hour),
		ClosesAt:  time.Now().UTC().Add(24 * time.Hour),
		AdminDID:  "did:indis:admin:integration",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create election: %v", err)
	}

	network := "mobile"
	attestationHash := "attestation-hash-123"
	nonceHash := "nonce-hash-abc"
	clientSubmittedAt := time.Now().UTC().Add(-30 * time.Second)
	acceptedAt := time.Now().UTC()

	if err := repo.CastBallot(ctx, BallotRecord{
		ReceiptHash:           "receipt-hash-1",
		ElectionID:            electionID,
		NullifierHash:         "nullifier-hash-1",
		EncryptedVote:         []byte("ciphertext"),
		ZKProof:               []byte("zk-proof"),
		BlockHeight:           "pending",
		RemoteNetwork:         &network,
		ClientAttestationHash: &attestationHash,
		TransportNonceHash:    &nonceHash,
		ClientSubmittedAt:     &clientSubmittedAt,
		AcceptedAt:            &acceptedAt,
		CastAt:                acceptedAt,
	}); err != nil {
		t.Fatalf("cast ballot: %v", err)
	}

	var gotNetwork, gotAttestationHash, gotNonceHash, gotBlockHeight string
	var gotClientSubmittedAt, gotAcceptedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT remote_network, client_attestation_hash, transport_nonce_hash, block_height, client_submitted_at, accepted_at
		FROM ballots
		WHERE election_id = $1 AND nullifier_hash = $2
	`, electionID, "nullifier-hash-1").Scan(
		&gotNetwork,
		&gotAttestationHash,
		&gotNonceHash,
		&gotBlockHeight,
		&gotClientSubmittedAt,
		&gotAcceptedAt,
	); err != nil {
		t.Fatalf("query ballot metadata: %v", err)
	}

	if gotNetwork != network {
		t.Fatalf("expected network=%s got=%s", network, gotNetwork)
	}
	if gotAttestationHash != attestationHash {
		t.Fatalf("expected attestation hash=%s got=%s", attestationHash, gotAttestationHash)
	}
	if gotNonceHash != nonceHash {
		t.Fatalf("expected nonce hash=%s got=%s", nonceHash, gotNonceHash)
	}
	if gotBlockHeight != "pending" {
		t.Fatalf("expected block_height=pending got=%s", gotBlockHeight)
	}
	if gotClientSubmittedAt.IsZero() || gotAcceptedAt.IsZero() {
		t.Fatalf("expected client_submitted_at and accepted_at to be persisted")
	}

	var ballotCount int64
	if err := pool.QueryRow(ctx, `SELECT ballot_count FROM elections WHERE id = $1`, electionID).Scan(&ballotCount); err != nil {
		t.Fatalf("query election ballot_count: %v", err)
	}
	if ballotCount != 1 {
		t.Fatalf("expected ballot_count=1 got=%d", ballotCount)
	}

	used, err := repo.NullifierExists(ctx, electionID, "nullifier-hash-1")
	if err != nil {
		t.Fatalf("nullifier exists: %v", err)
	}
	if !used {
		t.Fatal("expected nullifier to exist after cast")
	}

	if _, err := repo.GetElection(ctx, electionID); err != nil {
		t.Fatalf("get election: %v", err)
	}

	// ensure unique constraint is still enforced
	err = repo.CastBallot(ctx, BallotRecord{
		ReceiptHash:   "receipt-hash-2",
		ElectionID:    electionID,
		NullifierHash: "nullifier-hash-1",
		EncryptedVote: []byte("ciphertext-2"),
		ZKProof:       []byte("zk-proof-2"),
		BlockHeight:   "pending",
		CastAt:        time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected duplicate nullifier insertion to fail")
	}
	if err.Error() == "" {
		t.Fatal("expected duplicate nullifier error details")
	}
}
