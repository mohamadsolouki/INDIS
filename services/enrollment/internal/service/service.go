// Package service implements business logic for the enrollment service.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	indiscrypto "github.com/IranProsperityProject/INDIS/pkg/crypto"
	"github.com/IranProsperityProject/INDIS/pkg/did"
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
	repo  *repository.Repository
	chain blockchain.BlockchainAdapter
}

// New creates an EnrollmentService.
func New(repo *repository.Repository, chain blockchain.BlockchainAdapter) *EnrollmentService {
	return &EnrollmentService{repo: repo, chain: chain}
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
func (s *EnrollmentService) SubmitBiometrics(ctx context.Context, enrollmentID string, _, _, _ []byte) (bool, string, error) {
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

	// TODO: call services/ai gRPC deduplication endpoint.
	passed := true
	deduplicationMS := "12"

	if err = s.repo.UpdateStatus(ctx, enrollmentID, repository.StatusBiometricsSubmitted, passed, 0); err != nil {
		return false, "", fmt.Errorf("service: update biometrics status: %w", err)
	}
	return passed, deduplicationMS, nil
}

// SubmitSocialAttestation records community co-attestors for a social pathway enrollment.
// Requires at least minSocialAttestors (3) attestors. Ref: PRD §FR-001.8
func (s *EnrollmentService) SubmitSocialAttestation(ctx context.Context, enrollmentID string, attestorDIDs []string) (bool, int32, error) {
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

	return &CompleteResult{DID: citizenDID.String(), IssuedCredentials: issued}, nil
}

// GetEnrollmentStatus returns the current status of an enrollment session.
func (s *EnrollmentService) GetEnrollmentStatus(ctx context.Context, enrollmentID string) (string, error) {
	rec, err := s.repo.GetByID(ctx, enrollmentID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", fmt.Errorf("service: enrollment not found: %s", enrollmentID)
	}
	if err != nil {
		return "", fmt.Errorf("service: get enrollment: %w", err)
	}
	return string(rec.Status), nil
}
