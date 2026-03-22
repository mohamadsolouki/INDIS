// Package service implements business logic for the notification service.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	notificationv1 "github.com/mohamadsolouki/INDIS/api/gen/go/notification/v1"
	"github.com/mohamadsolouki/INDIS/services/notification/internal/repository"
)

// NotificationService handles multi-channel notification delivery.
type NotificationService struct {
	repo *repository.Repository
}

// New creates a NotificationService.
func New(repo *repository.Repository) *NotificationService {
	return &NotificationService{repo: repo}
}

func generateID(prefix string) (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// Send dispatches a notification immediately (stores in DB; actual delivery is async).
// In production an external dispatcher worker reads queued rows and calls SMS/Push/Email APIs.
func (s *NotificationService) Send(ctx context.Context, req *notificationv1.SendRequest) (string, string, error) {
	id, err := generateID("ntf_")
	if err != nil {
		return "", "", fmt.Errorf("service: generate id: %w", err)
	}
	now := time.Now().UTC()
	rec := repository.NotificationRecord{
		ID:           id,
		RecipientDID: req.GetRecipientDid(),
		Channel:      int32(req.GetChannel()),
		Type:         int32(req.GetType()),
		Locale:       req.GetLocale(),
		Subject:      req.GetSubject(),
		Body:         req.GetBody(),
		Status:       "queued",
		CreatedAt:    now,
	}
	if err = s.repo.Create(ctx, rec); err != nil {
		return "", "", fmt.Errorf("service: create notification: %w", err)
	}
	return id, "queued", nil
}

// ScheduleExpiryAlert schedules three reminders (30d / 7d / 1d) before credential expiry.
// Ref: PRD §FR-002.R4
func (s *NotificationService) ScheduleExpiryAlert(ctx context.Context, req *notificationv1.ScheduleExpiryAlertRequest) ([]string, error) {
	expiresAt, err := time.Parse(time.RFC3339, req.GetExpiresAt())
	if err != nil {
		return nil, fmt.Errorf("service: parse expires_at: %w", err)
	}

	offsets := []time.Duration{30 * 24 * time.Hour, 7 * 24 * time.Hour, 24 * time.Hour}
	var ids []string
	for _, offset := range offsets {
		scheduledAt := expiresAt.Add(-offset)
		if scheduledAt.Before(time.Now().UTC()) {
			continue // skip past alerts
		}
		id, err := generateID("alr_")
		if err != nil {
			return nil, fmt.Errorf("service: generate alert id: %w", err)
		}
		days := int(offset.Hours() / 24)
		rec := repository.NotificationRecord{
			ID:           id,
			RecipientDID: req.GetRecipientDid(),
			Channel:      int32(notificationv1.Channel_CHANNEL_PUSH),
			Type:         int32(notificationv1.NotificationType_NOTIFICATION_TYPE_CREDENTIAL_EXPIRY),
			Locale:       req.GetLocale(),
			Subject:      fmt.Sprintf("Credential expiring in %d days", days),
			Body:         fmt.Sprintf("Your %s credential expires in %d days.", req.GetCredentialType(), days),
			Status:       "queued",
			ScheduledAt:  &scheduledAt,
			CreatedAt:    time.Now().UTC(),
		}
		if err = s.repo.Create(ctx, rec); err != nil {
			return nil, fmt.Errorf("service: schedule alert: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// CancelAlert cancels a previously scheduled alert.
func (s *NotificationService) CancelAlert(ctx context.Context, alertID string) (bool, error) {
	if err := s.repo.Cancel(ctx, alertID); err != nil {
		return false, fmt.Errorf("service: cancel alert: %w", err)
	}
	return true, nil
}

// Dispatcher is the interface subset the dispatcher needs from the repository.
type Dispatcher interface {
	GetDueForDispatch(ctx context.Context, limit int) ([]repository.NotificationRecord, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, reason string) error
}

// RunDispatcher starts a background loop that delivers queued notifications.
// It polls at the given interval and dispatches up to 100 notifications per tick.
// In production, replace the delivery stubs with real SMS/push/email API calls.
// Ref: PRD §FR-002.R4 — 3-tier expiry alerts must actually reach citizens.
func (s *NotificationService) RunDispatcher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Printf("notification dispatcher started (interval=%s)", interval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("notification dispatcher stopped")
			return
		case <-ticker.C:
			s.dispatchBatch(ctx)
		}
	}
}

func (s *NotificationService) dispatchBatch(ctx context.Context) {
	recs, err := s.repo.GetDueForDispatch(ctx, 100)
	if err != nil {
		log.Printf("notification dispatcher: fetch error: %v", err)
		return
	}
	for _, rec := range recs {
		if err := s.deliver(ctx, rec); err != nil {
			log.Printf("notification dispatcher: delivery failed id=%s: %v", rec.ID, err)
			_ = s.repo.MarkFailed(ctx, rec.ID, err.Error())
		} else {
			_ = s.repo.MarkDelivered(ctx, rec.ID)
		}
	}
}

// deliver dispatches a single notification via the appropriate channel.
// Production: replace each case with a real provider API call (Infobip, Firebase, SMTP).
func (s *NotificationService) deliver(ctx context.Context, rec repository.NotificationRecord) error {
	_ = ctx
	switch rec.Channel {
	case 1: // SMS
		// TODO(production): call national telecom SMS API (Infobip, MCI, MTN).
		log.Printf("notification[SMS] to=%s subject=%q id=%s", rec.RecipientDID, rec.Subject, rec.ID)
	case 2: // Push
		// TODO(production): call Firebase FCM or self-hosted push service.
		log.Printf("notification[PUSH] to=%s subject=%q id=%s", rec.RecipientDID, rec.Subject, rec.ID)
	case 3: // Email
		// TODO(production): call SMTP relay or transactional email service.
		log.Printf("notification[EMAIL] to=%s subject=%q id=%s", rec.RecipientDID, rec.Subject, rec.ID)
	default:
		log.Printf("notification[UNKNOWN channel=%d] to=%s id=%s", rec.Channel, rec.RecipientDID, rec.ID)
	}
	return nil
}
