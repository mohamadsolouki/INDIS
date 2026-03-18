// Package repository implements data access for the enrollment service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnrollmentStatus represents the lifecycle state of an enrollment session.
type EnrollmentStatus string

const (
	StatusPending               EnrollmentStatus = "pending"
	StatusBiometricsSubmitted   EnrollmentStatus = "biometrics_submitted"
	StatusAttestationSubmitted  EnrollmentStatus = "attestation_submitted"
	StatusCompleted             EnrollmentStatus = "completed"
	StatusFailed                EnrollmentStatus = "failed"
)

// EnrollmentRecord is a row in the enrollments table.
type EnrollmentRecord struct {
	ID              string
	Pathway         string // standard | enhanced | social
	Status          EnrollmentStatus
	AgentID         string
	Locale          string
	BiometricsPassed bool
	AttestorCount   int
	AssignedDID     string // set after CompleteEnrollment
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ErrNotFound is returned when an enrollment ID is not in the database.
var ErrNotFound = errors.New("repository: enrollment not found")

// Repository provides CRUD for enrollment records backed by PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository connected to the given pool.
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

// Create inserts a new enrollment record.
func (r *Repository) Create(ctx context.Context, rec EnrollmentRecord) error {
	q := `
		INSERT INTO enrollments
			(id, pathway, status, agent_id, locale, biometrics_passed, attestor_count, assigned_did, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.Pathway, string(rec.Status), rec.AgentID, rec.Locale,
		rec.BiometricsPassed, rec.AttestorCount, rec.AssignedDID,
		rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create enrollment: %w", err)
	}
	return nil
}

// GetByID fetches an enrollment record by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*EnrollmentRecord, error) {
	q := `
		SELECT id, pathway, status, agent_id, locale, biometrics_passed, attestor_count, assigned_did, created_at, updated_at
		FROM enrollments
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	var rec EnrollmentRecord
	var statusStr string
	err := row.Scan(
		&rec.ID, &rec.Pathway, &statusStr, &rec.AgentID, &rec.Locale,
		&rec.BiometricsPassed, &rec.AttestorCount, &rec.AssignedDID,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get enrollment: %w", err)
	}
	rec.Status = EnrollmentStatus(statusStr)
	return &rec, nil
}

// UpdateStatus updates the status (and optionally biometrics/attestor data) of an enrollment.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status EnrollmentStatus, biometricsPassed bool, attestorCount int) error {
	q := `
		UPDATE enrollments
		SET status = $2, biometrics_passed = $3, attestor_count = $4, updated_at = $5
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q, id, string(status), biometricsPassed, attestorCount, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: update enrollment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Complete marks an enrollment as completed and records the assigned DID.
func (r *Repository) Complete(ctx context.Context, id, assignedDID string) error {
	q := `
		UPDATE enrollments
		SET status = $2, assigned_did = $3, updated_at = $4
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q, id, string(StatusCompleted), assignedDID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: complete enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
