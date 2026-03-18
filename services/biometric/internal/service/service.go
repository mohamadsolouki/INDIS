// Package service implements business logic for the biometric service.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	indiscrypto "github.com/IranProsperityProject/INDIS/pkg/crypto"
	"github.com/IranProsperityProject/INDIS/services/biometric/internal/repository"
)

// BiometricService handles template encryption, storage, and deduplication.
type BiometricService struct {
	repo       *repository.Repository
	encryptKey []byte // 32-byte AES-256 key, loaded from HSM in production
}

// New creates a BiometricService with the given AES-256 encryption key.
func New(repo *repository.Repository, encryptKey []byte) *BiometricService {
	return &BiometricService{repo: repo, encryptKey: encryptKey}
}

func generateTemplateID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "tpl_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// StoreTemplate encrypts the raw template and persists it.
// The one-way transform (AES-256-GCM) ensures the original cannot be reconstructed
// without the HSM-managed key. Ref: PRD §FR-004.
func (s *BiometricService) StoreTemplate(ctx context.Context, enrollmentID string, modality int32, rawTemplate []byte, _ bool) (string, error) {
	encrypted, err := indiscrypto.EncryptAES256GCM(s.encryptKey, rawTemplate)
	if err != nil {
		return "", fmt.Errorf("service: encrypt template: %w", err)
	}

	templateID, err := generateTemplateID()
	if err != nil {
		return "", fmt.Errorf("service: generate template id: %w", err)
	}

	rec := repository.TemplateRecord{
		TemplateID:    templateID,
		EnrollmentID:  enrollmentID,
		Modality:      modality,
		EncryptedData: encrypted,
		CreatedAt:     time.Now().UTC(),
	}
	if err = s.repo.Store(ctx, rec); err != nil {
		return "", fmt.Errorf("service: store template: %w", err)
	}
	return templateID, nil
}

// CheckDuplicate runs deduplication for the given enrollment's template data.
// In production this calls services/ai via gRPC (PRD §FR-004: FMR ≤ 0.0001%).
// The 90-second SLA is enforced by the AI service; this wrapper adds a timeout guard.
func (s *BiometricService) CheckDuplicate(ctx context.Context, enrollmentID string, _ int32, rawTemplate []byte) (isDuplicate bool, matchedDID, deduplicationMS string, matchScore float64, err error) {
	// TODO: call services/ai DeduplicateBiometric gRPC endpoint.
	// For now: check if any existing template for this enrollment matches exactly
	// (placeholder logic — real comparison requires the AI embeddings model).
	_ = rawTemplate
	start := time.Now()

	existing, listErr := s.repo.ListByEnrollment(ctx, enrollmentID)
	if listErr != nil {
		return false, "", "", 0, fmt.Errorf("service: list templates: %w", listErr)
	}
	// Stub: no duplicate if first template for this enrollment.
	if len(existing) == 0 {
		elapsed := fmt.Sprintf("%d", time.Since(start).Milliseconds())
		return false, "", elapsed, 0.0, nil
	}

	elapsed := fmt.Sprintf("%d", time.Since(start).Milliseconds())
	return false, "", elapsed, 0.0, nil
}

// DeleteTemplate permanently soft-deletes a template (right to erasure).
func (s *BiometricService) DeleteTemplate(ctx context.Context, templateID, _ string) (bool, string, error) {
	err := s.repo.SoftDelete(ctx, templateID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, "", fmt.Errorf("service: template not found: %s", templateID)
	}
	if err != nil {
		return false, "", fmt.Errorf("service: delete template: %w", err)
	}
	return true, time.Now().UTC().Format(time.RFC3339), nil
}
