// Package repository implements data access for the govportal service.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PortalUserRecord represents a row in the portal_users table.
type PortalUserRecord struct {
	// ID is the UUID primary key.
	ID string
	// Username is the unique login name.
	Username string
	// Ministry is the government ministry this user belongs to.
	Ministry string
	// Role is one of 'viewer', 'operator', 'senior', 'admin'.
	Role string
	// APIKeyHash is a hashed API key for machine clients.
	APIKeyHash string
	// CreatedAt is the record creation timestamp.
	CreatedAt time.Time
	// LastLoginAt records the most recent successful login, if any.
	LastLoginAt *time.Time
}

// BulkOperationRecord represents a row in the bulk_operations table.
type BulkOperationRecord struct {
	// ID is the UUID primary key.
	ID string
	// OperationType is one of 'issue_credential', 'revoke_credential', 'enroll_batch'.
	OperationType string
	// Ministry is the owning ministry.
	Ministry string
	// RequestedBy is the portal_user.id who created the operation.
	RequestedBy string
	// ApprovedBy is the portal_user.id who approved it, or empty.
	ApprovedBy string
	// Status is one of 'pending', 'approved', 'rejected', 'executing', 'completed', 'failed'.
	Status string
	// TargetDIDs lists the DID strings targeted by the operation.
	TargetDIDs []string
	// Parameters is arbitrary JSON metadata for the operation.
	Parameters json.RawMessage
	// ResultSummary is the post-execution JSON summary (nil before completion).
	ResultSummary json.RawMessage
	// CreatedAt is the record creation timestamp.
	CreatedAt time.Time
	// UpdatedAt is the last modification timestamp.
	UpdatedAt time.Time
}

// ErrNotFound is returned when a requested record is absent from the database.
var ErrNotFound = errors.New("repository: record not found")

// Repository provides CRUD operations for govportal records backed by PostgreSQL.
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

