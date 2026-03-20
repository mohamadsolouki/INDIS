package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IranProsperityProject/INDIS/services/verifier/internal/repository"
)

// fakeRepo is an in-memory implementation of VerifierRepository.
type fakeRepo struct {
	verifiers map[string]*repository.VerifierRecord
	events    []*repository.VerificationEventRecord

	createVerifierFn        func(ctx context.Context, rec repository.VerifierRecord) error
	getVerifierByIDFn       func(ctx context.Context, id string) (*repository.VerifierRecord, error)
	updateVerifierStatusFn  func(ctx context.Context, id, status string) error
	createVerificationEvtFn func(ctx context.Context, evt repository.VerificationEventRecord) error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		verifiers: make(map[string]*repository.VerifierRecord),
	}
}

func (f *fakeRepo) CreateVerifier(_ context.Context, rec repository.VerifierRecord) error {
	if f.createVerifierFn != nil {
		return f.createVerifierFn(context.Background(), rec)
	}
	r := rec
	f.verifiers[rec.ID] = &r
	return nil
}

func (f *fakeRepo) GetVerifierByID(_ context.Context, id string) (*repository.VerifierRecord, error) {
	if f.getVerifierByIDFn != nil {
		return f.getVerifierByIDFn(context.Background(), id)
	}
	rec, ok := f.verifiers[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return rec, nil
}

func (f *fakeRepo) ListVerifiers(_ context.Context, statusFilter string) ([]*repository.VerifierRecord, error) {
	var out []*repository.VerifierRecord
	for _, v := range f.verifiers {
		if statusFilter == "" || v.Status == statusFilter {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateVerifierStatus(_ context.Context, id, status string) error {
	if f.updateVerifierStatusFn != nil {
		return f.updateVerifierStatusFn(context.Background(), id, status)
	}
	rec, ok := f.verifiers[id]
	if !ok {
		return repository.ErrNotFound
	}
	rec.Status = status
	return nil
}

func (f *fakeRepo) CreateVerificationEvent(_ context.Context, evt repository.VerificationEventRecord) error {
	if f.createVerificationEvtFn != nil {
		return f.createVerificationEvtFn(context.Background(), evt)
	}
	cp := evt
	f.events = append(f.events, &cp)
	return nil
}

func (f *fakeRepo) ListVerificationEvents(_ context.Context, verifierID string, limit int32) ([]*repository.VerificationEventRecord, error) {
	var out []*repository.VerificationEventRecord
	for _, e := range f.events {
		if e.VerifierID == verifierID {
			cp := *e
			out = append(out, &cp)
		}
	}
	if limit > 0 && int(limit) < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// helper: register a verifier with authorized types
func registerVerifier(t *testing.T, svc *VerifierService, authorizedTypes []string) *RegisterResult {
	t.Helper()
	res, err := svc.RegisterVerifier(context.Background(), "Test Org", "government", authorizedTypes, "", 0)
	if err != nil {
		t.Fatalf("RegisterVerifier failed: %v", err)
	}
	return res
}

func TestRegisterVerifier_Success(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "http://localhost:1")
	res, err := svc.RegisterVerifier(context.Background(), "Ministry of Interior", "government", []string{"citizenship"}, "nationwide", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.VerifierID == "" {
		t.Error("expected VerifierID to be set")
	}
	if res.CertificateID == "" {
		t.Error("expected CertificateID to be set")
	}
	// Ed25519 public key is 32 bytes → 64 hex chars
	if len(res.PublicKeyHex) != 64 {
		t.Errorf("expected PublicKeyHex length 64, got %d", len(res.PublicKeyHex))
	}
}

func TestRegisterVerifier_MissingOrgName(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "http://localhost:1")
	_, err := svc.RegisterVerifier(context.Background(), "", "government", nil, "", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "org_name") {
		t.Errorf("expected error containing 'org_name', got: %v", err)
	}
}

func TestRegisterVerifier_MissingOrgType(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "http://localhost:1")
	_, err := svc.RegisterVerifier(context.Background(), "Test Org", "", nil, "", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "org_type") {
		t.Errorf("expected error containing 'org_type', got: %v", err)
	}
}

func TestGetVerifier_Found(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "http://localhost:1")
	res := registerVerifier(t, svc, []string{"citizenship"})

	got, err := svc.GetVerifier(context.Background(), res.VerifierID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != res.VerifierID {
		t.Errorf("expected ID %s, got %s", res.VerifierID, got.ID)
	}
	if got.PublicKeyHex != res.PublicKeyHex {
		t.Errorf("public key mismatch")
	}
}

func TestGetVerifier_NotFound(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "http://localhost:1")
	_, err := svc.GetVerifier(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSuspendVerifier_Success(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "http://localhost:1")
	res := registerVerifier(t, svc, nil)

	if err := svc.SuspendVerifier(context.Background(), res.VerifierID, "policy violation"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := svc.GetVerifier(context.Background(), res.VerifierID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "suspended" {
		t.Errorf("expected status 'suspended', got %s", got.Status)
	}
}

func TestSuspendVerifier_Revoked(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "http://localhost:1")
	res := registerVerifier(t, svc, nil)

	if err := svc.SuspendVerifier(context.Background(), res.VerifierID, "revoked"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := svc.GetVerifier(context.Background(), res.VerifierID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "revoked" {
		t.Errorf("expected status 'revoked', got %s", got.Status)
	}
}

func TestVerifyCredential_RequiresVerifierID(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "http://localhost:1")
	_, _, err := svc.VerifyCredential(context.Background(), "", "citizenship", "", "nonce1", "groth16", "proofB64", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyCredential_ActiveVerifierCallsZKProof(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer ts.Close()

	repo := newFakeRepo()
	svc := New(repo, ts.URL)
	res := registerVerifier(t, svc, []string{"citizenship"})

	valid, eventID, err := svc.VerifyCredential(context.Background(), res.VerifierID, "citizenship", "", "nonce1", "groth16", "proofB64==", "inputs==")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true")
	}
	if eventID == "" {
		t.Error("expected eventID to be set")
	}
}

func TestVerifyCredential_SuspendedVerifierRejected(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "http://localhost:1")
	res := registerVerifier(t, svc, []string{"citizenship"})

	if err := svc.SuspendVerifier(context.Background(), res.VerifierID, "policy"); err != nil {
		t.Fatalf("unexpected error suspending: %v", err)
	}

	_, _, err := svc.VerifyCredential(context.Background(), res.VerifierID, "citizenship", "", "nonce1", "groth16", "proof", "")
	if err == nil {
		t.Fatal("expected error for suspended verifier")
	}
	if !strings.Contains(err.Error(), "suspended") {
		t.Errorf("expected 'suspended' in error, got: %v", err)
	}
}

func TestVerifyCredential_UnauthorizedType(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer ts.Close()

	repo := newFakeRepo()
	svc := New(repo, ts.URL)
	res := registerVerifier(t, svc, []string{"citizenship"})

	_, _, err := svc.VerifyCredential(context.Background(), res.VerifierID, "voter_eligibility", "", "nonce1", "groth16", "proof", "")
	if err == nil {
		t.Fatal("expected error for unauthorized type")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Errorf("expected 'not authorized' in error, got: %v", err)
	}
}

func TestGetVerificationHistory_Empty(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "http://localhost:1")
	res := registerVerifier(t, svc, nil)

	evts, err := svc.GetVerificationHistory(context.Background(), res.VerifierID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 0 {
		t.Errorf("expected empty history, got %d events", len(evts))
	}
}

func TestGetVerificationHistory_AfterVerify(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	defer ts.Close()

	repo := newFakeRepo()
	svc := New(repo, ts.URL)
	res := registerVerifier(t, svc, []string{"citizenship"})

	if _, _, err := svc.VerifyCredential(context.Background(), res.VerifierID, "citizenship", "", "nonce1", "groth16", "proofB64==", "inputs=="); err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	evts, err := svc.GetVerificationHistory(context.Background(), res.VerifierID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 1 {
		t.Errorf("expected 1 event, got %d", len(evts))
	}
	if evts[0].CredentialType != "citizenship" {
		t.Errorf("unexpected credential type: %s", evts[0].CredentialType)
	}
	_ = time.Now() // suppress import warning
}
