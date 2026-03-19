// Package repository implements data access for the USSD/SMS gateway service.
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

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("repository: record not found")

// USSDSession represents a row in the ussd_sessions table.
type USSDSession struct {
	SessionID       string
	PhoneNumberHash string
	ServiceCode     string
	CurrentStep     int
	FlowType        string // "voter" | "pension" | "credential"
	Locale          string
	StateData       map[string]string
	StartedAt       time.Time
	LastActiveAt    time.Time
	EndedAt         *time.Time
}

// SMSOtp represents a row in the sms_otps table.
type SMSOtp struct {
	ID              string
	PhoneNumberHash string
	OTPHash         string
	ExpiresAt       time.Time
	Used            bool
	CreatedAt       time.Time
}

// Repository provides access to USSD and OTP tables.
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

// CreateSession inserts a new USSD session.
func (r *Repository) CreateSession(ctx context.Context, s USSDSession) error {
	raw, err := json.Marshal(s.StateData)
	if err != nil {
		return fmt.Errorf("repository: marshal state_data: %w", err)
	}
	q := `
		INSERT INTO ussd_sessions
			(session_id, phone_number_hash, service_code, current_step, flow_type, locale, state_data, started_at, last_active_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`
	_, err = r.pool.Exec(ctx, q,
		s.SessionID, s.PhoneNumberHash, s.ServiceCode, s.CurrentStep,
		s.FlowType, s.Locale, raw, s.StartedAt, s.LastActiveAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create session: %w", err)
	}
	return nil
}

// GetSession fetches an active USSD session by session ID.
func (r *Repository) GetSession(ctx context.Context, sessionID string) (*USSDSession, error) {
	q := `
		SELECT session_id, phone_number_hash, service_code, current_step, flow_type, locale,
		       state_data, started_at, last_active_at, ended_at
		FROM ussd_sessions WHERE session_id = $1
	`
	row := r.pool.QueryRow(ctx, q, sessionID)
	var s USSDSession
	var rawState []byte
	err := row.Scan(
		&s.SessionID, &s.PhoneNumberHash, &s.ServiceCode, &s.CurrentStep,
		&s.FlowType, &s.Locale, &rawState, &s.StartedAt, &s.LastActiveAt, &s.EndedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get session: %w", err)
	}
	if err = json.Unmarshal(rawState, &s.StateData); err != nil {
		return nil, fmt.Errorf("repository: unmarshal state_data: %w", err)
	}
	return &s, nil
}

// UpdateSession updates the step, locale, state_data and last_active_at for an existing session.
func (r *Repository) UpdateSession(ctx context.Context, s USSDSession) error {
	raw, err := json.Marshal(s.StateData)
	if err != nil {
		return fmt.Errorf("repository: marshal state_data: %w", err)
	}
	q := `
		UPDATE ussd_sessions
		SET current_step = $2, locale = $3, state_data = $4, last_active_at = $5
		WHERE session_id = $1
	`
	tag, err := r.pool.Exec(ctx, q,
		s.SessionID, s.CurrentStep, s.Locale, raw, s.LastActiveAt,
	)
	if err != nil {
		return fmt.Errorf("repository: update session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EndSession marks a session as ended and wipes PII state_data per FR-015.6.
func (r *Repository) EndSession(ctx context.Context, sessionID string) error {
	now := time.Now().UTC()
	q := `
		UPDATE ussd_sessions
		SET ended_at = $2, state_data = '{}'::jsonb, last_active_at = $2
		WHERE session_id = $1 AND ended_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, q, sessionID, now)
	if err != nil {
		return fmt.Errorf("repository: end session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateOTP inserts a new SMS OTP record.
func (r *Repository) CreateOTP(ctx context.Context, otp SMSOtp) error {
	q := `
		INSERT INTO sms_otps (id, phone_number_hash, otp_hash, expires_at, used, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`
	_, err := r.pool.Exec(ctx, q,
		otp.ID, otp.PhoneNumberHash, otp.OTPHash, otp.ExpiresAt, otp.Used, otp.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create otp: %w", err)
	}
	return nil
}

// GetActiveOTP returns the most recent unexpired, unused OTP for a phone hash.
func (r *Repository) GetActiveOTP(ctx context.Context, phoneHash string) (*SMSOtp, error) {
	q := `
		SELECT id, phone_number_hash, otp_hash, expires_at, used, created_at
		FROM sms_otps
		WHERE phone_number_hash = $1
		  AND used = FALSE
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, q, phoneHash)
	var o SMSOtp
	err := row.Scan(&o.ID, &o.PhoneNumberHash, &o.OTPHash, &o.ExpiresAt, &o.Used, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get active otp: %w", err)
	}
	return &o, nil
}

// MarkOTPUsed marks an OTP record as consumed.
func (r *Repository) MarkOTPUsed(ctx context.Context, id string) error {
	q := `UPDATE sms_otps SET used = TRUE WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("repository: mark otp used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