// CreatePortalUser inserts a new portal user. Returns an error if the username already exists.
func (r *Repository) CreatePortalUser(ctx context.Context, rec PortalUserRecord) error {
	q := `
		INSERT INTO portal_users (id, username, ministry, role, api_key_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.Username, rec.Ministry, rec.Role, rec.APIKeyHash, rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create portal user: %w", err)
	}
	return nil
}

// GetPortalUserByID fetches a portal user by UUID. Returns ErrNotFound if absent.
func (r *Repository) GetPortalUserByID(ctx context.Context, id string) (*PortalUserRecord, error) {
	q := `
		SELECT id, username, ministry, role, api_key_hash, created_at, last_login_at
		FROM portal_users
		WHERE id = $1
	`
	return r.scanPortalUser(r.pool.QueryRow(ctx, q, id))
}

// GetPortalUserByUsername fetches a portal user by username. Returns ErrNotFound if absent.
func (r *Repository) GetPortalUserByUsername(ctx context.Context, username string) (*PortalUserRecord, error) {
	q := `
		SELECT id, username, ministry, role, api_key_hash, created_at, last_login_at
		FROM portal_users
		WHERE username = $1
	`
	return r.scanPortalUser(r.pool.QueryRow(ctx, q, username))
}

// ListPortalUsers returns all portal users, optionally filtered by ministry.
func (r *Repository) ListPortalUsers(ctx context.Context, ministryFilter string) ([]*PortalUserRecord, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if ministryFilter != "" {
		q := `SELECT id, username, ministry, role, api_key_hash, created_at, last_login_at FROM portal_users WHERE ministry = $1 ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, q, ministryFilter)
	} else {
		q := `SELECT id, username, ministry, role, api_key_hash, created_at, last_login_at FROM portal_users ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, q)
	}
	if err != nil {
		return nil, fmt.Errorf("repository: list portal users: %w", err)
	}
	defer rows.Close()

	var out []*PortalUserRecord
	for rows.Next() {
		var rec PortalUserRecord
		if err := rows.Scan(&rec.ID, &rec.Username, &rec.Ministry, &rec.Role, &rec.APIKeyHash, &rec.CreatedAt, &rec.LastLoginAt); err != nil {
			return nil, fmt.Errorf("repository: scan portal user: %w", err)
		}
		out = append(out, &rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list portal users rows: %w", err)
	}
	return out, nil
}

// UpdatePortalUserRole updates the role of a portal user.
func (r *Repository) UpdatePortalUserRole(ctx context.Context, id, role string) error {
	q := `UPDATE portal_users SET role = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, role)
	if err != nil {
		return fmt.Errorf("repository: update portal user role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateBulkOperation inserts a new bulk operation record.
func (r *Repository) CreateBulkOperation(ctx context.Context, rec BulkOperationRecord) error {
	params := rec.Parameters
	if params == nil {
		params = json.RawMessage("{}")
	}
	q := `
		INSERT INTO bulk_operations (id, operation_type, ministry, requested_by, status, target_dids, parameters, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.OperationType, rec.Ministry, rec.RequestedBy,
		rec.Status, rec.TargetDIDs, params, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create bulk operation: %w", err)
	}
	return nil
}

// GetBulkOperationByID fetches a bulk operation by UUID. Returns ErrNotFound if absent.
func (r *Repository) GetBulkOperationByID(ctx context.Context, id string) (*BulkOperationRecord, error) {
	q := `
		SELECT id, operation_type, ministry, requested_by, COALESCE(approved_by,''),
		       status, target_dids, parameters, result_summary, created_at, updated_at
		FROM bulk_operations
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	var rec BulkOperationRecord
	var resultSummary []byte
	err := row.Scan(
		&rec.ID, &rec.OperationType, &rec.Ministry, &rec.RequestedBy, &rec.ApprovedBy,
		&rec.Status, &rec.TargetDIDs, &rec.Parameters, &resultSummary,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get bulk operation: %w", err)
	}
	rec.ResultSummary = resultSummary
	return &rec, nil
}

// ListBulkOperations returns bulk operations, optionally filtered by status and/or ministry.
func (r *Repository) ListBulkOperations(ctx context.Context, statusFilter, ministryFilter string) ([]*BulkOperationRecord, error) {
	q := `
		SELECT id, operation_type, ministry, requested_by, COALESCE(approved_by,''),
		       status, target_dids, parameters, result_summary, created_at, updated_at
		FROM bulk_operations
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR ministry = $2)
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, statusFilter, ministryFilter)
	if err != nil {
		return nil, fmt.Errorf("repository: list bulk operations: %w", err)
	}
	defer rows.Close()

	var out []*BulkOperationRecord
	for rows.Next() {
		var rec BulkOperationRecord
		var resultSummary []byte
		if err := rows.Scan(
			&rec.ID, &rec.OperationType, &rec.Ministry, &rec.RequestedBy, &rec.ApprovedBy,
			&rec.Status, &rec.TargetDIDs, &rec.Parameters, &resultSummary,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan bulk operation: %w", err)
		}
		rec.ResultSummary = resultSummary
		out = append(out, &rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list bulk operations rows: %w", err)
	}
	return out, nil
}

// ApproveBulkOperation sets the approved_by and status fields.
func (r *Repository) ApproveBulkOperation(ctx context.Context, id, approvedBy, newStatus string) error {
	q := `
		UPDATE bulk_operations
		SET approved_by = $2, status = $3, updated_at = $4
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q, id, approvedBy, newStatus, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: approve bulk operation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetBulkOperationResult updates a bulk operation final state and persists the JSON outcome.
func (r *Repository) SetBulkOperationResult(ctx context.Context, id, approvedBy, newStatus string, resultSummary json.RawMessage) error {
	if resultSummary == nil {
		resultSummary = json.RawMessage("{}")
	}
	q := `
		UPDATE bulk_operations
		SET approved_by = $2,
		    status = $3,
		    result_summary = $4,
		    updated_at = $5
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q, id, approvedBy, newStatus, resultSummary, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("repository: set bulk operation result: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountVerifiers returns the count of all rows in a given table — used for aggregate stats.
// tableName is trusted (internal use only, never from HTTP input).
func (r *Repository) CountRows(ctx context.Context, tableName string) (int64, error) {
	// tableName is supplied only from internal code, never from user input.
	q := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName) //nolint:gosec
	var n int64
	if err := r.pool.QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("repository: count %s: %w", tableName, err)
	}
	return n, nil
}

// scanPortalUser is a shared row scanner for portal_users.
func (r *Repository) scanPortalUser(row pgx.Row) (*PortalUserRecord, error) {
	var rec PortalUserRecord
	err := row.Scan(&rec.ID, &rec.Username, &rec.Ministry, &rec.Role, &rec.APIKeyHash, &rec.CreatedAt, &rec.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: scan portal user: %w", err)
	}
	return &rec, nil
}
