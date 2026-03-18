package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/repository"
)

type mockRepo struct {
	records map[string]repository.CredentialRecord

	createErr error
	getErr    error
	revokeErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{records: make(map[string]repository.CredentialRecord)}
}

func (m *mockRepo) Create(_ context.Context, rec repository.CredentialRecord) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.records[rec.ID] = rec
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*repository.CredentialRecord, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	rec, ok := m.records[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &rec, nil
}

func (m *mockRepo) Revoke(_ context.Context, id, reason string) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	rec, ok := m.records[id]
	if !ok {
		return repository.ErrNotFound
	}
	if rec.Revoked {
		return repository.ErrAlreadyRevoked
	}
	now := time.Now().UTC()
	rec.Revoked = true
	rec.RevokeReason = reason
	rec.RevokedAt = &now
	m.records[id] = rec
	return nil
}

type mockChain struct {
	anchorErr error
	revokeErr error
}

func (m *mockChain) RegisterDID(_ context.Context, _ string, _ blockchain.DIDDocument) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{TxID: "did-tx"}, nil
}

func (m *mockChain) ResolveDID(_ context.Context, _ string) (*blockchain.DIDDocument, error) {
	return &blockchain.DIDDocument{}, nil
}

func (m *mockChain) UpdateDIDDocument(_ context.Context, _ string, _ blockchain.DIDDocument) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{TxID: "did-update-tx"}, nil
}

func (m *mockChain) DeactivateDID(_ context.Context, _ string) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{TxID: "did-deactivate-tx"}, nil
}

func (m *mockChain) AnchorCredential(_ context.Context, _ blockchain.Hash, _ string) (*blockchain.TxReceipt, error) {
	if m.anchorErr != nil {
		return nil, m.anchorErr
	}
	return &blockchain.TxReceipt{TxID: "anchor-tx"}, nil
}

func (m *mockChain) VerifyAnchor(_ context.Context, _ blockchain.Hash) (*blockchain.AnchorStatus, error) {
	return &blockchain.AnchorStatus{Exists: true}, nil
}

func (m *mockChain) RevokeCredential(_ context.Context, _ string, _ blockchain.RevocationReason) (*blockchain.TxReceipt, error) {
	if m.revokeErr != nil {
		return nil, m.revokeErr
	}
	return &blockchain.TxReceipt{TxID: "revoke-tx"}, nil
}

func (m *mockChain) CheckRevocationStatus(_ context.Context, _ string) (*blockchain.RevocationStatus, error) {
	return &blockchain.RevocationStatus{Revoked: false}, nil
}

func (m *mockChain) GetRevocationList(_ context.Context, _ string) (*blockchain.RevocationList, error) {
	return &blockchain.RevocationList{}, nil
}

func (m *mockChain) LogVerificationEvent(_ context.Context, _ blockchain.AnonymizedVerificationEvent) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{TxID: "event-tx"}, nil
}

func (m *mockChain) GetBlockHeight(_ context.Context) (uint64, error) {
	return 1, nil
}

func (m *mockChain) GetValidatorStatus(_ context.Context) ([]blockchain.ValidatorStatus, error) {
	return nil, nil
}

func (m *mockChain) EstimateTxTime(_ context.Context) (time.Duration, error) {
	return 100 * time.Millisecond, nil
}

func newServiceForTest(t *testing.T, repo credentialRepository, chain blockchain.BlockchainAdapter) *CredentialService {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return New(repo, chain, "did:indis:issuer123", privateKey)
}

func TestIssueCredential_UnknownType(t *testing.T) {
	svc := newServiceForTest(t, newMockRepo(), &mockChain{})

	_, err := svc.IssueCredential(context.Background(), "did:indis:subject1", 999, map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected unknown credential type error, got nil")
	}
}

