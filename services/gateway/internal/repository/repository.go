// Package repository provides data access for gateway-owned tables:
// consent_rules and data_export_requests (PRD §FR-008).
//
// Schema (applied at startup via Migrate):
//
//	CREATE TABLE IF NOT EXISTS consent_rules (
//	    id                TEXT        PRIMARY KEY,
//	    citizen_did       TEXT        NOT NULL,
//	    verifier_category TEXT        NOT NULL,
//	    credential_type   TEXT        NOT NULL,
//	    rule              TEXT        NOT NULL DEFAULT 'ask',  -- 'always'|'ask'|'never'
//	    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
//	);
//	CREATE INDEX IF NOT EXISTS idx_consent_rules_did ON consent_rules(citizen_did);
//
//	CREATE TABLE IF NOT EXISTS data_export_requests (
//	    id           TEXT        PRIMARY KEY,
//	    citizen_did  TEXT        NOT NULL,
//	    status       TEXT        NOT NULL DEFAULT 'pending',
//	    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//	    completed_at TIMESTAMPTZ
//	);
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConsentRule represents a single citizen consent rule stored in the gateway DB.
type ConsentRule struct {
	ID               string
	CitizenDID       string
	VerifierCategory string
	CredentialType   string
	Rule             string // "always" | "ask" | "never"
	CreatedAt        time.Time
}

// DataExportRequest represents a citizen data-export request.
type DataExportRequest struct {
	ID          string
	CitizenDID  string
	Status      string // "pending" | "processing" | "completed" | "failed"
	RequestedAt time.Time
	CompletedAt *time.Time
}

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("repository: record not found")

// Repository provides access to the gateway's own Postgres tables.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository backed by the given pool.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// NewPool opens a pgxpool connection and pings the database.
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

// Migrate applies the gateway DDL if the tables do not yet exist.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS consent_rules (
		id                TEXT        PRIMARY KEY,
		citizen_did       TEXT        NOT NULL,
		verifier_category TEXT        NOT NULL,
		credential_type   TEXT        NOT NULL,
		rule              TEXT        NOT NULL DEFAULT 'ask',
		created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_consent_rules_did ON consent_rules(citizen_did);

	CREATE TABLE IF NOT EXISTS data_export_requests (
		id           TEXT        PRIMARY KEY,
		citizen_did  TEXT        NOT NULL,
		status       TEXT        NOT NULL DEFAULT 'pending',
		requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		completed_at TIMESTAMPTZ
	);
	`
	_, err := pool.Exec(ctx, ddl)
	if err != nil {
		return fmt.Errorf("repository: migrate: %w", err)
	}
	return nil
}

// ── Consent rules ─────────────────────────────────────────────────────────────

// InsertConsentRule stores a new consent rule. The caller must supply a unique ID.
func (r *Repository) InsertConsentRule(ctx context.Context, rule ConsentRule) error {
	q := `
		INSERT INTO consent_rules (id, citizen_did, verifier_category, credential_type, rule, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		rule.ID, rule.CitizenDID, rule.VerifierCategory, rule.CredentialType, rule.Rule, rule.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: insert consent rule: %w", err)
	}
	return nil
}

// ListConsentRules returns all consent rules for the given citizen DID, ordered by
// creation time descending.
func (r *Repository) ListConsentRules(ctx context.Context, citizenDID string) ([]ConsentRule, error) {
	q := `
		SELECT id, citizen_did, verifier_category, credential_type, rule, created_at
		FROM consent_rules
		WHERE citizen_did = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, citizenDID)
	if err != nil {
		return nil, fmt.Errorf("repository: list consent rules: %w", err)
	}
	defer rows.Close()

	var rules []ConsentRule
	for rows.Next() {
		var cr ConsentRule
		if err = rows.Scan(&cr.ID, &cr.CitizenDID, &cr.VerifierCategory, &cr.CredentialType, &cr.Rule, &cr.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan consent rule: %w", err)
		}
		rules = append(rules, cr)
	}
	return rules, rows.Err()
}

// DeleteConsentRule removes the consent rule with the given ID, only if it belongs to
// the specified citizen. Returns ErrNotFound if no row was deleted.
func (r *Repository) DeleteConsentRule(ctx context.Context, id, citizenDID string) error {
	q := `DELETE FROM consent_rules WHERE id = $1 AND citizen_did = $2`
	tag, err := r.pool.Exec(ctx, q, id, citizenDID)
	if err != nil {
		return fmt.Errorf("repository: delete consent rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Data-export requests ───────────────────────────────────────────────────────

// InsertDataExportRequest stores a new data-export request.
func (r *Repository) InsertDataExportRequest(ctx context.Context, req DataExportRequest) error {
	q := `
		INSERT INTO data_export_requests (id, citizen_did, status, requested_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, q, req.ID, req.CitizenDID, req.Status, req.RequestedAt)
	if err != nil {
		return fmt.Errorf("repository: insert data export request: %w", err)
	}
	return nil
}

// GetDataExportRequest retrieves a single data-export request by ID, scoped to the
// citizen to prevent information leakage. Returns ErrNotFound when absent.
func (r *Repository) GetDataExportRequest(ctx context.Context, id, citizenDID string) (*DataExportRequest, error) {
	q := `
		SELECT id, citizen_did, status, requested_at, completed_at
		FROM data_export_requests
		WHERE id = $1 AND citizen_did = $2
	`
	row := r.pool.QueryRow(ctx, q, id, citizenDID)
	var req DataExportRequest
	err := row.Scan(&req.ID, &req.CitizenDID, &req.Status, &req.RequestedAt, &req.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get data export request: %w", err)
	}
	return &req, nil
}
