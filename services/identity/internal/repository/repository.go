// Package repository implements data access for the identity service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DIDRecord is a row in the identities table.
type DIDRecord struct {
	DID         string
	PublicKeyHex string
	Document    []byte // JSON-encoded DID Document
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Deactivated bool
}

// ErrNotFound is returned when a DID is not found in the database.
var ErrNotFound = errors.New("repository: DID not found")

// Repository provides CRUD operations for DID records backed by PostgreSQL.
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

// Create inserts a new DID record. Returns an error if the DID already exists.
func (r *Repository) Create(ctx context.Context, rec DIDRecord) error {
	q := `
		INSERT INTO identities (did, public_key_hex, document, created_at, updated_at, deactivated)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.DID, rec.PublicKeyHex, rec.Document,
		rec.CreatedAt, rec.UpdatedAt, rec.Deactivated,
	)
	if err != nil {
		return fmt.Errorf("repository: create: %w", err)
	}
	return nil
}

// GetByDID fetches a DID record by its DID string. Returns ErrNotFound if absent.
func (r *Repository) GetByDID(ctx context.Context, did string) (*DIDRecord, error) {
	q := `
		SELECT did, public_key_hex, document, created_at, updated_at, deactivated
		FROM identities
		WHERE did = $1
	`
	row := r.pool.QueryRow(ctx, q, did)
	var rec DIDRecord
	err := row.Scan(
		&rec.DID, &rec.PublicKeyHex, &rec.Document,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.Deactivated,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get: %w", err)
	}
	return &rec, nil
}

// UpdateDocument replaces the document JSON and bumps updated_at for the given DID.
func (r *Repository) UpdateDocument(ctx context.Context, did string, document []byte) error {
	q := `
		UPDATE identities
		SET document = $2, updated_at = $3
		WHERE did = $1 AND deactivated = FALSE
	`
	tag, err := r.pool.Exec(ctx, q, did, document, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Deactivate marks a DID record as deactivated.
func (r *Repository) Deactivate(ctx context.Context, did string) error {
	q := `
		UPDATE identities
		SET deactivated = TRUE, updated_at = $2
		WHERE did = $1 AND deactivated = FALSE
	`
	tag, err := r.pool.Exec(ctx, q, did, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: deactivate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
