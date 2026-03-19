// Package repository implements data access for the card service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested card record does not exist.
var ErrNotFound = errors.New("repository: card not found")

// ErrAlreadyExists is returned when a DID already has an active card record.
var ErrAlreadyExists = errors.New("repository: card already exists for DID")

// CardRecord represents a row in the cards table.
type CardRecord struct {
	ID                 string
	DID                string
	MRZLine1           string
	MRZLine2           string
	ChipDataHex        string
	QRPayloadB64       string
	IssuerSig          string
	Status             string // "active" | "invalidated"
	IssuedAt           time.Time
	ExpiresAt          time.Time
	InvalidatedAt      *time.Time
	InvalidationReason *string
}

// Repository provides access to the cards table.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository backed by pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// NewPool creates a pgxpool from a connection URL and verifies connectivity.
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

// Create inserts a new card record.
func (r *Repository) Create(ctx context.Context, rec CardRecord) error {
	q := `
		INSERT INTO cards
			(id, did, mrz_line1, mrz_line2, chip_data_hex, qr_payload_b64, issuer_sig, status, issued_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.DID, rec.MRZLine1, rec.MRZLine2, rec.ChipDataHex,
		rec.QRPayloadB64, rec.IssuerSig, rec.Status, rec.IssuedAt, rec.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create card: %w", err)
	}
	return nil
}

// GetByDID fetches the card record for a given DID.
func (r *Repository) GetByDID(ctx context.Context, did string) (*CardRecord, error) {
	q := `
		SELECT id, did, mrz_line1, mrz_line2, chip_data_hex, qr_payload_b64, issuer_sig,
		       status, issued_at, expires_at, invalidated_at, invalidation_reason
		FROM cards WHERE did = $1
	`
	row := r.pool.QueryRow(ctx, q, did)
	var rec CardRecord
	err := row.Scan(
		&rec.ID, &rec.DID, &rec.MRZLine1, &rec.MRZLine2, &rec.ChipDataHex,
		&rec.QRPayloadB64, &rec.IssuerSig, &rec.Status, &rec.IssuedAt, &rec.ExpiresAt,
		&rec.InvalidatedAt, &rec.InvalidationReason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get card by DID: %w", err)
	}
	return &rec, nil
}

// Invalidate marks a card as invalidated.
func (r *Repository) Invalidate(ctx context.Context, did, reason string) error {
	now := time.Now().UTC()
	q := `
		UPDATE cards
		SET status = 'invalidated', invalidated_at = $2, invalidation_reason = $3
		WHERE did = $1 AND status = 'active'
	`
	tag, err := r.pool.Exec(ctx, q, did, now, reason)
	if err != nil {
		return fmt.Errorf("repository: invalidate card: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Check whether the DID simply doesn't exist.
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT TRUE FROM cards WHERE did = $1`, did).Scan(&exists)
		if !exists {
			return ErrNotFound
		}
		// Card exists but is already invalidated — idempotent, treat as success.
	}
	return nil
}
