// Package repository implements data access for the credential service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CredentialRecord is a row in the credentials table.
type CredentialRecord struct {
	ID          string
	SubjectDID  string
	IssuerDID   string
	Type        string
	Data        []byte // JSON-encoded VerifiableCredential
	Revoked     bool
	RevokeReason string
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

// ErrNotFound is returned when a credential ID is not in the database.
var ErrNotFound = errors.New("repository: credential not found")

// ErrAlreadyRevoked is returned when trying to revoke an already-revoked credential.
var ErrAlreadyRevoked = errors.New("repository: credential already revoked")

// Repository provides CRUD operations for credential records backed by PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository connected to the given PostgreSQL pool.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// NewPool creates a pgxpool from a connection URL.
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

// Create inserts a new credential record.
func (r *Repository) Create(ctx context.Context, rec CredentialRecord) error {
	q := `
		INSERT INTO credentials (id, subject_did, issuer_did, type, data, revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, FALSE, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.SubjectDID, rec.IssuerDID, rec.Type, rec.Data, rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create credential: %w", err)
	}
	return nil
}

// GetByID fetches a credential by its ID. Returns ErrNotFound if absent.
func (r *Repository) GetByID(ctx context.Context, id string) (*CredentialRecord, error) {
	q := `
		SELECT id, subject_did, issuer_did, type, data, revoked, revoke_reason, revoked_at, created_at
		FROM credentials
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	var rec CredentialRecord
	err := row.Scan(
		&rec.ID, &rec.SubjectDID, &rec.IssuerDID, &rec.Type,
		&rec.Data, &rec.Revoked, &rec.RevokeReason, &rec.RevokedAt, &rec.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get credential: %w", err)
	}
	return &rec, nil
}

// Revoke marks a credential as revoked with the given reason.
func (r *Repository) Revoke(ctx context.Context, id, reason string) error {
	now := time.Now().UTC()
	q := `
		UPDATE credentials
		SET revoked = TRUE, revoke_reason = $2, revoked_at = $3
		WHERE id = $1 AND revoked = FALSE
	`
	tag, err := r.pool.Exec(ctx, q, id, reason, now)
	if err != nil {
		return fmt.Errorf("repository: revoke credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Check if it exists at all.
		if _, err2 := r.GetByID(ctx, id); errors.Is(err2, ErrNotFound) {
			return ErrNotFound
		}
		return ErrAlreadyRevoked
	}
	return nil
}