func TestIssueCredential_Success(t *testing.T) {
	repo := newMockRepo()
	svc := newServiceForTest(t, repo, &mockChain{})

	result, err := svc.IssueCredential(
		context.Background(),
		"did:indis:subject1",
		1,
		map[string]string{"citizenship": "IR"},
	)
	if err != nil {
		t.Fatalf("IssueCredential returned error: %v", err)
	}
	if result.CredentialID == "" {
		t.Fatal("expected non-empty credential ID")
	}
	if result.TxID != "anchor-tx" {
		t.Fatalf("expected tx id anchor-tx, got %q", result.TxID)
	}
	if len(result.CredentialData) == 0 {
		t.Fatal("expected credential data bytes")
	}

	stored, ok := repo.records[result.CredentialID]
	if !ok {
		t.Fatalf("expected credential record %q to be stored", result.CredentialID)
	}
	if stored.SubjectDID != "did:indis:subject1" {
		t.Fatalf("expected subject DID did:indis:subject1, got %q", stored.SubjectDID)
	}
}

func TestIssueCredential_BlockchainFailureNonFatal(t *testing.T) {
	repo := newMockRepo()
	svc := newServiceForTest(t, repo, &mockChain{anchorErr: errors.New("anchor failed")})

	result, err := svc.IssueCredential(context.Background(), "did:indis:subject2", 2, map[string]string{"age_range": "18-25"})
	if err != nil {
		t.Fatalf("IssueCredential returned error on anchor failure: %v", err)
	}
	if result.TxID != "" {
		t.Fatalf("expected empty tx id when anchoring fails, got %q", result.TxID)
	}
}

func TestVerifyCredential_CurrentStubContract(t *testing.T) {
	svc := newServiceForTest(t, newMockRepo(), &mockChain{})

	valid, reason := svc.VerifyCredential(context.Background(), []byte("proof"), []byte("vk"))
	if !valid {
		t.Fatal("expected valid=true for current stub verifier")
	}
	if reason != "" {
		t.Fatalf("expected empty reason for current stub verifier, got %q", reason)
	}
}

func TestRevokeCredential_Success(t *testing.T) {
	repo := newMockRepo()
	repo.records["cred-1"] = repository.CredentialRecord{ID: "cred-1"}
	svc := newServiceForTest(t, repo, &mockChain{})

	txID, err := svc.RevokeCredential(context.Background(), "cred-1", "expired", "did:indis:revoker")
	if err != nil {
		t.Fatalf("RevokeCredential returned error: %v", err)
	}
	if txID != "revoke-tx" {
		t.Fatalf("expected tx id revoke-tx, got %q", txID)
	}
}

func TestRevokeCredential_MapsNotFound(t *testing.T) {
	svc := newServiceForTest(t, newMockRepo(), &mockChain{})

	_, err := svc.RevokeCredential(context.Background(), "missing", "expired", "did:indis:revoker")
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}

func TestRevokeCredential_MapsAlreadyRevoked(t *testing.T) {
	repo := newMockRepo()
	now := time.Now().UTC()
	repo.records["cred-1"] = repository.CredentialRecord{ID: "cred-1", Revoked: true, RevokedAt: &now}
	svc := newServiceForTest(t, repo, &mockChain{})

	_, err := svc.RevokeCredential(context.Background(), "cred-1", "expired", "did:indis:revoker")
	if err == nil {
		t.Fatal("expected already revoked error, got nil")
	}
}

func TestCheckRevocationStatus_Success(t *testing.T) {
	repo := newMockRepo()
	now := time.Date(2026, 3, 18, 10, 20, 30, 0, time.UTC)
	repo.records["cred-1"] = repository.CredentialRecord{
		ID:           "cred-1",
		Revoked:      true,
		RevokeReason: "expired",
		RevokedAt:    &now,
	}
	svc := newServiceForTest(t, repo, &mockChain{})

	status, err := svc.CheckRevocationStatus(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("CheckRevocationStatus returned error: %v", err)
	}
	if !status.Revoked {
		t.Fatal("expected revoked=true")
	}
	if status.Reason != "expired" {
		t.Fatalf("expected reason expired, got %q", status.Reason)
	}
	if status.RevokedAt != "2026-03-18T10:20:30Z" {
		t.Fatalf("expected RFC3339 timestamp, got %q", status.RevokedAt)
	}
}

func TestCheckRevocationStatus_NotFound(t *testing.T) {
	svc := newServiceForTest(t, newMockRepo(), &mockChain{})

	_, err := svc.CheckRevocationStatus(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}
