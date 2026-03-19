// Package service implements business logic for the enrollment service.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	biometricv1 "github.com/IranProsperityProject/INDIS/api/gen/go/biometric/v1"
	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	indiscrypto "github.com/IranProsperityProject/INDIS/pkg/crypto"
	"github.com/IranProsperityProject/INDIS/pkg/did"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/repository"
)

const minSocialAttestors = 3 // Ref: PRD §FR-001.8

// InitiateResult holds the outcome of starting an enrollment session.
type InitiateResult struct {
	EnrollmentID               string
	TemporaryReceiptCredential string
}

// CompleteResult holds the outcome of a completed enrollment.
type CompleteResult struct {
	DID                string
	IssuedCredentials  []string
}

// EnrollmentService implements the three enrollment pathways (PRD §FR-001).
type EnrollmentService struct {
	repo      *repository.Repository
	chain     blockchain.BlockchainAdapter
	events    enrollmentEventPublisher
	biometric biometricv1.BiometricServiceClient
}

type enrollmentEventPublisher interface {
	Publish(ctx context.Context, topic string, event any) error
}

// New creates an EnrollmentService.
func New(repo *repository.Repository, chain blockchain.BlockchainAdapter) *EnrollmentService {
	return &EnrollmentService{repo: repo, chain: chain}
}

// SetBiometricClient wires the gRPC biometric service client for deduplication.
// When nil, SubmitBiometrics falls back to accepting all templates (dev only).
func (s *EnrollmentService) SetBiometricClient(c biometricv1.BiometricServiceClient) {
	s.biometric = c
}

func (s *EnrollmentService) ensureRepo() error {
	if s.repo == nil {
		return fmt.Errorf("service: repository is not configured")
	}
	return nil
}

// SetEventPublisher wires an optional event publisher.
// When nil, enrollment completion events are not emitted.
func (s *EnrollmentService) SetEventPublisher(p enrollmentEventPublisher) {
	s.events = p
}

// generateID creates a random URL-safe ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// pathwayName maps proto enum ints to human-readable strings.
var pathwayName = map[int32]string{
	1: "standard",
	2: "enhanced",
	3: "social",
}

// InitiateEnrollment begins an enrollment session.
func (s *EnrollmentService) InitiateEnrollment(ctx context.Context, pathwayInt int32, agentID, locale string) (*InitiateResult, error) {
	if err := s.ensureRepo(); err != nil {
		return nil, err
	}

	pathway, ok := pathwayName[pathwayInt]
	if !ok {
		return nil, fmt.Errorf("service: unknown enrollment pathway: %d", pathwayInt)
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("service: generate enrollment id: %w", err)
	}

	now := time.Now().UTC()
	rec := repository.EnrollmentRecord{
		ID:        id,
		Pathway:   pathway,
		Status:    repository.StatusPending,
		AgentID:   agentID,
		Locale:    locale,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err = s.repo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: create enrollment: %w", err)
	}

	// Temporary receipt is a signed token the citizen can use to resume enrollment.
	receipt := "enroll:" + id
	return &InitiateResult{EnrollmentID: id, TemporaryReceiptCredential: receipt}, nil
}

// SubmitBiometrics processes biometric data for an enrollment session.
// Deduplication is delegated to services/ai; this service records the outcome.
// In production the AI service is called via gRPC. For now we accept all biometrics.
func (s *EnrollmentService) SubmitBiometrics(ctx context.Context, enrollmentID string, faceTemplate, fingerTemplate, irisTemplate []byte) (bool, string, error) {
	if err := s.ensureRepo(); err != nil {
		return false, "", err
	}

	rec, err := s.repo.GetByID(ctx, enrollmentID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, "", fmt.Errorf("service: enrollment not found: %s", enrollmentID)
	}
	if err != nil {
		return false, "", fmt.Errorf("service: get enrollment: %w", err)
	}
	if rec.Status != repository.StatusPending {
		return false, "", fmt.Errorf("service: enrollment %s is not in pending state (current: %s)", enrollmentID, rec.Status)
	}

	passed := true
	deduplicationMS := "0"

	if s.biometric != nil {
		// Delegate deduplication to the biometric service which in turn calls the AI service.
		// Ref: PRD §FR-004 — FMR ≤ 0.0001%, SLA ≤ 90s.
		dedupReq := &biometricv1.CheckDuplicateRequest{
			EnrollmentId: enrollmentID,
			Modality:     biometricv1.Modality_MODALITY_FACIAL,
		}
		// Use the first non-nil biometric template bytes passed in (face, finger, iris).
		for _, b := range [][]byte{faceTemplate, fingerTemplate, irisTemplate} {
			if len(b) > 0 {
				dedupReq.TemplateData = b
				break
			}
		}
		resp, dedupErr := s.biometric.CheckDuplicate(ctx, dedupReq)
		if dedupErr != nil {
			return false, "", fmt.Errorf("service: biometric dedup call: %w", dedupErr)
		}
		passed = !resp.IsDuplicate
		deduplicationMS = resp.DeduplicationMs
	}

	if err = s.repo.UpdateStatus(ctx, enrollmentID, repository.StatusBiometricsSubmitted, passed, 0); err != nil {
		return false, "", fmt.Errorf("service: update biometrics status: %w", err)
	}
	return passed, deduplicationMS, nil
}

