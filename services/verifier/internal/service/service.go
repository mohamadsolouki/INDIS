// Package service implements business logic for the verifier service.
// It handles verifier registration (PRD FR-012) and ZK-proof-based credential
// verification (PRD FR-013). Every verification attempt is persisted as an
// audit event in the verification_events table.
package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mohamadsolouki/INDIS/services/verifier/internal/repository"
)

// zkProofVerifyRequest is the JSON body sent to the zkproof HTTP /verify endpoint.
type zkProofVerifyRequest struct {
	ProofSystem    string `json:"proof_system"`
	ProofB64       string `json:"proof_b64"`
	PublicInputsB64 string `json:"public_inputs_b64"`
	CredentialType string `json:"credential_type"`
	Predicate      string `json:"predicate"`
	Nonce          string `json:"nonce"`
}

// zkProofVerifyResponse is the JSON body returned by the zkproof service.
type zkProofVerifyResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// RegisterResult is the outcome of a successful verifier registration.
type RegisterResult struct {
	VerifierID    string
	CertificateID string
	PublicKeyHex  string
}

// VerifierRepository defines the data-access behavior required by the service.
type VerifierRepository interface {
	CreateVerifier(ctx context.Context, rec repository.VerifierRecord) error
	GetVerifierByID(ctx context.Context, id string) (*repository.VerifierRecord, error)
	ListVerifiers(ctx context.Context, statusFilter string) ([]*repository.VerifierRecord, error)
	UpdateVerifierStatus(ctx context.Context, id, status string) error
	CreateVerificationEvent(ctx context.Context, evt repository.VerificationEventRecord) error
	ListVerificationEvents(ctx context.Context, verifierID string, limit int32) ([]*repository.VerificationEventRecord, error)
}

// VerifierService implements business logic for verifier management and credential verification.
type VerifierService struct {
	repo       VerifierRepository
	zkProofURL string
	httpClient *http.Client
}

// New creates a VerifierService.
func New(repo VerifierRepository, zkProofURL string) *VerifierService {
	return &VerifierService{
		repo:       repo,
		zkProofURL: zkProofURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// RegisterVerifier creates a new verifier organization record and issues an Ed25519 certificate.
// It generates a random UUID for the verifier ID and certificate ID, then stores both the
// public key and the full record in PostgreSQL.
func (s *VerifierService) RegisterVerifier(ctx context.Context, orgName, orgType string, authorizedTypes []string, geographicScope string, maxPerDay int32) (*RegisterResult, error) {
	if orgName == "" {
		return nil, fmt.Errorf("service: org_name is required")
	}
	if orgType == "" {
		return nil, fmt.Errorf("service: org_type is required")
	}

	verifierID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("service: generate verifier ID: %w", err)
	}
	certID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("service: generate certificate ID: %w", err)
	}

	// Generate Ed25519 key pair for this verifier certificate.
	// Ref: RFC 8037 — Ed25519 for signing
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("service: generate Ed25519 key: %w", err)
	}
	pubKeyHex := hex.EncodeToString(pubKey)

	if geographicScope == "" {
		geographicScope = "nationwide"
	}
	if maxPerDay <= 0 {
		maxPerDay = 10000
	}

	now := time.Now().UTC()
	rec := repository.VerifierRecord{
		ID:                       verifierID,
		OrgName:                  orgName,
		OrgType:                  orgType,
		AuthorizedCredentialTypes: authorizedTypes,
		GeographicScope:          geographicScope,
		MaxVerificationsPerDay:   maxPerDay,
		Status:                   "active",
		CertificateID:            certID,
		PublicKeyHex:             pubKeyHex,
		RegisteredAt:             now,
		UpdatedAt:                now,
	}
	if err := s.repo.CreateVerifier(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: store verifier: %w", err)
	}

	return &RegisterResult{
		VerifierID:    verifierID,
		CertificateID: certID,
		PublicKeyHex:  pubKeyHex,
	}, nil
}

// GetVerifier retrieves a verifier record by ID. Returns an error if not found.
func (s *VerifierService) GetVerifier(ctx context.Context, verifierID string) (*repository.VerifierRecord, error) {
	if verifierID == "" {
		return nil, fmt.Errorf("service: verifier_id is required")
	}
	rec, err := s.repo.GetVerifierByID(ctx, verifierID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: verifier not found: %s", verifierID)
	}
	if err != nil {
		return nil, fmt.Errorf("service: get verifier: %w", err)
	}
	return rec, nil
}

// ListVerifiers returns all verifiers, optionally filtered by status string.
func (s *VerifierService) ListVerifiers(ctx context.Context, statusFilter string) ([]*repository.VerifierRecord, error) {
	recs, err := s.repo.ListVerifiers(ctx, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("service: list verifiers: %w", err)
	}
	return recs, nil
}

// SuspendVerifier sets the verifier status to 'suspended'.
// Use SuspendVerifier with reason "revoked" in the caller to handle full revocation.
func (s *VerifierService) SuspendVerifier(ctx context.Context, verifierID, reason string) error {
	if verifierID == "" {
		return fmt.Errorf("service: verifier_id is required")
	}
	// Determine new status from reason keyword.
	newStatus := "suspended"
	if reason == "revoked" {
		newStatus = "revoked"
	}
	if err := s.repo.UpdateVerifierStatus(ctx, verifierID, newStatus); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("service: verifier not found: %s", verifierID)
		}
		return fmt.Errorf("service: suspend verifier: %w", err)
	}
	return nil
}

