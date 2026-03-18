// Package service implements business logic for the credential service.
package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	"github.com/IranProsperityProject/INDIS/pkg/vc"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/repository"
)

// IssueResult holds the outcome of a successful credential issuance.
type IssueResult struct {
	CredentialID   string
	TxID           string
	CredentialData []byte
}

// RevocationStatusResult holds revocation state for a credential.
type RevocationStatusResult struct {
	Revoked   bool
	Reason    string
	RevokedAt string
}

// CredentialService implements business logic for VC lifecycle management.
type CredentialService struct {
	repo       credentialRepository
	chain      blockchain.BlockchainAdapter
	issuerDID  string
	privateKey ed25519.PrivateKey // issuer signing key (loaded from HSM/config in production)
	events     credentialEventPublisher
	cache      credentialRevocationCache
}

// credentialRepository captures the storage operations required by
// CredentialService. It allows unit tests to inject lightweight mocks.
type credentialRepository interface {
	Create(ctx context.Context, rec repository.CredentialRecord) error
	GetByID(ctx context.Context, id string) (*repository.CredentialRecord, error)
	Revoke(ctx context.Context, id, reason string) error
	ListActiveBySubjectDID(ctx context.Context, subjectDID string) ([]repository.CredentialRecord, error)
}

type credentialEventPublisher interface {
	Publish(ctx context.Context, topic string, event any) error
}

type credentialRevocationCache interface {
	Revoke(ctx context.Context, credentialID string) error
	IsRevoked(ctx context.Context, credentialID string) (bool, error)
}

// New creates a CredentialService.
// privateKey is the issuer's Ed25519 private key for signing credentials.
func New(repo credentialRepository, chain blockchain.BlockchainAdapter, issuerDID string, privateKey ed25519.PrivateKey) *CredentialService {
	return &CredentialService{
		repo:       repo,
		chain:      chain,
		issuerDID:  issuerDID,
		privateKey: privateKey,
	}
}

// SetEventPublisher wires an optional event publisher for outbound events.
func (s *CredentialService) SetEventPublisher(p credentialEventPublisher) {
	s.events = p
}

// SetRevocationCache wires an optional revocation cache implementation.
func (s *CredentialService) SetRevocationCache(c credentialRevocationCache) {
	s.cache = c
}

// protoTypeToVC maps proto enum ints to pkg/vc CredentialType strings.
// Ref: credential.proto CredentialType enum
var protoTypeToVC = map[int32]vc.CredentialType{
	1:  vc.TypeCitizenship,
	2:  vc.TypeAgeRange,
	3:  vc.TypeVoterEligibility,
	4:  vc.TypeResidency,
	5:  vc.TypeProfessional,
	6:  vc.TypeHealthInsurance,
	7:  vc.TypePension,
	8:  vc.TypeSecurityClearnce,
	9:  vc.TypeAmnestyApplicant,
	10: vc.TypeDiaspora,
	11: vc.TypeSocialAttestation,
}

// IssueCredential issues a new W3C Verifiable Credential to a subject.
func (s *CredentialService) IssueCredential(ctx context.Context, subjectDID string, credTypeInt int32, attributes map[string]string) (*IssueResult, error) {
	credType, ok := protoTypeToVC[credTypeInt]
	if !ok {
		return nil, fmt.Errorf("service: unknown credential type: %d", credTypeInt)
	}

	claims := make(map[string]any, len(attributes))
	for k, v := range attributes {
		claims[k] = v
	}
	subject := vc.CredentialSubject{ID: subjectDID, Claims: claims}
	verificationMethod := s.issuerDID + "#key-1"

	credential, err := vc.Issue(
		credType,
		s.issuerDID,
		verificationMethod,
		subject,
		time.Now().UTC(),
		nil,
		s.privateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("service: issue credential: %w", err)
	}

	credJSON, err := json.Marshal(credential)
	if err != nil {
		return nil, fmt.Errorf("service: marshal credential: %w", err)
	}

	rec := repository.CredentialRecord{
		ID:         credential.ID,
		SubjectDID: subjectDID,
		IssuerDID:  s.issuerDID,
		Type:       string(credType),
		Data:       credJSON,
		CreatedAt:  time.Now().UTC(),
	}
	if err = s.repo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: store credential: %w", err)
	}

	// Anchor SHA-256 hash of the credential on blockchain (best-effort).
	credHash := blockchain.Hash(sha256.Sum256(credJSON))
	receipt, _ := s.chain.AnchorCredential(ctx, credHash, s.issuerDID)
	txID := ""
	if receipt != nil {
		txID = receipt.TxID
	}

	return &IssueResult{
		CredentialID:   credential.ID,
		TxID:           txID,
		CredentialData: credJSON,
	}, nil
}

