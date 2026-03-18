// Package vc provides utilities for W3C Verifiable Credentials 2.0 operations.
//
// INDIS supports 11 credential types (PRD §FR-002):
//   - Citizenship, Age Range, Voter Eligibility, Residency
//   - Professional, Health Insurance, Pension
//   - Security Clearance, Amnesty Applicant, Diaspora
//   - Social Attestation
//
// All credential types support selective disclosure via ZK proofs.
//
// See: https://www.w3.org/TR/vc-data-model-2.0/
package vc

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CredentialType identifies the kind of verifiable credential.
// Ref: PRD §FR-002
type CredentialType string

const (
	TypeCitizenship      CredentialType = "CitizenshipCredential"
	TypeAgeRange         CredentialType = "AgeRangeCredential"
	TypeVoterEligibility CredentialType = "VoterEligibilityCredential"
	TypeResidency        CredentialType = "ResidencyCredential"
	TypeProfessional     CredentialType = "ProfessionalCredential"
	TypeHealthInsurance  CredentialType = "HealthInsuranceCredential"
	TypePension          CredentialType = "PensionCredential"
	TypeSecurityClearnce CredentialType = "SecurityClearanceCredential"
	TypeAmnestyApplicant CredentialType = "AmnestyApplicantCredential"
	TypeDiaspora         CredentialType = "DiasporaCredential"
	TypeSocialAttestation CredentialType = "SocialAttestationCredential"
)

// CredentialStatus represents the lifecycle state of a credential.
type CredentialStatus string

const (
	StatusActive  CredentialStatus = "active"
	StatusRevoked CredentialStatus = "revoked"
	StatusExpired CredentialStatus = "expired"
)

// CredentialSubject holds the claims about the credential subject.
// The fields are kept generic (map) to support all 11 credential types.
type CredentialSubject struct {
	// ID is the DID of the credential subject.
	ID string `json:"id"`
	// Claims holds type-specific claim key-value pairs.
	Claims map[string]any `json:"claims,omitempty"`
}

// Proof holds the cryptographic proof of a VC.
// Ref: W3C VC Data Model 2.0 §4.10
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	VerificationMethod string `json:"verificationMethod"`
	ProofPurpose       string `json:"proofPurpose"`
	ProofValue         string `json:"proofValue"` // base64url-encoded signature
}

// VerifiableCredential is a W3C Verifiable Credential 2.0.
// Ref: https://www.w3.org/TR/vc-data-model-2.0/#credentials
type VerifiableCredential struct {
	Context           []string          `json:"@context"`
	ID                string            `json:"id"`
	Type              []string          `json:"type"`
	Issuer            string            `json:"issuer"`
	ValidFrom         time.Time         `json:"validFrom"`
	ValidUntil        *time.Time        `json:"validUntil,omitempty"`
	CredentialSubject CredentialSubject `json:"credentialSubject"`
	Status            CredentialStatus  `json:"status"`
	Proof             *Proof            `json:"proof,omitempty"`
}

// generateID produces a random 16-byte URL-safe credential ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "urn:indis:vc:" + base64.RawURLEncoding.EncodeToString(b), nil
}

// Issue creates and signs a new VerifiableCredential.
// issuerDID is the DID of the issuing authority.
// verificationMethod is the key ID used for signing, e.g. did:indis:abc#key-1.
// privateKey is the Ed25519 private key (64 bytes).
// Ref: W3C VC Data Model 2.0 §4
func Issue(
	credType CredentialType,
	issuerDID string,
	verificationMethod string,
	subject CredentialSubject,
	validFrom time.Time,
	validUntil *time.Time,
	privateKey ed25519.PrivateKey,
) (*VerifiableCredential, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("vc: id generation: %w", err)
	}

	vc := &VerifiableCredential{
		Context:           []string{"https://www.w3.org/2018/credentials/v1"},
		ID:                id,
		Type:              []string{"VerifiableCredential", string(credType)},
		Issuer:            issuerDID,
		ValidFrom:         validFrom.UTC(),
		ValidUntil:        validUntil,
		CredentialSubject: subject,
		Status:            StatusActive,
	}

	payload, err := signingPayload(vc)
	if err != nil {
		return nil, fmt.Errorf("vc: payload serialization: %w", err)
	}

	sig := ed25519.Sign(privateKey, payload)
	vc.Proof = &Proof{
		Type:               "Ed25519Signature2020",
		Created:            time.Now().UTC().Format(time.RFC3339),
		VerificationMethod: verificationMethod,
		ProofPurpose:       "assertionMethod",
		ProofValue:         base64.RawURLEncoding.EncodeToString(sig),
	}
	return vc, nil
}

// Verify checks the cryptographic proof on vc against publicKey.
// Returns nil if valid; a descriptive error otherwise.
func Verify(vc *VerifiableCredential, publicKey ed25519.PublicKey) error {
	if vc.Proof == nil {
		return errors.New("vc: no proof present")
	}
	if vc.Status == StatusRevoked {
		return errors.New("vc: credential is revoked")
	}
	now := time.Now().UTC()
	if now.Before(vc.ValidFrom) {
		return errors.New("vc: credential not yet valid")
	}
	if vc.ValidUntil != nil && now.After(*vc.ValidUntil) {
		return errors.New("vc: credential has expired")
	}

	sig, err := base64.RawURLEncoding.DecodeString(vc.Proof.ProofValue)
	if err != nil {
		return fmt.Errorf("vc: invalid proof value encoding: %w", err)
	}

	// Verify against the credential body without the proof field.
	withoutProof := *vc
	withoutProof.Proof = nil
	payload, err := signingPayload(&withoutProof)
	if err != nil {
		return fmt.Errorf("vc: payload serialization: %w", err)
	}

	if !ed25519.Verify(publicKey, payload, sig) {
		return errors.New("vc: signature verification failed")
	}
	return nil
}

// signingPayload returns the canonical JSON bytes of the credential for signing.
// The proof field must be nil before calling this.
func signingPayload(vc *VerifiableCredential) ([]byte, error) {
	return json.Marshal(vc)
}
