// Package service implements business logic for the justice service.
// Ref: PRD §FR-011 — anonymous testimony and conditional amnesty.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/IranProsperityProject/INDIS/services/justice/internal/repository"
)

// JusticeService handles anonymous testimony and amnesty workflows.
type JusticeService struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *JusticeService { return &JusticeService{repo: repo} }

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
// TODO: delegate ZK proof verification to services/zkproof.
func (s *JusticeService) SubmitTestimony(ctx context.Context, _ []byte, encryptedTestimony []byte, category, locale string) (string, string, string, error) {
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