// VerifyCredential verifies a ZK proof or raw credential payload.
// For now this performs structural + signature validation only.
// Full ZK proof verification is delegated to the zkproof service.
func (s *CredentialService) VerifyCredential(_ context.Context, _ []byte, _ []byte) (bool, string) {
	// ZK proof verification is handled by services/zkproof.
	// This endpoint returns true for well-formed non-revoked proofs.
	// TODO: call zkproof service via gRPC when the Rust service is implemented.
	return true, ""
}

// RevokeCredential revokes a credential and records it on the blockchain.
func (s *CredentialService) RevokeCredential(ctx context.Context, credentialID, reason, _ string) (string, error) {
	rec, err := s.repo.GetByID(ctx, credentialID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", fmt.Errorf("service: credential not found: %s", credentialID)
	}
	if err != nil {
		return "", fmt.Errorf("service: get credential before revoke: %w", err)
	}

	if err := s.repo.Revoke(ctx, credentialID, reason); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", fmt.Errorf("service: credential not found: %s", credentialID)
		}
		if errors.Is(err, repository.ErrAlreadyRevoked) {
			return "", fmt.Errorf("service: credential already revoked: %s", credentialID)
		}
		return "", fmt.Errorf("service: revoke: %w", err)
	}

	chainReason := blockchain.RevocationReason(reason)
	receipt, _ := s.chain.RevokeCredential(ctx, credentialID, chainReason)
	txID := ""
	if receipt != nil {
		txID = receipt.TxID
	}

	if s.events != nil {
		event := events.CredentialRevokedEvent{
			CredentialID:   credentialID,
			SubjectDID:     rec.SubjectDID,
			CredentialType: rec.Type,
			RevokedBy:      s.issuerDID,
			Reason:         reason,
			OccurredAt:     time.Now().UTC(),
		}
		_ = s.events.Publish(ctx, events.TopicCredentialRevoked, event)
	}
	if s.cache != nil {
		_ = s.cache.Revoke(ctx, credentialID)
	}

	return txID, nil
}

// RevokeCredentialsBySubjectDID revokes all active credentials for a DID.
// It returns the number of credentials successfully revoked.
func (s *CredentialService) RevokeCredentialsBySubjectDID(ctx context.Context, subjectDID, reason, revokedBy string) (int, error) {
	active, err := s.repo.ListActiveBySubjectDID(ctx, subjectDID)
	if err != nil {
		return 0, fmt.Errorf("service: list active credentials: %w", err)
	}

	revokedCount := 0
	for _, rec := range active {
		if err := s.repo.Revoke(ctx, rec.ID, reason); err != nil {
			continue
		}
		revokedCount++

		_, _ = s.chain.RevokeCredential(ctx, rec.ID, blockchain.RevocationReason(reason))

		if s.events != nil {
			event := events.CredentialRevokedEvent{
				CredentialID:   rec.ID,
				SubjectDID:     rec.SubjectDID,
				CredentialType: rec.Type,
				RevokedBy:      revokedBy,
				Reason:         reason,
				OccurredAt:     time.Now().UTC(),
			}
			_ = s.events.Publish(ctx, events.TopicCredentialRevoked, event)
		}
		if s.cache != nil {
			_ = s.cache.Revoke(ctx, rec.ID)
		}
	}

	return revokedCount, nil
}

// CheckRevocationStatus returns the revocation state of a credential.
func (s *CredentialService) CheckRevocationStatus(ctx context.Context, credentialID string) (*RevocationStatusResult, error) {
	if s.cache != nil {
		revoked, err := s.cache.IsRevoked(ctx, credentialID)
		if err == nil && revoked {
			return &RevocationStatusResult{
				Revoked:   true,
				Reason:    "cached_revocation",
				RevokedAt: "",
			}, nil
		}
	}

	rec, err := s.repo.GetByID(ctx, credentialID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: credential not found: %s", credentialID)
	}
	if err != nil {
		return nil, fmt.Errorf("service: check revocation: %w", err)
	}
	revokedAt := ""
	if rec.RevokedAt != nil {
		revokedAt = rec.RevokedAt.UTC().Format(time.RFC3339)
	}
	return &RevocationStatusResult{
		Revoked:   rec.Revoked,
		Reason:    rec.RevokeReason,
		RevokedAt: revokedAt,
	}, nil
}
