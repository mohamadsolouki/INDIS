// Package service implements business logic for the justice service.
// Ref: PRD §FR-011 — anonymous testimony and conditional amnesty.
package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/IranProsperityProject/INDIS/services/justice/internal/repository"
)

// JusticeService handles anonymous testimony and amnesty workflows.
type JusticeService struct {
	repo      JusticeRepository
	zkService ZKProofService
}

// JusticeRepository defines storage behavior required by service workflows.
type JusticeRepository interface {
	CreateTestimony(ctx context.Context, rec repository.TestimonyRecord) error
	GetTestimonyByReceipt(ctx context.Context, receiptToken string) (*repository.TestimonyRecord, error)
	GetCaseStatus(ctx context.Context, caseID string) (string, time.Time, error)
	CreateAmnestyCase(ctx context.Context, rec repository.AmnestyRecord) error
	UpdateCaseStatus(ctx context.Context, caseID, newStatus string) error
}

// ZKProofService defines ZK operations used by anonymous testimony submission.
type ZKProofService interface {
	ProveAndVerifyCitizenship(ctx context.Context, zkCitizenshipProof []byte) (bool, string, error)
}

func New(repo JusticeRepository, zkProofURL string) *JusticeService {
	return &JusticeService{
		repo:      repo,
		zkService: newHTTPZKProofService(zkProofURL),
	}
}

func generateID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func generateReceiptToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16]), nil
}

// SubmitTestimony records an anonymous encrypted testimony.
// The ZK citizenship proof is verified before acceptance.
func (s *JusticeService) SubmitTestimony(ctx context.Context, zkCitizenshipProof []byte, encryptedTestimony []byte, category, locale string) (string, string, string, error) {
	valid, reason, err := s.zkService.ProveAndVerifyCitizenship(ctx, zkCitizenshipProof)
	if err != nil {
		return "", "", "", fmt.Errorf("service: zk citizenship proof: %w", err)
	}
	if !valid {
		return "", "", "", fmt.Errorf("service: invalid citizenship proof: %s", reason)
	}

	caseID, err := generateID("jt_")
	if err != nil {
		return "", "", "", fmt.Errorf("service: generate case id: %w", err)
	}
	receiptToken, err := generateReceiptToken()
	if err != nil {
		return "", "", "", fmt.Errorf("service: generate receipt: %w", err)
	}
	now := time.Now().UTC()
	rec := repository.TestimonyRecord{
		CaseID:             caseID,
		ReceiptToken:       receiptToken,
		EncryptedTestimony: encryptedTestimony,
		Category:           category,
		Locale:             locale,
		Status:             "received",
		CreatedAt:          now,
	}
	if err = s.repo.CreateTestimony(ctx, rec); err != nil {
		return "", "", "", fmt.Errorf("service: store testimony: %w", err)
	}
	return receiptToken, caseID, now.Format(time.RFC3339), nil
}

// LinkTestimony attaches a follow-up testimony to the original case via receipt token.
func (s *JusticeService) LinkTestimony(ctx context.Context, receiptToken string, encryptedTestimony []byte, locale string) (string, string, error) {
	original, err := s.repo.GetTestimonyByReceipt(ctx, receiptToken)
	if errors.Is(err, repository.ErrNotFound) {
		return "", "", fmt.Errorf("service: receipt token not found")
	}
	if err != nil {
		return "", "", fmt.Errorf("service: get original testimony: %w", err)
	}
	newToken, err := generateReceiptToken()
	if err != nil {
		return "", "", fmt.Errorf("service: generate token: %w", err)
	}
	now := time.Now().UTC()
	rec := repository.TestimonyRecord{
		CaseID:             original.CaseID,
		ReceiptToken:       newToken,
		EncryptedTestimony: encryptedTestimony,
		Category:           original.Category,
		Locale:             locale,
		Status:             "received",
		LinkedToCaseID:     original.CaseID,
		CreatedAt:          now,
	}
	if err = s.repo.CreateTestimony(ctx, rec); err != nil {
		return "", "", fmt.Errorf("service: store linked testimony: %w", err)
	}
	return original.CaseID, now.Format(time.RFC3339), nil
}

