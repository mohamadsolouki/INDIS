package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	electoralv1 "github.com/mohamadsolouki/INDIS/api/gen/go/electoral/v1"
	"github.com/mohamadsolouki/INDIS/services/electoral/internal/repository"
)

type fakeRepo struct {
	getElectionFn     func(ctx context.Context, id string) (*repository.ElectionRecord, error)
	nullifierExistsFn func(ctx context.Context, electionID, nullifierHash string) (bool, error)
	nonceExistsFn     func(ctx context.Context, electionID, nonceHash string, since time.Time) (bool, error)
	createElectionFn  func(ctx context.Context, rec repository.ElectionRecord) error
	castBallotFn      func(ctx context.Context, rec repository.BallotRecord) error
	castedNullifiers  map[string]struct{}
	castedNonces      map[string]time.Time
	elections         map[string]*repository.ElectionRecord
}

func (f *fakeRepo) CreateElection(ctx context.Context, rec repository.ElectionRecord) error {
	if f.createElectionFn != nil {
		return f.createElectionFn(ctx, rec)
	}
	if f.elections == nil {
		f.elections = make(map[string]*repository.ElectionRecord)
	}
	r := rec
	f.elections[rec.ID] = &r
	return nil
}

func (f *fakeRepo) GetElection(ctx context.Context, id string) (*repository.ElectionRecord, error) {
	if f.getElectionFn != nil {
		return f.getElectionFn(ctx, id)
	}
	if f.elections != nil {
		if rec, ok := f.elections[id]; ok {
			return rec, nil
		}
	}
	// Default: return an open election so tests that don't pre-register one can proceed.
	now := time.Now().UTC()
	return &repository.ElectionRecord{
		ID:       id,
		Status:   "scheduled",
		OpensAt:  now.Add(-1 * time.Hour),
		ClosesAt: now.Add(24 * time.Hour),
	}, nil
}

func (f *fakeRepo) UpdateElectionStatus(_ context.Context, id, newStatus string) error {
	if f.elections != nil {
		if rec, ok := f.elections[id]; ok {
			rec.Status = newStatus
		}
	}
	return nil
}

func (f *fakeRepo) NullifierExists(ctx context.Context, electionID, nullifierHash string) (bool, error) {
	if f.nullifierExistsFn != nil {
		return f.nullifierExistsFn(ctx, electionID, nullifierHash)
	}
	if f.castedNullifiers == nil {
		return false, nil
	}
	_, exists := f.castedNullifiers[electionID+"|"+nullifierHash]
	if exists {
		return true, nil
	}
	return false, nil
}

func (f *fakeRepo) CastBallot(ctx context.Context, rec repository.BallotRecord) error {
	if f.castBallotFn != nil {
		return f.castBallotFn(ctx, rec)
	}
	if f.castedNullifiers == nil {
		f.castedNullifiers = make(map[string]struct{})
	}
	f.castedNullifiers[rec.ElectionID+"|"+rec.NullifierHash] = struct{}{}
	if rec.TransportNonceHash != nil {
		if f.castedNonces == nil {
			f.castedNonces = make(map[string]time.Time)
		}
		ts := rec.CastAt
		f.castedNonces[rec.ElectionID+"|"+*rec.TransportNonceHash] = ts
	}
	return nil
}

func (f *fakeRepo) TransportNonceExistsSince(ctx context.Context, electionID, nonceHash string, since time.Time) (bool, error) {
	if f.nonceExistsFn != nil {
		return f.nonceExistsFn(ctx, electionID, nonceHash, since)
	}
	if f.castedNonces == nil {
		return false, nil
	}
	ts, exists := f.castedNonces[electionID+"|"+nonceHash]
	if !exists {
		return false, nil
	}
	return !ts.Before(since), nil
}

