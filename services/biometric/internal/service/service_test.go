package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IranProsperityProject/INDIS/services/biometric/internal/repository"
)

type fakeRepo struct {
	listFn       func(ctx context.Context, enrollmentID string) ([]repository.TemplateRecord, error)
	storeFn      func(ctx context.Context, rec repository.TemplateRecord) error
	softDeleteFn func(ctx context.Context, templateID string) error
}

func (f *fakeRepo) Store(ctx context.Context, rec repository.TemplateRecord) error {
	if f.storeFn != nil {
		return f.storeFn(ctx, rec)
	}
	return nil
}

func (f *fakeRepo) ListByEnrollment(ctx context.Context, enrollmentID string) ([]repository.TemplateRecord, error) {
	if f.listFn != nil {
		return f.listFn(ctx, enrollmentID)
	}
	return nil, nil
}

func (f *fakeRepo) SoftDelete(ctx context.Context, templateID string) error {
	if f.softDeleteFn != nil {
		return f.softDeleteFn(ctx, templateID)
	}
	return nil
}

func TestCallAIDedupSuccess(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/biometric/deduplicate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_duplicate":true,"confidence":0.99,"matched_did":"did:indis:abc","deduplication_ms":"12"}`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, make([]byte, 32), ts.URL)

	isDup, did, ms, score, err := svc.callAIDedup(context.Background(), "enr-1", []byte("abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isDup {
		t.Fatalf("expected duplicate")
	}
	if did != "did:indis:abc" {
		t.Fatalf("unexpected did: %s", did)
	}
	if ms != "12" {
		t.Fatalf("unexpected dedup ms: %s", ms)
	}
	if score != 0.99 {
		t.Fatalf("unexpected confidence: %v", score)
	}
}

func TestCallAIDedupTimeout(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_duplicate":false,"confidence":0.1,"matched_did":"","deduplication_ms":"150"}`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, make([]byte, 32), ts.URL)
	svc.httpClient = &http.Client{Timeout: 50 * time.Millisecond}

	_, _, _, _, err := svc.callAIDedup(context.Background(), "enr-1", []byte("abc"))
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestCallAIDedupMalformedResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_duplicate":`))
	}))
	defer ts.Close()

	svc := New(&fakeRepo{}, make([]byte, 32), ts.URL)

	_, _, _, _, err := svc.callAIDedup(context.Background(), "enr-1", []byte("abc"))
	if err == nil {
		t.Fatalf("expected decode error, got nil")
	}
}

func TestCheckDuplicateFallsBackWhenAIUnavailable(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{
		listFn: func(_ context.Context, enrollmentID string) ([]repository.TemplateRecord, error) {
			if enrollmentID != "enr-1" {
				t.Fatalf("unexpected enrollment id: %s", enrollmentID)
			}
			return []repository.TemplateRecord{}, nil
		},
	}

	svc := New(repo, make([]byte, 32), "http://127.0.0.1:1")
	isDup, matchedDID, dedupMS, score, err := svc.CheckDuplicate(context.Background(), "enr-1", 1, []byte("abc"))
	if err != nil {
		t.Fatalf("unexpected fallback error: %v", err)
	}
	if isDup {
		t.Fatalf("expected non-duplicate fallback")
	}
	if matchedDID != "" {
		t.Fatalf("unexpected matched DID: %s", matchedDID)
	}
	if dedupMS == "" {
		t.Fatalf("expected deduplication ms to be populated")
	}
	if score != 0 {
		t.Fatalf("expected zero score in fallback path, got %v", score)
	}
}

func TestDeleteTemplateNotFound(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{
		softDeleteFn: func(context.Context, string) error {
			return repository.ErrNotFound
		},
	}
	svc := New(repo, make([]byte, 32), "")

	ok, _, err := svc.DeleteTemplate(context.Background(), "tpl-1", "actor-1")
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if ok {
		t.Fatalf("expected delete result false")
	}
	if !strings.Contains(err.Error(), "template not found") {
		t.Fatalf("expected not found message, got %v", err)
	}
}
