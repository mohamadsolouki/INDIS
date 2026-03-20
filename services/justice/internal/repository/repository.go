// Package repository implements data access for the justice service.
// Testimony and amnesty data carry an additional encryption layer.
// Keys are in judicial multi-party escrow. Ref: PRD §FR-011.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestimonyRecord is a row in the testimonies table.
type TestimonyRecord struct {
	CaseID             string
	ReceiptToken       string // opaque token for linking follow-ups
	EncryptedTestimony []byte
	Category           string
	Locale             string
	Status             string
	LinkedToCaseID     string // if this is a follow-up
	CreatedAt          time.Time
}

// AmnestyRecord is a row in the amnesty_cases table.
type AmnestyRecord struct {
	CaseID               string
	ApplicantDID         string
	EncryptedDeclaration []byte
	Category             string
	Status               string
	Receipt              string
	CreatedAt            time.Time
}

var ErrNotFound = errors.New("repository: case not found")

// Repository provides access to testimony and amnesty tables.
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

func (r *Repository) CreateTestimony(ctx context.Context, rec TestimonyRecord) error {
	q := `INSERT INTO testimonies (case_id, receipt_token, encrypted_testimony, category, locale, status, linked_to_case_id, created_at)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.pool.Exec(ctx, q, rec.CaseID, rec.ReceiptToken, rec.EncryptedTestimony,
		rec.Category, rec.Locale, rec.Status, rec.LinkedToCaseID, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("repository: create testimony: %w", err)
	}
	return nil
}

func (r *Repository) GetTestimonyByReceipt(ctx context.Context, receiptToken string) (*TestimonyRecord, error) {
	q := `SELECT case_id, receipt_token, encrypted_testimony, category, locale, status, linked_to_case_id, created_at
	      FROM testimonies WHERE receipt_token = $1 ORDER BY created_at DESC LIMIT 1`
	row := r.pool.QueryRow(ctx, q, receiptToken)
	var rec TestimonyRecord
	err := row.Scan(&rec.CaseID, &rec.ReceiptToken, &rec.EncryptedTestimony,
		&rec.Category, &rec.Locale, &rec.Status, &rec.LinkedToCaseID, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get testimony: %w", err)
	}
	return &rec, nil
}

func (r *Repository) GetCaseStatus(ctx context.Context, caseID string) (string, time.Time, error) {
	// Check testimonies first, then amnesty_cases.
	var status string
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `SELECT status, created_at FROM testimonies WHERE case_id = $1 LIMIT 1`, caseID).
		Scan(&status, &updatedAt)
	if err == nil {
		return status, updatedAt, nil
	}
	err = r.pool.QueryRow(ctx, `SELECT status, created_at FROM amnesty_cases WHERE case_id = $1`, caseID).
		Scan(&status, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, ErrNotFound
	}
	if err != nil {
		return "", time.Time{}, fmt.Errorf("repository: get case status: %w", err)
	}
	return status, updatedAt, nil
}

func (r *Repository) CreateAmnestyCase(ctx context.Context, rec AmnestyRecord) error {
	q := `INSERT INTO amnesty_cases (case_id, applicant_did, encrypted_declaration, category, status, receipt, created_at)
	      VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := r.pool.Exec(ctx, q, rec.CaseID, rec.ApplicantDID, rec.EncryptedDeclaration,
		rec.Category, rec.Status, rec.Receipt, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("repository: create amnesty case: %w", err)
	}
	return nil
}

// UpdateCaseStatus transitions a case to the next status.
// Checks both testimonies and amnesty_cases tables.
// Valid statuses: received → under_review → referred → closed
func (r *Repository) UpdateCaseStatus(ctx context.Context, caseID, newStatus string) error {
	// Try testimonies first.
	tag, err := r.pool.Exec(ctx, `UPDATE testimonies SET status = $1 WHERE case_id = $2`, newStatus, caseID)
	if err != nil {
		return fmt.Errorf("repository: update testimony status: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	// Try amnesty_cases.
	tag, err = r.pool.Exec(ctx, `UPDATE amnesty_cases SET status = $1 WHERE case_id = $2`, newStatus, caseID)
	if err != nil {
		return fmt.Errorf("repository: update amnesty status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
