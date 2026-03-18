package vc

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"
)

// helpers

func generateKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	return pub, priv
}

func sampleSubject() CredentialSubject {
	return CredentialSubject{
		ID: "did:indis:abc123def456abc123def456abc123def456ab",
		Claims: map[string]any{
			"name": "Ali Hosseini",
		},
	}
}

// ---------------------------------------------------------------------------
// Issue + Verify round-trips for several credential types
// ---------------------------------------------------------------------------

func TestIssueVerify_RoundTrip(t *testing.T) {
	credTypes := []CredentialType{
		TypeCitizenship,
		TypeAgeRange,
		TypeVoterEligibility,
		TypeResidency,
		TypeDiaspora,
	}

	pub, priv := generateKeyPair(t)
	issuerDID := "did:indis:issuer0000000000000000000000000000000000"
	vm := issuerDID + "#key-1"
	validFrom := time.Now().UTC().Add(-time.Minute)

	for _, ct := range credTypes {
		ct := ct
		t.Run(string(ct), func(t *testing.T) {
			issued, err := Issue(ct, issuerDID, vm, sampleSubject(), validFrom, nil, priv)
			if err != nil {
				t.Fatalf("Issue(%q): %v", ct, err)
			}
			if err := Verify(issued, pub); err != nil {
				t.Errorf("Verify(%q): %v", ct, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Proof and ID not empty after Issue
// ---------------------------------------------------------------------------

func TestIssue_ProofNotEmpty(t *testing.T) {
	_, priv := generateKeyPair(t)
	issued, err := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), time.Now().UTC().Add(-time.Minute), nil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if issued.Proof == nil {
		t.Fatal("Proof must not be nil after Issue")
	}
	if issued.Proof.ProofValue == "" {
		t.Error("Proof.ProofValue must not be empty")
	}
	if issued.Proof.Type == "" {
		t.Error("Proof.Type must not be empty")
	}
}

func TestIssue_IDNotEmpty(t *testing.T) {
	_, priv := generateKeyPair(t)
	issued, err := Issue(TypeAgeRange, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), time.Now().UTC().Add(-time.Minute), nil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if issued.ID == "" {
		t.Error("VC ID must not be empty")
	}
	if !strings.HasPrefix(issued.ID, "urn:indis:vc:") {
		t.Errorf("VC ID %q should start with \"urn:indis:vc:\"", issued.ID)
	}
}

// ---------------------------------------------------------------------------
// Verify with wrong public key
// ---------------------------------------------------------------------------

func TestVerify_WrongPublicKey(t *testing.T) {
	_, priv1 := generateKeyPair(t)
	pub2, _ := generateKeyPair(t)

	issued, err := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), time.Now().UTC().Add(-time.Minute), nil, priv1)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := Verify(issued, pub2); err == nil {
		t.Error("Verify should return error when using a different public key")
	}
}

// ---------------------------------------------------------------------------
// Verify with tampered subject
// ---------------------------------------------------------------------------

func TestVerify_TamperedSubject(t *testing.T) {
	pub, priv := generateKeyPair(t)
	issued, err := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), time.Now().UTC().Add(-time.Minute), nil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	// Tamper: change the subject's claims without re-signing.
	issued.CredentialSubject.Claims["name"] = "Tampered Name"
	if err := Verify(issued, pub); err == nil {
		t.Error("Verify should return error when credential subject has been tampered with")
	}
}

// ---------------------------------------------------------------------------
// Issue with nil validUntil — must not panic
// ---------------------------------------------------------------------------

func TestIssue_NilValidUntil_DoesNotPanic(t *testing.T) {
	_, priv := generateKeyPair(t)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Issue with nil validUntil panicked: %v", r)
		}
	}()
	_, err := Issue(TypeResidency, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), time.Now().UTC().Add(-time.Minute), nil, priv)
	if err != nil {
		t.Fatalf("Issue with nil validUntil: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Issue: ValidFrom set correctly
// ---------------------------------------------------------------------------

func TestIssue_ValidFromSetCorrectly(t *testing.T) {
	_, priv := generateKeyPair(t)
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	issued, err := Issue(TypePension, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), validFrom, nil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !issued.ValidFrom.Equal(validFrom) {
		t.Errorf("ValidFrom = %v, want %v", issued.ValidFrom, validFrom)
	}
}

// ---------------------------------------------------------------------------
// Verify: no proof present returns error
// ---------------------------------------------------------------------------

func TestVerify_NoProof_Error(t *testing.T) {
	pub, _ := generateKeyPair(t)
	vc := &VerifiableCredential{
		ID:                "urn:indis:vc:test",
		Type:              []string{"VerifiableCredential", string(TypeCitizenship)},
		Issuer:            "did:indis:issuer",
		ValidFrom:         time.Now().UTC().Add(-time.Minute),
		CredentialSubject: sampleSubject(),
		Status:            StatusActive,
		Proof:             nil,
	}
	if err := Verify(vc, pub); err == nil {
		t.Error("Verify should return error when Proof is nil")
	}
}

// ---------------------------------------------------------------------------
// Verify: revoked credential returns error
// ---------------------------------------------------------------------------

func TestVerify_RevokedCredential_Error(t *testing.T) {
	pub, priv := generateKeyPair(t)
	issued, err := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), time.Now().UTC().Add(-time.Minute), nil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	issued.Status = StatusRevoked
	if err := Verify(issued, pub); err == nil {
		t.Error("Verify should return error for a revoked credential")
	}
}

// ---------------------------------------------------------------------------
// Verify: expired credential returns error
// ---------------------------------------------------------------------------

func TestVerify_ExpiredCredential_Error(t *testing.T) {
	pub, priv := generateKeyPair(t)
	validFrom := time.Now().UTC().Add(-2 * time.Hour)
	validUntil := time.Now().UTC().Add(-time.Hour) // already expired
	issued, err := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), validFrom, &validUntil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := Verify(issued, pub); err == nil {
		t.Error("Verify should return error for an expired credential")
	}
}

// ---------------------------------------------------------------------------
// Issue with validUntil set — round-trip verification succeeds
// ---------------------------------------------------------------------------

func TestIssueVerify_WithValidUntil(t *testing.T) {
	pub, priv := generateKeyPair(t)
	validFrom := time.Now().UTC().Add(-time.Minute)
	validUntil := time.Now().UTC().Add(time.Hour)
	issued, err := Issue(TypeHealthInsurance, "did:indis:issuer", "did:indis:issuer#key-1",
		sampleSubject(), validFrom, &validUntil, priv)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if issued.ValidUntil == nil {
		t.Fatal("ValidUntil should be set in the issued credential")
	}
	if err := Verify(issued, pub); err != nil {
		t.Errorf("Verify with valid validUntil: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Issue: two consecutive calls produce different IDs
// ---------------------------------------------------------------------------

func TestIssue_UniqueIDs(t *testing.T) {
	_, priv := generateKeyPair(t)
	validFrom := time.Now().UTC().Add(-time.Minute)
	vc1, err1 := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1", sampleSubject(), validFrom, nil, priv)
	vc2, err2 := Issue(TypeCitizenship, "did:indis:issuer", "did:indis:issuer#key-1", sampleSubject(), validFrom, nil, priv)
	if err1 != nil || err2 != nil {
		t.Fatalf("Issue errors: %v, %v", err1, err2)
	}
	if vc1.ID == vc2.ID {
		t.Error("two consecutive Issue calls should produce different credential IDs")
	}
}
