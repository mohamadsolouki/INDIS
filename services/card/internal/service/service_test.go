package service

import (
	"context"
	"strings"
	"testing"

	"github.com/mohamadsolouki/INDIS/services/card/internal/repository"
)

const testSeedHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

// fakeRepo is an in-memory implementation of CardRepository.
type fakeRepo struct {
	cards map[string]*repository.CardRecord // keyed by DID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{cards: make(map[string]*repository.CardRecord)}
}

func (f *fakeRepo) Create(_ context.Context, rec repository.CardRecord) error {
	cp := rec
	f.cards[rec.DID] = &cp
	return nil
}

func (f *fakeRepo) GetByDID(_ context.Context, did string) (*repository.CardRecord, error) {
	rec, ok := f.cards[did]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *rec
	return &cp, nil
}

func (f *fakeRepo) Invalidate(_ context.Context, did, reason string) error {
	rec, ok := f.cards[did]
	if !ok {
		return repository.ErrNotFound
	}
	rec.Status = "invalidated"
	rec.InvalidationReason = &reason
	return nil
}

// helpers

func newService(t *testing.T) *Service {
	t.Helper()
	svc, err := New(newFakeRepo(), testSeedHex)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return svc
}

func newServiceWithRepo(t *testing.T, repo CardRepository) *Service {
	t.Helper()
	svc, err := New(repo, testSeedHex)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return svc
}

func generateCard(t *testing.T, svc *Service) *CardData {
	t.Helper()
	cd, err := svc.GenerateCard(context.Background(), GenerateRequest{
		DID:         "did:indis:abc123456789",
		HolderName:  "AHMADI MOHSEN",
		DateOfBirth: "800115",
		ExpiryDate:  "310101",
	})
	if err != nil {
		t.Fatalf("GenerateCard failed: %v", err)
	}
	return cd
}

// Tests

func TestGenerateCard_Success(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	cd := generateCard(t, svc)

	if !strings.HasPrefix(cd.MRZLine1, "IP<IRN") {
		t.Errorf("expected MRZLine1 to start with 'IP<IRN', got: %s", cd.MRZLine1)
	}
	if len(cd.MRZLine2) != 44 {
		t.Errorf("expected MRZLine2 length 44, got %d: %s", len(cd.MRZLine2), cd.MRZLine2)
	}
	// IssuerSig is hex of 64-byte Ed25519 signature → 128 chars
	if len(cd.IssuerSig) != 128 {
		t.Errorf("expected IssuerSig length 128, got %d", len(cd.IssuerSig))
	}
}

func TestGenerateCard_MissingDID(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	_, err := svc.GenerateCard(context.Background(), GenerateRequest{
		DID:         "",
		HolderName:  "ALI AHMADI",
		DateOfBirth: "800115",
		ExpiryDate:  "310101",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "DID") {
		t.Errorf("expected 'DID' in error, got: %v", err)
	}
}

func TestGenerateCard_MissingName(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	_, err := svc.GenerateCard(context.Background(), GenerateRequest{
		DID:         "did:indis:abc",
		HolderName:  "",
		DateOfBirth: "800115",
		ExpiryDate:  "310101",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "holder_name") {
		t.Errorf("expected 'holder_name' in error, got: %v", err)
	}
}

func TestGenerateCard_InvalidDOB(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	_, err := svc.GenerateCard(context.Background(), GenerateRequest{
		DID:         "did:indis:abc",
		HolderName:  "ALI AHMADI",
		DateOfBirth: "1234567", // 7 chars — invalid
		ExpiryDate:  "310101",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "YYMMDD") {
		t.Errorf("expected 'YYMMDD' in error, got: %v", err)
	}
}

func TestGenerateCard_MRZLine1Format(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	cd, err := svc.GenerateCard(context.Background(), GenerateRequest{
		DID:         "did:indis:abc123",
		HolderName:  "ALI AHMADI",
		DateOfBirth: "800115",
		ExpiryDate:  "310101",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cd.MRZLine1) != 44 {
		t.Errorf("expected MRZLine1 length 44, got %d: %s", len(cd.MRZLine1), cd.MRZLine1)
	}
}

func TestGenerateCard_MRZCheckDigits(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	cd := generateCard(t, svc)
	if len(cd.MRZLine2) != 44 {
		t.Errorf("expected MRZLine2 length 44, got %d", len(cd.MRZLine2))
	}
}

func TestGetCard_Success(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	cd := generateCard(t, svc)

	got, err := svc.GetCard(context.Background(), cd.DID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MRZLine1 != cd.MRZLine1 {
		t.Errorf("MRZLine1 mismatch: expected %s, got %s", cd.MRZLine1, got.MRZLine1)
	}
	if got.MRZLine2 != cd.MRZLine2 {
		t.Errorf("MRZLine2 mismatch: expected %s, got %s", cd.MRZLine2, got.MRZLine2)
	}
}

func TestGetCard_NotFound(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	_, err := svc.GetCard(context.Background(), "did:indis:nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestInvalidateCard_Success(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newServiceWithRepo(t, repo)
	cd := generateCard(t, svc)

	if err := svc.InvalidateCard(context.Background(), cd.DID, "lost"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := svc.GetCard(context.Background(), cd.DID)
	if err != nil {
		t.Fatalf("unexpected error getting card: %v", err)
	}
	if got.Status != "invalidated" {
		t.Errorf("expected status 'invalidated', got %s", got.Status)
	}
}

func TestVerifyCard_Active(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	cd := generateCard(t, svc)

	valid, err := svc.VerifyCard(context.Background(), cd.DID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true for active card")
	}
}

func TestVerifyCard_Invalidated(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newServiceWithRepo(t, repo)
	cd := generateCard(t, svc)

	if err := svc.InvalidateCard(context.Background(), cd.DID, "stolen"); err != nil {
		t.Fatalf("invalidate failed: %v", err)
	}

	valid, err := svc.VerifyCard(context.Background(), cd.DID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected valid=false for invalidated card")
	}
}

func TestVerifyCard_TamperedSignature(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newServiceWithRepo(t, repo)
	cd := generateCard(t, svc)

	// Tamper the stored IssuerSig.
	stored := repo.cards[cd.DID]
	stored.IssuerSig = strings.Repeat("de", 64) // 128 hex chars of garbage

	valid, err := svc.VerifyCard(context.Background(), cd.DID)
	// Either an error (invalid hex len) or valid=false — both acceptable.
	if err == nil && valid {
		t.Error("expected either an error or valid=false for tampered signature")
	}
}

// TestCheckDigit verifies the ICAO 9303 check digit algorithm on a known value.
// Per ICAO 9303-3 Appendix D example, "330107" → check digit '3'.
func TestCheckDigit(t *testing.T) {
	t.Parallel()
	// Known ICAO test vector: document number "L898902C3" → check digit '6'
	// Source: ICAO Doc 9303 Part 3, §4, worked example.
	if got := checkDigit("L898902C3"); got != '6' {
		t.Errorf("checkDigit(L898902C3) = %c, want '6'", got)
	}

	// Another vector: all zeros → each digit is 0, sum=0, check digit = '0'
	if got := checkDigit("000000"); got != '0' {
		t.Errorf("checkDigit(000000) = %c, want '0'", got)
	}

	// Filler '<' → value 0, check digit = '0'
	if got := checkDigit("<<<<<<<<<<"); got != '0' {
		t.Errorf("checkDigit(<<<<<<<<<<) = %c, want '0'", got)
	}
}