func TestVerifyEligibilityCallsZKService(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, ts.URL)
	eligible, nullifier, reason, err := svc.VerifyEligibility(context.Background(), "el-1", []byte("proof"), []byte("public"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !eligible {
		t.Fatalf("expected eligible=true")
	}
	if nullifier == "" {
		t.Fatalf("expected nullifier hash")
	}
	if reason != "" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestVerifyEligibilityRejectsInvalidProof(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":false,"reason":"invalid stark proof"}`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, ts.URL)
	eligible, nullifier, reason, err := svc.VerifyEligibility(context.Background(), "el-1", []byte("proof"), []byte("public"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eligible {
		t.Fatalf("expected eligible=false")
	}
	if nullifier != "" {
		t.Fatalf("expected empty nullifier for invalid proof")
	}
	if reason != "invalid stark proof" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestVerifyEligibilityDetectsNullifierReuse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
	}))
	defer ts.Close()

	repo := &fakeRepo{
		nullifierExistsFn: func(context.Context, string, string) (bool, error) {
			return true, nil
		},
	}
	svc := New(repo, ts.URL)

	eligible, _, reason, err := svc.VerifyEligibility(context.Background(), "el-1", []byte("proof"), []byte("public"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eligible {
		t.Fatalf("expected eligible=false")
	}
	if reason != "voter has already voted in this election" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestVerifyEligibilityZKUnavailableReturnsError(t *testing.T) {
	t.Parallel()

	svc := New(&fakeRepo{}, "http://127.0.0.1:1")
	_, _, _, err := svc.VerifyEligibility(context.Background(), "el-1", []byte("proof"), []byte("public"))
	if err == nil {
		t.Fatalf("expected zk availability error")
	}
}

func TestFullElectoralFlowRegisterVerifyCastAndRejectDoubleVote(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
	}))
	defer ts.Close()

	repo := &fakeRepo{}
	svc := New(repo, ts.URL)

	openAt := time.Now().UTC().Add(-1 * time.Hour) // already open
	closeAt := time.Now().UTC().Add(24 * time.Hour)
	electionID, err := svc.RegisterElection(context.Background(), &electoralv1.RegisterElectionRequest{
		Name:     "Referendum 2026",
		OpensAt:  openAt.Format(time.RFC3339),
		ClosesAt: closeAt.Format(time.RFC3339),
		AdminDid: "did:indis:admin:electoral",
	})
	if err != nil {
		t.Fatalf("register election failed: %v", err)
	}

	eligible, nullifier, reason, err := svc.VerifyEligibility(
		context.Background(),
		electionID,
		[]byte("stark-proof"),
		[]byte("stark-public-inputs"),
	)
	if err != nil {
		t.Fatalf("verify eligibility failed: %v", err)
	}
	if !eligible || reason != "" || nullifier == "" {
		t.Fatalf("unexpected verify output: eligible=%v nullifier=%q reason=%q", eligible, nullifier, reason)
	}

	firstReceipt, _, err := svc.CastBallot(context.Background(), &electoralv1.CastBallotRequest{
		ElectionId:    electionID,
		NullifierHash: nullifier,
		EncryptedVote: []byte("encrypted-choice-a"),
		ZkProof:       []byte("stark-ballot-proof"),
	})
	if err != nil {
		t.Fatalf("cast ballot failed: %v", err)
	}
	if firstReceipt == "" {
		t.Fatalf("expected non-empty receipt hash")
	}

	_, _, err = svc.CastBallot(context.Background(), &electoralv1.CastBallotRequest{
		ElectionId:    electionID,
		NullifierHash: nullifier,
		EncryptedVote: []byte("encrypted-choice-b"),
		ZkProof:       []byte("stark-ballot-proof"),
	})
	if err == nil {
		t.Fatalf("expected double-vote rejection")
	}
	if !strings.Contains(err.Error(), "double-vote detected") {
		t.Fatalf("expected double-vote error, got: %v", err)
	}
}

func TestSubmitRemoteBallotAcceptsValidRequest(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
	}))
	defer ts.Close()

	var persisted repository.BallotRecord
	svc := New(&fakeRepo{
		castBallotFn: func(ctx context.Context, rec repository.BallotRecord) error {
			persisted = rec
			return nil
		},
	}, ts.URL)

	receipt, blockHeight, acceptedAt, err := svc.SubmitRemoteBallot(context.Background(), &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        "elc-1",
		NullifierHash:     "nullifier-1",
		EncryptedVote:     []byte("elgamal-ciphertext"),
		ZkProof:           []byte("stark-proof"),
		ClientAttestation: []byte("attestation"),
		SubmittedAt:       time.Now().UTC().Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("nonce-1"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if receipt == "" || blockHeight == "" || acceptedAt == "" {
		t.Fatalf("expected receipt, block height, and accepted_at")
	}
	if persisted.ClientAttestationHash == nil || *persisted.ClientAttestationHash == "" {
		t.Fatalf("expected client attestation hash to be persisted")
	}
	if persisted.TransportNonceHash == nil || *persisted.TransportNonceHash == "" {
		t.Fatalf("expected transport nonce hash to be persisted")
	}
	if persisted.ClientSubmittedAt == nil || persisted.AcceptedAt == nil {
		t.Fatalf("expected client_submitted_at and accepted_at metadata")
	}
}

func TestSubmitRemoteBallotRejectsExpiredTimestamp(t *testing.T) {
	t.Parallel()

	svc := New(&fakeRepo{}, "http://127.0.0.1:1")

	_, _, _, err := svc.SubmitRemoteBallot(context.Background(), &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        "elc-1",
		NullifierHash:     "nullifier-1",
		EncryptedVote:     []byte("elgamal-ciphertext"),
		ZkProof:           []byte("stark-proof"),
		ClientAttestation: []byte("attestation"),
		SubmittedAt:       time.Now().UTC().Add(-11 * time.Minute).Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("nonce-1"),
	})
	if err == nil {
		t.Fatalf("expected timestamp expiry error")
	}
	if !strings.Contains(err.Error(), "timestamp expired") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubmitRemoteBallotRejectsFutureTimestamp(t *testing.T) {
	t.Parallel()

	svc := New(&fakeRepo{}, "http://127.0.0.1:1")

	_, _, _, err := svc.SubmitRemoteBallot(context.Background(), &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        "elc-1",
		NullifierHash:     "nullifier-1",
		EncryptedVote:     []byte("elgamal-ciphertext"),
		ZkProof:           []byte("stark-proof"),
		ClientAttestation: []byte("attestation"),
		SubmittedAt:       time.Now().UTC().Add(3 * time.Minute).Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("nonce-1"),
	})
	if err == nil {
		t.Fatalf("expected future timestamp rejection")
	}
	if !strings.Contains(err.Error(), "too far in the future") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubmitRemoteBallotRejectsReplayedNonce(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, ts.URL)

	req := &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        "elc-1",
		NullifierHash:     "nullifier-1",
		EncryptedVote:     []byte("elgamal-ciphertext"),
		ZkProof:           []byte("stark-proof"),
		ClientAttestation: []byte("attestation"),
		SubmittedAt:       time.Now().UTC().Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("nonce-replay"),
	}

	if _, _, _, err := svc.SubmitRemoteBallot(context.Background(), req); err != nil {
		t.Fatalf("first submit should succeed, got %v", err)
	}

	req.NullifierHash = "nullifier-2"
	_, _, _, err := svc.SubmitRemoteBallot(context.Background(), req)
	if err == nil {
		t.Fatalf("expected replay nonce rejection")
	}
	if !strings.Contains(err.Error(), "replayed remote ballot nonce") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubmitRemoteBallotAllowsNonceOutsideReplayWindow(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
	}))
	defer ts.Close()

	repo := &fakeRepo{
		nonceExistsFn: func(ctx context.Context, electionID, nonceHash string, since time.Time) (bool, error) {
			_ = ctx
			_ = electionID
			_ = nonceHash
			_ = since
			return false, nil
		},
	}

	svc := NewWithNonceReplayWindow(repo, ts.URL, 30*time.Minute)

	_, _, _, err := svc.SubmitRemoteBallot(context.Background(), &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        "elc-1",
		NullifierHash:     "nullifier-1",
		EncryptedVote:     []byte("elgamal-ciphertext"),
		ZkProof:           []byte("stark-proof"),
		ClientAttestation: []byte("attestation"),
		SubmittedAt:       time.Now().UTC().Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("nonce-old-window"),
	})
	if err != nil {
		t.Fatalf("expected nonce outside replay window to be accepted, got: %v", err)
	}
}

func TestCastBallotRejectsInvalidBallotProof(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":false,"reason":"invalid ballot proof"}`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, ts.URL)

	_, _, err := svc.CastBallot(context.Background(), &electoralv1.CastBallotRequest{
		ElectionId:    "elc-1",
		NullifierHash: "nullifier-1",
		EncryptedVote: []byte("encrypted-choice-a"),
		ZkProof:       []byte("invalid-proof"),
	})
	if err == nil {
		t.Fatalf("expected invalid proof error")
	}
	if !strings.Contains(err.Error(), "invalid ballot proof") {
		t.Fatalf("unexpected error: %v", err)
	}
}
