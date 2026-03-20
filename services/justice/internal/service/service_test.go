package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	createdAmnesty     []repository.AmnestyRecord
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
	f.createdAmnesty = append(f.createdAmnesty, rec)
	if f.createAmnestyFn != nil {
		return f.createAmnestyFn(ctx, rec)
	}
	return nil
}

func (f *fakeRepo) UpdateCaseStatus(_ context.Context, _ string, _ string) error {
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

func TestFullJusticeFlowSubmitLinkAndAmnesty(t *testing.T) {
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
	repo.getByReceiptFn = func(ctx context.Context, receiptToken string) (*repository.TestimonyRecord, error) {
		for _, rec := range repo.createdTestimonies {
			if rec.ReceiptToken == receiptToken {
				copy := rec
				return &copy, nil
			}
		}
		return nil, repository.ErrNotFound
	}

	svc := New(repo, ts.URL)

	receipt, caseID, _, err := svc.SubmitTestimony(
		context.Background(),
		[]byte("citizenship-zk-proof"),
		[]byte("encrypted-testimony"),
		"human-rights",
		"fa",
	)
	if err != nil {
		t.Fatalf("submit testimony failed: %v", err)
	}
	if receipt == "" || caseID == "" {
		t.Fatalf("expected non-empty receipt and case id")
	}

	linkedCaseID, _, err := svc.LinkTestimony(
		context.Background(),
		receipt,
		[]byte("encrypted-follow-up"),
		"fa",
	)
	if err != nil {
		t.Fatalf("link testimony failed: %v", err)
	}
	if linkedCaseID != caseID {
		t.Fatalf("expected linked case %s, got %s", caseID, linkedCaseID)
	}
	if len(repo.createdTestimonies) != 2 {
		t.Fatalf("expected two testimonies persisted, got %d", len(repo.createdTestimonies))
	}

	amnestyCaseID, amnestyReceipt, _, err := svc.InitiateAmnesty(
		context.Background(),
		"did:indis:citizen-1",
		[]byte("encrypted-declaration"),
		"restorative-justice",
	)
	if err != nil {
		t.Fatalf("initiate amnesty failed: %v", err)
	}
	if amnestyCaseID == "" || amnestyReceipt == "" {
		t.Fatalf("expected non-empty amnesty case and receipt")
	}
	if len(repo.createdAmnesty) != 1 {
		t.Fatalf("expected one amnesty record persisted, got %d", len(repo.createdAmnesty))
	}
	if repo.createdAmnesty[0].ApplicantDID != "did:indis:citizen-1" {
		t.Fatalf("unexpected amnesty applicant DID: %s", repo.createdAmnesty[0].ApplicantDID)
	}
}

func TestGetCaseStatusByReceiptResolvesCaseID(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	repo.getByReceiptFn = func(ctx context.Context, receiptToken string) (*repository.TestimonyRecord, error) {
		if receiptToken == "rcpt-123" {
			return &repository.TestimonyRecord{CaseID: "jt_case_123"}, nil
		}
		return nil, repository.ErrNotFound
	}
	repo.getCaseStatusFn = func(ctx context.Context, caseID string) (string, time.Time, error) {
		if caseID == "jt_case_123" {
			return "received", time.Date(2026, time.March, 19, 10, 0, 0, 0, time.UTC), nil
		}
		return "", time.Time{}, repository.ErrNotFound
	}

	svc := New(repo, "")

	resolvedCaseID, status, updatedAt, err := svc.GetCaseStatus(context.Background(), "", "rcpt-123")
	if err != nil {
		t.Fatalf("get case status failed: %v", err)
	}
	if resolvedCaseID != "jt_case_123" {
		t.Fatalf("expected resolved case id jt_case_123, got %s", resolvedCaseID)
	}
	if status != "received" {
		t.Fatalf("expected status received, got %s", status)
	}
	if !strings.HasSuffix(updatedAt, "Z") {
		t.Fatalf("expected RFC3339 UTC timestamp, got %s", updatedAt)
	}
}

func TestInitiateAmnestyRequiresApplicantDID(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := New(repo, "")

	_, _, _, err := svc.InitiateAmnesty(
		context.Background(),
		"",
		[]byte("encrypted-declaration"),
		"restorative-justice",
	)
	if err == nil {
		t.Fatalf("expected applicant DID validation error")
	}
	if len(repo.createdAmnesty) != 0 {
		t.Fatalf("expected no amnesty record to be created")
	}
}
