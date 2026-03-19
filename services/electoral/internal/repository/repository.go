// Package repository implements data access for the electoral service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ElectionRecord is a row in the elections table.
type ElectionRecord struct {
	ID          string
	Name        string
	Status      string // scheduled | open | closed | tallied
	OpensAt     time.Time
	ClosesAt    time.Time
	AdminDID    string
	BallotCount int64
	CreatedAt   time.Time
}

// BallotRecord is a row in the ballots table.
type BallotRecord struct {
	ReceiptHash           string
	ElectionID            string
	NullifierHash         string
	EncryptedVote         []byte
	ZKProof               []byte
	BlockHeight           string
	RemoteNetwork         *string
	ClientAttestationHash *string
	TransportNonceHash    *string
	ClientSubmittedAt     *time.Time
	AcceptedAt            *time.Time
	CastAt                time.Time
}

var ErrNotFound = errors.New("repository: record not found")
var ErrNullifierUsed = errors.New("repository: nullifier already used (double-vote)")

// Repository provides access to elections and ballots.
type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("repository: connect: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("repository: ping: %w", err)
	}
	return pool, nil
}

func (r *Repository) CreateElection(ctx context.Context, rec ElectionRecord) error {
	q := `INSERT INTO elections (id, name, status, opens_at, closes_at, admin_did, ballot_count, created_at)
	      VALUES ($1,$2,$3,$4,$5,$6,0,$7)`
	_, err := r.pool.Exec(ctx, q, rec.ID, rec.Name, rec.Status, rec.OpensAt, rec.ClosesAt, rec.AdminDID, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("repository: create election: %w", err)
	}
	return nil
}

func (r *Repository) GetElection(ctx context.Context, id string) (*ElectionRecord, error) {
	q := `SELECT id, name, status, opens_at, closes_at, admin_did, ballot_count, created_at FROM elections WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	var rec ElectionRecord
	err := row.Scan(&rec.ID, &rec.Name, &rec.Status, &rec.OpensAt, &rec.ClosesAt, &rec.AdminDID, &rec.BallotCount, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get election: %w", err)
	}
	return &rec, nil
}

func (r *Repository) NullifierExists(ctx context.Context, electionID, nullifierHash string) (bool, error) {
	var exists bool
	q := `SELECT EXISTS(SELECT 1 FROM ballots WHERE election_id = $1 AND nullifier_hash = $2)`
	if err := r.pool.QueryRow(ctx, q, electionID, nullifierHash).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository: check nullifier: %w", err)
	}
	return exists, nil
}

func (r *Repository) TransportNonceExistsSince(ctx context.Context, electionID, nonceHash string, since time.Time) (bool, error) {
	var exists bool
	q := `SELECT EXISTS(
		SELECT 1
		FROM ballots
		WHERE election_id = $1
		  AND transport_nonce_hash = $2
		  AND COALESCE(accepted_at, cast_at) >= $3
	)`
	if err := r.pool.QueryRow(ctx, q, electionID, nonceHash, since).Scan(&exists); err != nil {
		return false, fmt.Errorf("repository: check transport nonce: %w", err)
	}
	return exists, nil
}

func (r *Repository) CastBallot(ctx context.Context, rec BallotRecord) error {
	q := `INSERT INTO ballots (
			receipt_hash,
			election_id,
			nullifier_hash,
			encrypted_vote,
			zk_proof,
			block_height,
			remote_network,
			client_attestation_hash,
			transport_nonce_hash,
			client_submitted_at,
			accepted_at,
			cast_at
	      ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.pool.Exec(ctx, q, rec.ReceiptHash, rec.ElectionID, rec.NullifierHash,
		rec.EncryptedVote, rec.ZKProof, rec.BlockHeight, rec.RemoteNetwork,
		rec.ClientAttestationHash, rec.TransportNonceHash, rec.ClientSubmittedAt,
		rec.AcceptedAt, rec.CastAt)
	if err != nil {
		return fmt.Errorf("repository: cast ballot: %w", err)
	}
	// Increment ballot count.
	_, _ = r.pool.Exec(ctx, `UPDATE elections SET ballot_count = ballot_count + 1 WHERE id = $1`, rec.ElectionID)
	return nil
}
