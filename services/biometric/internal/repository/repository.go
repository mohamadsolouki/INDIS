// Package repository implements data access for the biometric service.
// Biometric templates are stored encrypted (AES-256-GCM). Ref: PRD §FR-004.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TemplateRecord is a row in the biometric_templates table.
type TemplateRecord struct {
	TemplateID    string
	EnrollmentID  string
	Modality      int32
	EncryptedData []byte // AES-256-GCM encrypted template
	Nonce         []byte // GCM nonce (stored separately from ciphertext for clarity)
	Deleted       bool
	CreatedAt     time.Time
	DeletedAt     *time.Time
}

// ErrNotFound is returned when a template is not in the database.
var ErrNotFound = errors.New("repository: biometric template not found")

// Repository provides CRUD for biometric template records.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

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

// Store inserts a new encrypted template record.
func (r *Repository) Store(ctx context.Context, rec TemplateRecord) error {
	q := `
		INSERT INTO biometric_templates
			(template_id, enrollment_id, modality, encrypted_data, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.TemplateID, rec.EnrollmentID, rec.Modality, rec.EncryptedData, rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: store template: %w", err)
	}
	return nil
}

// GetByID fetches a template by its ID.
func (r *Repository) GetByID(ctx context.Context, templateID string) (*TemplateRecord, error) {
	q := `
		SELECT template_id, enrollment_id, modality, encrypted_data, deleted, created_at, deleted_at
		FROM biometric_templates WHERE template_id = $1 AND deleted = FALSE
	`
	row := r.pool.QueryRow(ctx, q, templateID)
	var rec TemplateRecord
	err := row.Scan(&rec.TemplateID, &rec.EnrollmentID, &rec.Modality,
		&rec.EncryptedData, &rec.Deleted, &rec.CreatedAt, &rec.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get template: %w", err)
	}
	return &rec, nil
}

// SoftDelete marks a template as deleted (right to erasure — GDPR/PRD §FR-004).
func (r *Repository) SoftDelete(ctx context.Context, templateID string) error {
	now := time.Now().UTC()
	q := `UPDATE biometric_templates SET deleted = TRUE, deleted_at = $2 WHERE template_id = $1 AND deleted = FALSE`
	tag, err := r.pool.Exec(ctx, q, templateID, now)
	if err != nil {
		return fmt.Errorf("repository: delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByEnrollment returns all active templates for an enrollment ID.
func (r *Repository) ListByEnrollment(ctx context.Context, enrollmentID string) ([]TemplateRecord, error) {
	q := `
		SELECT template_id, enrollment_id, modality, encrypted_data, deleted, created_at, deleted_at
		FROM biometric_templates WHERE enrollment_id = $1 AND deleted = FALSE
	`
	rows, err := r.pool.Query(ctx, q, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("repository: list templates: %w", err)
	}
	defer rows.Close()
	var recs []TemplateRecord
	for rows.Next() {
		var rec TemplateRecord
		if err = rows.Scan(&rec.TemplateID, &rec.EnrollmentID, &rec.Modality,
			&rec.EncryptedData, &rec.Deleted, &rec.CreatedAt, &rec.DeletedAt); err != nil {
			return nil, fmt.Errorf("repository: scan template: %w", err)
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}
