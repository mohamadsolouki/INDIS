package did

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// FromPublicKey
// ---------------------------------------------------------------------------

func TestFromPublicKey_PrefixAndFormat(t *testing.T) {
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}
	d, err := FromPublicKey(pubKey)
	if err != nil {
		t.Fatalf("FromPublicKey: %v", err)
	}
	if !strings.HasPrefix(string(d), "did:indis:") {
		t.Errorf("DID %q does not start with \"did:indis:\"", d)
	}
}

func TestFromPublicKey_Deterministic(t *testing.T) {
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i + 1)
	}
	d1, err1 := FromPublicKey(pubKey)
	d2, err2 := FromPublicKey(pubKey)
	if err1 != nil || err2 != nil {
		t.Fatalf("FromPublicKey errors: %v, %v", err1, err2)
	}
	if d1 != d2 {
		t.Errorf("FromPublicKey is not deterministic: %q != %q", d1, d2)
	}
}

func TestFromPublicKey_CorrectMethodSpecificID(t *testing.T) {
	pubKey := []byte("test public key bytes here!!")
	d, err := FromPublicKey(pubKey)
	if err != nil {
		t.Fatalf("FromPublicKey: %v", err)
	}
	hash := sha256.Sum256(pubKey)
	wantID := hex.EncodeToString(hash[:20])
	if d.MethodSpecificID() != wantID {
		t.Errorf("MethodSpecificID = %q, want %q", d.MethodSpecificID(), wantID)
	}
}

func TestFromPublicKey_EmptyKey_Error(t *testing.T) {
	_, err := FromPublicKey([]byte{})
	if err == nil {
		t.Error("FromPublicKey should return an error for empty public key")
	}
}

func TestFromPublicKey_DifferentKeys_DifferentDIDs(t *testing.T) {
	k1 := make([]byte, 32)
	k2 := make([]byte, 32)
	k2[0] = 0xFF
	d1, _ := FromPublicKey(k1)
	d2, _ := FromPublicKey(k2)
	if d1 == d2 {
		t.Error("different public keys should produce different DIDs")
	}
}

// ---------------------------------------------------------------------------
// Parse
// ---------------------------------------------------------------------------

func TestParse_ValidDID(t *testing.T) {
	// Construct a valid DID manually.
	id := strings.Repeat("ab", 20) // 40 hex chars
	raw := "did:indis:" + id
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%q): %v", raw, err)
	}
	if string(d) != raw {
		t.Errorf("Parse returned %q, want %q", d, raw)
	}
}

func TestParse_NonIndisDID_Error(t *testing.T) {
	_, err := Parse("did:web:example.com")
	if err == nil {
		t.Error("Parse should return error for non-indis DID method")
	}
}

func TestParse_TooShort_Error(t *testing.T) {
	_, err := Parse("did:indis:")
	if err == nil {
		t.Error("Parse should return error for DID with empty method-specific ID")
	}
}

func TestParse_Empty_Error(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("Parse should return error for empty string")
	}
}

func TestParse_NonHexMethodID_Error(t *testing.T) {
	_, err := Parse("did:indis:gggggggggggggggggggggggggggggggggggggggg")
	if err == nil {
		t.Error("Parse should return error for non-hex method-specific ID")
	}
}

// ---------------------------------------------------------------------------
// Validate
// ---------------------------------------------------------------------------

func TestValidate_ValidDID(t *testing.T) {
	id := strings.Repeat("cd", 20)
	d := DID("did:indis:" + id)
	if err := d.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestValidate_MissingPrefix(t *testing.T) {
	d := DID("did:example:abc123")
	if err := d.Validate(); err == nil {
		t.Error("Validate should fail for non-indis method")
	}
}

func TestValidate_EmptyMethodID(t *testing.T) {
	d := DID("did:indis:")
	if err := d.Validate(); err == nil {
		t.Error("Validate should fail for empty method-specific ID")
	}
}

func TestValidate_NonHexID(t *testing.T) {
	d := DID("did:indis:xyz!@#notvalid")
	if err := d.Validate(); err == nil {
		t.Error("Validate should fail for non-hex method-specific ID")
	}
}

// ---------------------------------------------------------------------------
// NewDocument
// ---------------------------------------------------------------------------

func TestNewDocument_IDMatchesDID(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	if doc.ID != d {
		t.Errorf("doc.ID = %q, want %q", doc.ID, d)
	}
}

func TestNewDocument_VerificationMethodsNotEmpty(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	if len(doc.VerificationMethods) == 0 {
		t.Error("NewDocument should produce at least one VerificationMethod")
	}
}

func TestNewDocument_CreatedNotZero(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	before := time.Now().UTC().Add(-time.Second)
	doc := NewDocument(d, pubKey)
	if doc.Created.IsZero() {
		t.Error("NewDocument Created must not be zero")
	}
	if doc.Created.Before(before) {
		t.Errorf("NewDocument Created (%v) is before test start (%v)", doc.Created, before)
	}
}

func TestNewDocument_AuthenticationAndAssertionMethod(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	if len(doc.Authentication) == 0 {
		t.Error("NewDocument should have at least one Authentication key ID")
	}
	if len(doc.AssertionMethod) == 0 {
		t.Error("NewDocument should have at least one AssertionMethod key ID")
	}
}

func TestNewDocument_VerificationMethodKeyIDContainsDID(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	vm := doc.VerificationMethods[0]
	if !strings.HasPrefix(vm.ID, d.String()) {
		t.Errorf("VerificationMethod ID %q should start with DID %q", vm.ID, d.String())
	}
}

func TestNewDocument_ContextSet(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	if len(doc.Context) == 0 {
		t.Error("NewDocument Context must not be empty")
	}
}

// ---------------------------------------------------------------------------
// MethodSpecificID
// ---------------------------------------------------------------------------

func TestMethodSpecificID_Valid(t *testing.T) {
	id := strings.Repeat("1a", 20)
	d := DID("did:indis:" + id)
	got := d.MethodSpecificID()
	if got != id {
		t.Errorf("MethodSpecificID = %q, want %q", got, id)
	}
}

func TestMethodSpecificID_Malformed(t *testing.T) {
	d := DID("notadid")
	got := d.MethodSpecificID()
	if got != "" {
		t.Errorf("MethodSpecificID for malformed DID = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// AddService / Deactivate
// ---------------------------------------------------------------------------

func TestAddService(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	svc := Service{ID: d.String() + "#svc-1", Type: "LinkedDomains", ServiceEndpoint: "https://example.com"}
	doc.AddService(svc)
	if len(doc.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(doc.Services))
	}
}

func TestDeactivate(t *testing.T) {
	pubKey := make([]byte, 32)
	d, _ := FromPublicKey(pubKey)
	doc := NewDocument(d, pubKey)
	if doc.Deactivated {
		t.Fatal("newly created document should not be deactivated")
	}
	doc.Deactivate()
	if !doc.Deactivated {
		t.Error("document should be deactivated after Deactivate()")
	}
}
