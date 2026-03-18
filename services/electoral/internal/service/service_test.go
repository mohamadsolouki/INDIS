package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IranProsperityProject/INDIS/services/electoral/internal/repository"
)

type fakeRepo struct {
	getElectionFn     func(ctx context.Context, id string) (*repository.ElectionRecord, error)
	nullifierExistsFn func(ctx context.Context, electionID, nullifierHash string) (bool, error)
	createElectionFn  func(ctx context.Context, rec repository.ElectionRecord) error
	castBallotFn      func(ctx context.Context, rec repository.BallotRecord) error
}

func (f *fakeRepo) CreateElection(ctx context.Context, rec repository.ElectionRecord) error {
	if f.createElectionFn != nil {
		return f.createElectionFn(ctx, rec)
	}
	return nil
}

func (f *fakeRepo) GetElection(ctx context.Context, id string) (*repository.ElectionRecord, error) {
	if f.getElectionFn != nil {
		return f.getElectionFn(ctx, id)
	}
	return &repository.ElectionRecord{ID: id}, nil
}

func (f *fakeRepo) NullifierExists(ctx context.Context, electionID, nullifierHash string) (bool, error) {
	if f.nullifierExistsFn != nil {
		return f.nullifierExistsFn(ctx, electionID, nullifierHash)
	}
	return false, nil
}

func (f *fakeRepo) CastBallot(ctx context.Context, rec repository.BallotRecord) error {
	if f.castBallotFn != nil {
		return f.castBallotFn(ctx, rec)
	}
	return nil
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
