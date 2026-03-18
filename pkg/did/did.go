// Package did provides utilities for W3C DID Core 1.0 operations.
//
// INDIS generates DIDs conforming to the W3C DID Core 1.0 specification.
// Private keys are generated ON-DEVICE only — government servers never hold
// citizen private keys (PRD §FR-001.4).
//
// DID Method: did:indis:<hex(sha256(publicKey)[:20])>
//
// See: https://www.w3.org/TR/did-core/
package did

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DIDMethodName is the INDIS DID method identifier.
const DIDMethodName = "indis"

// Key type identifiers per W3C DID Core and Linked Data Cryptography suites.
const (
	KeyTypeEd25519VerificationKey2020 = "Ed25519VerificationKey2020"
	KeyTypeEcdsaSecp256r1             = "EcdsaSecp256r1VerificationKey2019"
)

// DID is a W3C Decentralized Identifier string.
// Format: did:indis:<method-specific-id>
type DID string

// String returns the DID as a plain string.
func (d DID) String() string { return string(d) }

// MethodSpecificID returns the portion after "did:indis:".
func (d DID) MethodSpecificID() string {
	parts := strings.SplitN(string(d), ":", 3)
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

// Validate checks that d follows the did:indis syntax.
func (d DID) Validate() error {
	s := string(d)
	if !strings.HasPrefix(s, "did:indis:") {
		return fmt.Errorf("did: invalid prefix, expected did:indis: got %q", s)
	}
	id := d.MethodSpecificID()
	if len(id) == 0 {
		return errors.New("did: empty method-specific id")
	}
	if _, err := hex.DecodeString(id); err != nil {
		return fmt.Errorf("did: method-specific id is not hex: %w", err)
	}
	return nil
}

// FromPublicKey derives a deterministic DID from a 32-byte Ed25519 public key.
// The method-specific ID is the first 20 bytes of SHA-256(publicKey), hex-encoded (40 chars).
// Ref: PRD §FR-001.1
func FromPublicKey(publicKey []byte) (DID, error) {
	if len(publicKey) == 0 {
		return "", errors.New("did: public key must not be empty")
	}
	hash := sha256.Sum256(publicKey)
	id := hex.EncodeToString(hash[:20])
	return DID("did:indis:" + id), nil
}

// Parse parses and validates a DID string.
func Parse(s string) (DID, error) {
	d := DID(s)
	if err := d.Validate(); err != nil {
		return "", err
	}
	return d, nil
}

// VerificationMethod represents a public key in a DID Document.
// Ref: W3C DID Core 1.0 §5.2.1
type VerificationMethod struct {
	// ID is the key identifier, e.g. did:indis:abc123#key-1
	ID string
	// Type is the cryptographic suite, e.g. Ed25519VerificationKey2020
	Type string
	// Controller is the DID that controls this key (usually the subject DID)
	Controller string
	// PublicKeyMultibase is the public key encoded in multibase (base64url without padding)
	PublicKeyMultibase string
}

// Service represents a service endpoint in a DID Document.
// Ref: W3C DID Core 1.0 §5.4
type Service struct {
	ID              string
	Type            string
	ServiceEndpoint string
}

// Document is a W3C DID Document.
// Ref: https://www.w3.org/TR/did-core/#did-documents
type Document struct {
	// Context is always []string{"https://www.w3.org/ns/did/v1"}
	Context             []string
	ID                  DID
	VerificationMethods []VerificationMethod
	Authentication      []string // key IDs that may authenticate
	AssertionMethod     []string // key IDs that may assert claims
	Services            []Service
	Created             time.Time
	Updated             time.Time
	Deactivated         bool
}

// NewDocument creates a minimal DID Document for the given DID and public key.
// The document contains a single Ed25519VerificationKey2020 method.
func NewDocument(did DID, publicKeyBytes []byte) *Document {
	now := time.Now().UTC()
	keyID := did.String() + "#key-1"
	return &Document{
		Context:             []string{"https://www.w3.org/ns/did/v1"},
		ID:                  did,
		VerificationMethods: []VerificationMethod{
			{
				ID:                 keyID,
				Type:               KeyTypeEd25519VerificationKey2020,
				Controller:         did.String(),
				PublicKeyMultibase: "z" + hex.EncodeToString(publicKeyBytes),
			},
		},
		Authentication:  []string{keyID},
		AssertionMethod: []string{keyID},
		Created:         now,
		Updated:         now,
		Deactivated:     false,
	}
}

// AddService adds a service endpoint to the document.
func (doc *Document) AddService(svc Service) {
	doc.Services = append(doc.Services, svc)
	doc.Updated = time.Now().UTC()
}

// Deactivate marks the document as deactivated.
func (doc *Document) Deactivate() {
	doc.Deactivated = true
	doc.Updated = time.Now().UTC()
}
