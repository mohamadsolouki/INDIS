// Package blockchain provides the abstraction layer for all blockchain interactions.
//
// No application service should call blockchain SDK directly. All blockchain
// interactions go through the BlockchainAdapter interface defined here.
//
// See INDIS PRD v1.0 §5.3 — Blockchain Abstraction Layer.
package blockchain

import (
	"context"
	"time"
)

// TxReceipt represents the result of a blockchain transaction.
type TxReceipt struct {
	TxID        string
	BlockHeight uint64
	Timestamp   time.Time
}

// DIDDocument represents a W3C DID Document stored on-chain.
type DIDDocument struct {
	ID                 string
	PublicKeys         []PublicKey
	ServiceEndpoints   []ServiceEndpoint
	Created            time.Time
	Updated            time.Time
}

// PublicKey represents a public key within a DID Document.
type PublicKey struct {
	ID           string
	Type         string
	Controller   string
	PublicKeyHex string
}

// ServiceEndpoint represents a service endpoint in a DID Document.
type ServiceEndpoint struct {
	ID              string
	Type            string
	ServiceEndpoint string
}

// Hash represents a cryptographic hash value.
type Hash [32]byte

// AnchorStatus represents the on-chain anchor status of a credential.
type AnchorStatus struct {
	Exists      bool
	IssuerDID   string
	BlockHeight uint64
	Timestamp   time.Time
}

// RevocationReason defines why a credential was revoked.
type RevocationReason string

const (
	RevocationReasonExpired     RevocationReason = "expired"
	RevocationReasonCompromised RevocationReason = "compromised"
	RevocationReasonSuperseded  RevocationReason = "superseded"
	RevocationReasonAdminAction RevocationReason = "admin_action"
)

// RevocationStatus represents the revocation state of a credential.
type RevocationStatus struct {
	Revoked   bool
	Reason    RevocationReason
	Timestamp time.Time
}

// RevocationList is a list of revoked credential IDs for an issuer.
type RevocationList struct {
	IssuerDID    string
	RevokedIDs   []string
	LastUpdated  time.Time
}

// AnonymizedVerificationEvent is an audit event with no personal data.
type AnonymizedVerificationEvent struct {
	EventID        string
	CredentialType string
	VerifierCategory string
	Result         bool
	Timestamp      time.Time
}

// ValidatorStatus represents the health of a blockchain validator node.
type ValidatorStatus struct {
	NodeID   string
	Address  string
	IsActive bool
	LastSeen time.Time
}

// BlockchainAdapter defines the interface for all blockchain interactions.
// All application services MUST use this interface — never the blockchain SDK directly.
type BlockchainAdapter interface {
	// DID Operations
	RegisterDID(ctx context.Context, did string, document DIDDocument) (*TxReceipt, error)
	ResolveDID(ctx context.Context, did string) (*DIDDocument, error)
	UpdateDIDDocument(ctx context.Context, did string, update DIDDocument) (*TxReceipt, error)
	DeactivateDID(ctx context.Context, did string) (*TxReceipt, error)

	// Credential Anchoring
	AnchorCredential(ctx context.Context, credentialHash Hash, issuerDID string) (*TxReceipt, error)
	VerifyAnchor(ctx context.Context, credentialHash Hash) (*AnchorStatus, error)

	// Revocation Registry
	RevokeCredential(ctx context.Context, credentialID string, reason RevocationReason) (*TxReceipt, error)
	CheckRevocationStatus(ctx context.Context, credentialID string) (*RevocationStatus, error)
	GetRevocationList(ctx context.Context, issuerDID string) (*RevocationList, error)

	// Audit Trail (anonymized)
	LogVerificationEvent(ctx context.Context, event AnonymizedVerificationEvent) (*TxReceipt, error)

	// Health and Status
	GetBlockHeight(ctx context.Context) (uint64, error)
	GetValidatorStatus(ctx context.Context) ([]ValidatorStatus, error)
	EstimateTxTime(ctx context.Context) (time.Duration, error)
}