// VerifyCredential forwards the ZK proof to the zkproof service and logs the result.
// It returns the boolean verification outcome and a newly created event ID.
// The verifier must be active and must have the requested credential_type in its
// authorized list; otherwise the request is rejected before reaching the ZK service.
func (s *VerifierService) VerifyCredential(
	ctx context.Context,
	verifierID, credentialType, predicate, nonce, proofSystem, proofB64, publicInputsB64 string,
) (valid bool, eventID string, err error) {
	if verifierID == "" {
		return false, "", fmt.Errorf("service: verifier_id is required")
	}
	if credentialType == "" {
		return false, "", fmt.Errorf("service: credential_type is required")
	}
	if nonce == "" {
		return false, "", fmt.Errorf("service: nonce is required")
	}
	if proofB64 == "" {
		return false, "", fmt.Errorf("service: proof_b64 is required")
	}

	// Fetch verifier and enforce access control.
	rec, err := s.repo.GetVerifierByID(ctx, verifierID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, "", fmt.Errorf("service: verifier not found: %s", verifierID)
	}
	if err != nil {
		return false, "", fmt.Errorf("service: get verifier: %w", err)
	}
	if rec.Status != "active" {
		return false, "", fmt.Errorf("service: verifier %s is %s", verifierID, rec.Status)
	}
	if !containsString(rec.AuthorizedCredentialTypes, credentialType) {
		return false, "", fmt.Errorf("service: verifier %s is not authorized for credential type %s", verifierID, credentialType)
	}

	// Call the zkproof service.
	valid, err = s.callZKProof(ctx, proofSystem, proofB64, publicInputsB64, credentialType, predicate, nonce)
	if err != nil {
		// Log as failed event but do not surface the internal error to the caller.
		_ = s.persistEvent(ctx, verifierID, credentialType, false, proofSystem, nonce)
		return false, "", fmt.Errorf("service: zk proof call failed: %w", err)
	}

	eventID, persistErr := s.persistEventReturningID(ctx, verifierID, credentialType, valid, proofSystem, nonce)
	if persistErr != nil {
		// Non-fatal: the proof result is still returned; the caller can retry.
		_ = persistErr
	}

	return valid, eventID, nil
}

// GetVerificationHistory returns the most recent verification events for a verifier.
func (s *VerifierService) GetVerificationHistory(ctx context.Context, verifierID string, limit int32) ([]*repository.VerificationEventRecord, error) {
	if verifierID == "" {
		return nil, fmt.Errorf("service: verifier_id is required")
	}
	evts, err := s.repo.ListVerificationEvents(ctx, verifierID, limit)
	if err != nil {
		return nil, fmt.Errorf("service: list verification events: %w", err)
	}
	return evts, nil
}

// callZKProof POSTs to the zkproof service at <zkProofURL>/verify and returns
// the boolean result. Any non-2xx response is treated as a transient failure.
func (s *VerifierService) callZKProof(ctx context.Context, proofSystem, proofB64, publicInputsB64, credentialType, predicate, nonce string) (bool, error) {
	body := zkProofVerifyRequest{
		ProofSystem:     proofSystem,
		ProofB64:        proofB64,
		PublicInputsB64: publicInputsB64,
		CredentialType:  credentialType,
		Predicate:       predicate,
		Nonce:           nonce,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return false, fmt.Errorf("marshal zk proof request: %w", err)
	}

	url := s.zkProofURL + "/verify"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return false, fmt.Errorf("build http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("http post zkproof: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, fmt.Errorf("read zkproof response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("zkproof service returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var zkResp zkProofVerifyResponse
	if err := json.Unmarshal(respBytes, &zkResp); err != nil {
		return false, fmt.Errorf("unmarshal zkproof response: %w", err)
	}
	return zkResp.Valid, nil
}

// persistEvent saves a verification event with a new UUID. Errors are non-fatal.
func (s *VerifierService) persistEvent(ctx context.Context, verifierID, credentialType string, result bool, proofSystem, nonce string) error {
	_, err := s.persistEventReturningID(ctx, verifierID, credentialType, result, proofSystem, nonce)
	return err
}

// persistEventReturningID saves a verification event and returns its ID.
func (s *VerifierService) persistEventReturningID(ctx context.Context, verifierID, credentialType string, result bool, proofSystem, nonce string) (string, error) {
	eventID, err := newUUID()
	if err != nil {
		return "", fmt.Errorf("generate event ID: %w", err)
	}
	evt := repository.VerificationEventRecord{
		ID:             eventID,
		VerifierID:     verifierID,
		CredentialType: credentialType,
		Result:         result,
		ProofSystem:    proofSystem,
		Nonce:          nonce,
		OccurredAt:     time.Now().UTC(),
	}
	if err := s.repo.CreateVerificationEvent(ctx, evt); err != nil {
		return "", fmt.Errorf("persist event: %w", err)
	}
	return eventID, nil
}

// newUUID generates a random UUID v4 using crypto/rand.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	// Set version 4 and variant bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// containsString returns true if slice contains target (case-sensitive).
func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
