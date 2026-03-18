package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IranProsperityProject/INDIS/services/justice/internal/repository"
)

type fakeRepo struct {
	createTestimonyFn  func(ctx context.Context, rec repository.TestimonyRecord) error
	getByReceiptFn     func(ctx context.Context, receiptToken string) (*repository.TestimonyRecord, error)
	getCaseStatusFn    func(ctx context.Context, caseID string) (string, time.Time, error)
	createAmnestyFn    func(ctx context.Context, rec repository.AmnestyRecord) error
	createdTestimonies []repository.TestimonyRecord
}

func (f *fakeRepo) CreateTestimony(ctx context.Context, rec repository.TestimonyRecord) error {
	f.createdTestimonies = append(f.createdTestimonies, rec)
	if f.createTestimonyFn != nil {
		return f.createTestimonyFn(ctx, rec)
	}
	return nil
}

func (f *fakeRepo) GetTestimonyByReceipt(ctx context.Context, receiptToken string) (*repository.TestimonyRecord, error) {
	if f.getByReceiptFn != nil {
		return f.getByReceiptFn(ctx, receiptToken)
	}
	return nil, repository.ErrNotFound
}

func (f *fakeRepo) GetCaseStatus(ctx context.Context, caseID string) (string, time.Time, error) {
	if f.getCaseStatusFn != nil {
		return f.getCaseStatusFn(ctx, caseID)
	}
	return "", time.Time{}, repository.ErrNotFound
}

func (f *fakeRepo) CreateAmnestyCase(ctx context.Context, rec repository.AmnestyRecord) error {
	if f.createAmnestyFn != nil {
		return f.createAmnestyFn(ctx, rec)
	}
	return nil
}

func TestSubmitTestimonyProveAndVerifySuccess(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prove":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"proof_b64":"cHJvb2Y="}`))
		case "/verify":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"valid":true,"reason":""}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	repo := &fakeRepo{}
	svc := New(repo, ts.URL)

	receipt, caseID, submittedAt, err := svc.SubmitTestimony(context.Background(), []byte("citizenship-proof"), []byte("encrypted"), "human-rights", "fa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receipt == "" || caseID == "" || submittedAt == "" {
		t.Fatalf("expected non-empty response values")
	}
	if len(repo.createdTestimonies) != 1 {
		t.Fatalf("expected testimony persisted")
	}
}

func TestSubmitTestimonyRejectsInvalidProof(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prove":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"proof_b64":"cHJvb2Y="}`))
		case "/verify":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"valid":false,"reason":"not a citizen"}`))
		}
	}))
	defer ts.Close()

	repo := &fakeRepo{}
	svc := New(repo, ts.URL)

	_, _, _, err := svc.SubmitTestimony(context.Background(), []byte("citizenship-proof"), []byte("encrypted"), "human-rights", "fa")
	if err == nil {
		t.Fatalf("expected invalid proof error")
	}
	if len(repo.createdTestimonies) != 0 {
		t.Fatalf("expected no testimony persisted for invalid proof")
	}
}

func TestSubmitTestimonyZKUnavailable(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := New(repo, "http://127.0.0.1:1")

	_, _, _, err := svc.SubmitTestimony(context.Background(), []byte("citizenship-proof"), []byte("encrypted"), "human-rights", "fa")
	if err == nil {
		t.Fatalf("expected zk unavailable error")
	}
}