// InitiateAmnesty begins the conditional amnesty workflow.
// Full verified identity is required — no ZK anonymity. Ref: PRD §FR-011.
func (s *JusticeService) InitiateAmnesty(ctx context.Context, applicantDID string, encryptedDeclaration []byte, category string) (string, string, string, error) {
	if applicantDID == "" {
		return "", "", "", fmt.Errorf("service: applicant DID is required for amnesty")
	}
	caseID, err := generateID("ja_")
	if err != nil {
		return "", "", "", fmt.Errorf("service: generate case id: %w", err)
	}
	receipt, err := generateReceiptToken()
	if err != nil {
		return "", "", "", fmt.Errorf("service: generate receipt: %w", err)
	}
	now := time.Now().UTC()
	rec := repository.AmnestyRecord{
		CaseID:               caseID,
		ApplicantDID:         applicantDID,
		EncryptedDeclaration: encryptedDeclaration,
		Category:             category,
		Status:               "received",
		Receipt:              receipt,
		CreatedAt:            now,
	}
	if err = s.repo.CreateAmnestyCase(ctx, rec); err != nil {
		return "", "", "", fmt.Errorf("service: store amnesty case: %w", err)
	}
	return caseID, receipt, now.Format(time.RFC3339), nil
}

// GetCaseStatus returns the public status of a case (no case details exposed).
func (s *JusticeService) GetCaseStatus(ctx context.Context, caseID, receiptToken string) (string, string, string, error) {
	lookupID := caseID
	if lookupID == "" && receiptToken != "" {
		rec, err := s.repo.GetTestimonyByReceipt(ctx, receiptToken)
		if err != nil {
			return "", "", "", fmt.Errorf("service: case not found")
		}
		lookupID = rec.CaseID
	}
	st, updatedAt, err := s.repo.GetCaseStatus(ctx, lookupID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", "", "", fmt.Errorf("service: case not found: %s", lookupID)
	}
	if err != nil {
		return "", "", "", fmt.Errorf("service: get status: %w", err)
	}
	return lookupID, st, updatedAt.UTC().Format(time.RFC3339), nil
}

// validStatusTransitions defines the allowed case status progression.
var validStatusTransitions = map[string]string{
	"received":     "under_review",
	"under_review": "referred",
	"referred":     "closed",
}

// AdvanceCaseStatus transitions a case to its next status.
// Only sequential forward transitions are allowed. Ref: PRD §FR-011.
func (s *JusticeService) AdvanceCaseStatus(ctx context.Context, caseID, adminDID string) (string, string, error) {
	current, _, err := s.repo.GetCaseStatus(ctx, caseID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", "", fmt.Errorf("service: case not found: %s", caseID)
	}
	if err != nil {
		return "", "", fmt.Errorf("service: get case status: %w", err)
	}
	next, ok := validStatusTransitions[current]
	if !ok {
		return "", "", fmt.Errorf("service: case %s is already in terminal status: %s", caseID, current)
	}
	if err := s.repo.UpdateCaseStatus(ctx, caseID, next); err != nil {
		return "", "", fmt.Errorf("service: update case status: %w", err)
	}
	return caseID, next, nil
}

type zkProveRequest struct {
	ProofSystem string `json:"proof_system"`
	CircuitID   string `json:"circuit_id"`
	InputB64    string `json:"input_b64"`
}

type zkProveResponse struct {
	ProofB64 string `json:"proof_b64"`
}

type zkVerifyRequest struct {
	ProofSystem string `json:"proof_system"`
	ProofB64    string `json:"proof_b64"`
}

type zkVerifyResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

type httpZKProofService struct {
	baseURL string
	client  *http.Client
}

func newHTTPZKProofService(baseURL string) ZKProofService {
	return &httpZKProofService{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (z *httpZKProofService) ProveAndVerifyCitizenship(ctx context.Context, zkCitizenshipProof []byte) (bool, string, error) {
	if z.baseURL == "" {
		return false, "", fmt.Errorf("zk proof URL is not configured")
	}

	proveReq := zkProveRequest{
		ProofSystem: "bulletproofs",
		CircuitID:   "citizenship_proof",
		InputB64:    base64.StdEncoding.EncodeToString(zkCitizenshipProof),
	}
	proveBody, err := json.Marshal(proveReq)
	if err != nil {
		return false, "", err
	}

	proofB64, err := z.callProve(ctx, proveBody)
	if err != nil {
		return false, "", err
	}

	verifyReq := zkVerifyRequest{ProofSystem: "bulletproofs", ProofB64: proofB64}
	verifyBody, err := json.Marshal(verifyReq)
	if err != nil {
		return false, "", err
	}

	return z.callVerify(ctx, verifyBody)
}

func (z *httpZKProofService) callProve(ctx context.Context, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, z.baseURL+"/prove", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := z.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("zk prove status: %s", resp.Status)
	}

	var out zkProveResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ProofB64 == "" {
		return "", fmt.Errorf("zk prove response missing proof")
	}
	return out.ProofB64, nil
}

func (z *httpZKProofService) callVerify(ctx context.Context, body []byte) (bool, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, z.baseURL+"/verify", bytes.NewReader(body))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := z.client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("zk verify status: %s", resp.Status)
	}

	var out zkVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "", err
	}

	return out.Valid, out.Reason, nil
}
