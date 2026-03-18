// Package repository implements data access for the audit service.
// The audit_log table is append-only — no UPDATE or DELETE is ever issued.
// Each row contains the SHA-256 hash of the previous row, forming a hash chain
// that makes tampering detectable. Ref: PRD §FR-007.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventRecord is a single row in the audit_log table.
type EventRecord struct {
	EventID    string
	Category   int32
	Action     string
	ActorDID   string
	SubjectDID string
	ResourceID string
	ServiceID  string
	Metadata   []byte
	PrevHash   string // SHA-256 hex of previous entry
	EntryHash  string // SHA-256 hex of this entry
	Timestamp  time.Time
}

// ErrNotFound is returned when an event ID is not in the log.
var ErrNotFound = errors.New("repository: audit event not found")

// Repository provides append-only access to the audit log.
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

// Append inserts a new audit event. The caller is responsible for computing
// prev_hash and entry_hash before calling this function.
func (r *Repository) Append(ctx context.Context, rec EventRecord) error {
	q := `
		INSERT INTO audit_log
			(event_id, category, action, actor_did, subject_did, resource_id,
			 service_id, metadata, prev_hash, entry_hash, timestamp)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.EventID, rec.Category, rec.Action, rec.ActorDID, rec.SubjectDID,
		rec.ResourceID, rec.ServiceID, rec.Metadata, rec.PrevHash, rec.EntryHash,
		rec.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("repository: append audit event: %w", err)
	}
	return nil
}

// GetByID retrieves a single event by its ID.
func (r *Repository) GetByID(ctx context.Context, eventID string) (*EventRecord, error) {
	q := `
		SELECT event_id, category, action, actor_did, subject_did, resource_id,
		       service_id, metadata, prev_hash, entry_hash, timestamp
		FROM audit_log WHERE event_id = $1
	`
	row := r.pool.QueryRow(ctx, q, eventID)
	var rec EventRecord
	err := row.Scan(
		&rec.EventID, &rec.Category, &rec.Action, &rec.ActorDID, &rec.SubjectDID,
		&rec.ResourceID, &rec.ServiceID, &rec.Metadata, &rec.PrevHash, &rec.EntryHash,
		&rec.Timestamp,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get audit event: %w", err)
	}
	return &rec, nil
}

// LatestHash returns the entry_hash of the most recent row, or "" if the log is empty.
// Used when building the hash chain for the next Append call.
func (r *Repository) LatestHash(ctx context.Context) (string, error) {
	q := `SELECT COALESCE(entry_hash,'') FROM audit_log ORDER BY timestamp DESC LIMIT 1`
	var h string
	err := r.pool.QueryRow(ctx, q).Scan(&h)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("repository: latest hash: %w", err)
	}
	return h, nil
}

// Query returns up to limit events matching optional filters (all params optional).
func (r *Repository) Query(ctx context.Context, actorDID, subjectDID string, category int32, from, to time.Time, limit int32, afterID string) ([]EventRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	q := `
		SELECT event_id, category, action, actor_did, subject_did, resource_id,
		       service_id, metadata, prev_hash, entry_hash, timestamp
		FROM audit_log
		WHERE ($1 = '' OR actor_did = $1)
		  AND ($2 = '' OR subject_did = $2)
		  AND ($3 = 0  OR category = $3)
		  AND ($4 = '0001-01-01'::timestamptz OR timestamp >= $4)
		  AND ($5 = '0001-01-01'::timestamptz OR timestamp <= $5)
		  AND ($6 = '' OR event_id > $6)
		ORDER BY timestamp ASC
		LIMIT $7
	`
	var zeroTime time.Time
	if from.IsZero() {
		from = zeroTime
	}
	if to.IsZero() {
		to = zeroTime
	}
	rows, err := r.pool.Query(ctx, q, actorDID, subjectDID, category, from, to, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("repository: query audit log: %w", err)
	}
	defer rows.Close()

	var recs []EventRecord
	for rows.Next() {
		var rec EventRecord
		if err = rows.Scan(
			&rec.EventID, &rec.Category, &rec.Action, &rec.ActorDID, &rec.SubjectDID,
			&rec.ResourceID, &rec.ServiceID, &rec.Metadata, &rec.PrevHash, &rec.EntryHash,
			&rec.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("repository: scan audit row: %w", err)
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}
