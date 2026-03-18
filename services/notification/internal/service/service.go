// Package service implements business logic for the notification service.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	notificationv1 "github.com/IranProsperityProject/INDIS/api/gen/go/notification/v1"
	"github.com/IranProsperityProject/INDIS/services/notification/internal/repository"
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