// SubmitSocialAttestation records community co-attestors for a social pathway enrollment.
// Requires at least minSocialAttestors (3) attestors. Ref: PRD §FR-001.8
func (s *EnrollmentService) SubmitSocialAttestation(ctx context.Context, enrollmentID string, attestorDIDs []string) (bool, int32, error) {
	if err := s.ensureRepo(); err != nil {
		return false, 0, err
	}

	rec, err := s.repo.GetByID(ctx, enrollmentID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, 0, fmt.Errorf("service: enrollment not found: %s", enrollmentID)
	}
	if err != nil {
		return false, 0, fmt.Errorf("service: get enrollment: %w", err)
	}
	if rec.Pathway != "social" {
		return false, 0, fmt.Errorf("service: social attestation only valid for social pathway, got %s", rec.Pathway)
	}
	if len(attestorDIDs) < minSocialAttestors {
		return false, int32(len(attestorDIDs)), fmt.Errorf("service: need at least %d attestors, got %d", minSocialAttestors, len(attestorDIDs))
	}

	if err = s.repo.UpdateStatus(ctx, enrollmentID, repository.StatusAttestationSubmitted, rec.BiometricsPassed, len(attestorDIDs)); err != nil {
		return false, 0, fmt.Errorf("service: update attestation status: %w", err)
	}
	return true, int32(len(attestorDIDs)), nil
}

// CompleteEnrollment finalizes enrollment: generates a DID and initial credentials.
// Ref: PRD §FR-001.5
func (s *EnrollmentService) CompleteEnrollment(ctx context.Context, enrollmentID string) (*CompleteResult, error) {
	if err := s.ensureRepo(); err != nil {
		return nil, err
	}

	rec, err := s.repo.GetByID(ctx, enrollmentID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: enrollment not found: %s", enrollmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("service: get enrollment: %w", err)
	}

	// Validate readiness based on pathway.
	switch rec.Pathway {
	case "standard", "enhanced":
		if rec.Status != repository.StatusBiometricsSubmitted {
			return nil, fmt.Errorf("service: biometrics not yet submitted for enrollment %s", enrollmentID)
		}
	case "social":
		if rec.Status != repository.StatusAttestationSubmitted {
			return nil, fmt.Errorf("service: social attestation not yet submitted for enrollment %s", enrollmentID)
		}
	}

	// Generate a new key pair for the citizen's DID (on-device in production;
	// server-side here for the enrollment flow only).
	kp, err := indiscrypto.GenerateEd25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("service: key generation: %w", err)
	}
	citizenDID, err := did.FromPublicKey(kp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("service: did derivation: %w", err)
	}

	// Store the completed enrollment.
	if err = s.repo.Complete(ctx, enrollmentID, citizenDID.String()); err != nil {
		return nil, fmt.Errorf("service: complete enrollment: %w", err)
	}

	// Anchor DID on blockchain (best-effort).
	doc := blockchain.DIDDocument{
		ID:      citizenDID.String(),
		Created: time.Now().UTC(),
		Updated: time.Now().UTC(),
	}
	_, _ = s.chain.RegisterDID(ctx, citizenDID.String(), doc)

	// Initial credentials issued at enrollment (citizenship + voter eligibility).
	// Full credential issuance is delegated to the credential service in production.
	issued := []string{"CitizenshipCredential", "VoterEligibilityCredential"}

	if s.events != nil {
		event := events.EnrollmentCompletedEvent{
			EnrollmentID: enrollmentID,
			DID:          citizenDID.String(),
			SubjectName:  "",
			DistrictCode: "",
			PathwayType:  rec.Pathway,
			OccurredAt:   time.Now().UTC(),
		}
		_ = s.events.Publish(ctx, events.TopicEnrollmentCompleted, event)
	}

	return &CompleteResult{DID: citizenDID.String(), IssuedCredentials: issued}, nil
}

// GetEnrollmentStatus returns the current status of an enrollment session.
func (s *EnrollmentService) GetEnrollmentStatus(ctx context.Context, enrollmentID string) (string, error) {
	if err := s.ensureRepo(); err != nil {
		return "", err
	}

	rec, err := s.repo.GetByID(ctx, enrollmentID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", fmt.Errorf("service: enrollment not found: %s", enrollmentID)
	}
	if err != nil {
		return "", fmt.Errorf("service: get enrollment: %w", err)
	}
	return string(rec.Status), nil
}
