// Package repository implements data access for the notification service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRecord is a row in the notifications table.
type NotificationRecord struct {
	ID           string
	RecipientDID string
	Channel      int32
	Type         int32
	Locale       string
	Subject      string
	Body         string
	Status       string // queued | delivered | failed
	ScheduledAt  *time.Time
	CreatedAt    time.Time
}

// ErrNotFound is returned when an alert ID is not in the database.
var ErrNotFound = errors.New("repository: notification not found")

// Repository provides access to the notifications table.
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

// Create inserts a new notification record.
func (r *Repository) Create(ctx context.Context, rec NotificationRecord) error {
	q := `
		INSERT INTO notifications (id, recipient_did, channel, type, locale, subject, body, status, scheduled_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`
	_, err := r.pool.Exec(ctx, q,
		rec.ID, rec.RecipientDID, rec.Channel, rec.Type, rec.Locale,
		rec.Subject, rec.Body, rec.Status, rec.ScheduledAt, rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create notification: %w", err)
	}
	return nil
}

// Cancel marks a notification as cancelled so the dispatcher skips it.
func (r *Repository) Cancel(ctx context.Context, id string) error {
	q := `UPDATE notifications SET status = 'cancelled' WHERE id = $1 AND status = 'queued'`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("repository: cancel notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Check existence.
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT TRUE FROM notifications WHERE id = $1`, id).Scan(&exists)
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

// GetByID fetches a notification record.
func (r *Repository) GetByID(ctx context.Context, id string) (*NotificationRecord, error) {
	q := `SELECT id, recipient_did, channel, type, locale, subject, body, status, scheduled_at, created_at FROM notifications WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	var rec NotificationRecord
	err := row.Scan(&rec.ID, &rec.RecipientDID, &rec.Channel, &rec.Type, &rec.Locale,
		&rec.Subject, &rec.Body, &rec.Status, &rec.ScheduledAt, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get notification: %w", err)
	}
	return &rec, nil
}
