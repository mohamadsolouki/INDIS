// Package repository implements data access for the verifier service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VerifierRecord represents a row in the verifiers table.
type VerifierRecord struct {
	// ID is the UUID primary key.
	ID string
	// OrgName is the display name of the verifier organization.
	OrgName string
	// OrgType is one of 'government', 'private', 'international'.
	OrgType string
	// AuthorizedCredentialTypes lists the credential types this verifier may request.
	AuthorizedCredentialTypes []string
	// GeographicScope defines where this verifier is authorized to operate.
	GeographicScope string
	// MaxVerificationsPerDay is the rate-limit ceiling.
	MaxVerificationsPerDay int32
	// Status is one of 'active', 'suspended', 'revoked'.
	Status string
	// CertificateID is the UUID of the issued certificate.
	CertificateID string
	// PublicKeyHex is the hex-encoded Ed25519 public key.
	PublicKeyHex string
	// RegisteredAt is the creation timestamp.
	RegisteredAt time.Time
	// UpdatedAt is the last modification timestamp.
	UpdatedAt time.Time
}

// VerificationEventRecord represents a row in the verification_events table.
type VerificationEventRecord struct {
	// ID is the UUID primary key.
	ID string
	// VerifierID is the FK to verifiers.id.
	VerifierID string
	// CredentialType is the type of credential that was verified.
	CredentialType string
	// Result is true if the ZK proof was valid.
	Result bool
	// ProofSystem identifies the proof scheme used (groth16, plonk, stark, bulletproofs).
	ProofSystem string
	// Nonce is the one-time nonce supplied in the verification request.
	Nonce string
	// OccurredAt is when the verification attempt took place.
	OccurredAt time.Time
}

// ErrNotFound is returned when a requested record is absent from the database.
var ErrNotFound = errors.New("repository: record not found")

// Repository provides CRUD operations for verifier records backed by PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository connected to the given PostgreSQL pool.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// NewPool creates and validates a pgxpool from a connection URL.
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

// CreateVerifier inserts a new verifier record. Returns an error if the ID already exists.
func (r *Repository) CreateVerifier(ctx context.Context, rec VerifierRecord) error {
	q := `
		INSERT INTO verifiers (
			id, org_name, org_type, authorized_credential_types,
			geographic_scope, max_verifications_per_day, status,
			certificate_id, public_key_hex, registered_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.OrgName, rec.OrgType, rec.AuthorizedCredentialTypes,
		rec.GeographicScope, rec.MaxVerificationsPerDay, rec.Status,
		rec.CertificateID, rec.PublicKeyHex, rec.RegisteredAt, rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create verifier: %w", err)
	}
	return nil
}

// GetVerifierByID fetches a verifier by its UUID. Returns ErrNotFound if absent.
func (r *Repository) GetVerifierByID(ctx context.Context, id string) (*VerifierRecord, error) {
	q := `
		SELECT id, org_name, org_type, authorized_credential_types,
		       geographic_scope, max_verifications_per_day, status,
		       certificate_id, public_key_hex, registered_at, updated_at
		FROM verifiers
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	var rec VerifierRecord
	err := row.Scan(
		&rec.ID, &rec.OrgName, &rec.OrgType, &rec.AuthorizedCredentialTypes,
		&rec.GeographicScope, &rec.MaxVerificationsPerDay, &rec.Status,
		&rec.CertificateID, &rec.PublicKeyHex, &rec.RegisteredAt, &rec.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get verifier: %w", err)
	}
	return &rec, nil
}

// ListVerifiers returns all verifiers, optionally filtered by status.
// If statusFilter is empty, all records are returned.
func (r *Repository) ListVerifiers(ctx context.Context, statusFilter string) ([]*VerifierRecord, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if statusFilter != "" {
		q := `
			SELECT id, org_name, org_type, authorized_credential_types,
			       geographic_scope, max_verifications_per_day, status,
			       certificate_id, public_key_hex, registered_at, updated_at
			FROM verifiers
			WHERE status = $1
			ORDER BY registered_at DESC
		`
		rows, err = r.pool.Query(ctx, q, statusFilter)
	} else {
		q := `
			SELECT id, org_name, org_type, authorized_credential_types,
			       geographic_scope, max_verifications_per_day, status,
			       certificate_id, public_key_hex, registered_at, updated_at
			FROM verifiers
			ORDER BY registered_at DESC
		`
		rows, err = r.pool.Query(ctx, q)
	}
	if err != nil {
		return nil, fmt.Errorf("repository: list verifiers: %w", err)
	}
	defer rows.Close()

	var out []*VerifierRecord
	for rows.Next() {
		var rec VerifierRecord
		if err := rows.Scan(
			&rec.ID, &rec.OrgName, &rec.OrgType, &rec.AuthorizedCredentialTypes,
			&rec.GeographicScope, &rec.MaxVerificationsPerDay, &rec.Status,
			&rec.CertificateID, &rec.PublicKeyHex, &rec.RegisteredAt, &rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan verifier: %w", err)
		}
		out = append(out, &rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list verifiers rows: %w", err)
	}
	return out, nil
}

// UpdateVerifierStatus sets the status field and bumps updated_at.
func (r *Repository) UpdateVerifierStatus(ctx context.Context, id, status string) error {
	q := `
		UPDATE verifiers
		SET status = $2, updated_at = $3
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q, id, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: update verifier status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateVerificationEvent inserts a verification event record.
func (r *Repository) CreateVerificationEvent(ctx context.Context, evt VerificationEventRecord) error {
	q := `
		INSERT INTO verification_events (id, verifier_id, credential_type, result, proof_system, nonce, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, q,
		evt.ID, evt.VerifierID, evt.CredentialType, evt.Result,
		evt.ProofSystem, evt.Nonce, evt.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create verification event: %w", err)
	}
	return nil
}

// ListVerificationEvents returns the most recent verification events for a verifier.
// limit caps the number of results; 0 means no limit.
func (r *Repository) ListVerificationEvents(ctx context.Context, verifierID string, limit int32) ([]*VerificationEventRecord, error) {
	q := `
		SELECT id, verifier_id, credential_type, result, proof_system, nonce, occurred_at
		FROM verification_events
		WHERE verifier_id = $1
		ORDER BY occurred_at DESC
	`
	args := []any{verifierID}
	if limit > 0 {
		q += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: list verification events: %w", err)
	}
	defer rows.Close()

	var out []*VerificationEventRecord
	for rows.Next() {
		var evt VerificationEventRecord
		if err := rows.Scan(
			&evt.ID, &evt.VerifierID, &evt.CredentialType, &evt.Result,
			&evt.ProofSystem, &evt.Nonce, &evt.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan verification event: %w", err)
		}
		out = append(out, &evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list verification events rows: %w", err)
	}
	return out, nil
}
