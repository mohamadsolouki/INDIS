// Package service implements business logic for the identity service.
package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/pkg/did"
	"github.com/IranProsperityProject/INDIS/services/identity/internal/repository"
)

// RegisterResult holds the outcome of a successful DID registration.
type RegisterResult struct {
	TxID        string
	BlockHeight uint64
}

// ResolveResult holds a resolved DID Document.
type ResolveResult struct {
	Record *repository.DIDRecord
}

// DeactivateResult holds the outcome of a successful DID deactivation.
type DeactivateResult struct {
	TxID string
}

// IdentityService implements business logic for DID lifecycle management.
type IdentityService struct {
	repo  *repository.Repository
	chain blockchain.BlockchainAdapter
}

// New creates an IdentityService.
func New(repo *repository.Repository, chain blockchain.BlockchainAdapter) *IdentityService {
	return &IdentityService{repo: repo, chain: chain}
}

// RegisterIdentity registers a new DID.
// It validates the DID format, stores the document in PostgreSQL, and anchors
// the DID on the blockchain. The blockchain call is best-effort — a failed
// anchor does not roll back the database write; a background reconciler
// would re-anchor on the next sweep.
func (s *IdentityService) RegisterIdentity(ctx context.Context, didStr string, doc *did.Document) (*RegisterResult, error) {
	d, err := did.Parse(didStr)
	if err != nil {
		return nil, fmt.Errorf("service: invalid DID: %w", err)
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("service: marshal document: %w", err)
	}

	// Derive public key hex from first verification method if present.
	pubKeyHex := ""
	if len(doc.VerificationMethods) > 0 {
		pubKeyHex = doc.VerificationMethods[0].PublicKeyMultibase
	}

	now := time.Now().UTC()
	rec := repository.DIDRecord{
		DID:          d.String(),
		PublicKeyHex: pubKeyHex,
		Document:     docJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
		Deactivated:  false,
	}
	if err = s.repo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: store DID: %w", err)
	}

	// Anchor on blockchain (best-effort).
	chainDoc := blockchain.DIDDocument{
		ID:      d.String(),
		Created: now,
		Updated: now,
	}
	for _, vm := range doc.VerificationMethods {
		keyBytes, _ := hex.DecodeString(vm.PublicKeyMultibase)
		chainDoc.PublicKeys = append(chainDoc.PublicKeys, blockchain.PublicKey{
			ID:           vm.ID,
			Type:         vm.Type,
			Controller:   vm.Controller,
			PublicKeyHex: hex.EncodeToString(keyBytes),
		})
	}
	receipt, err := s.chain.RegisterDID(ctx, d.String(), chainDoc)
	if err != nil {
		// Log and continue — reconciler will retry.
		_ = err
		return &RegisterResult{TxID: "", BlockHeight: 0}, nil
	}
	return &RegisterResult{TxID: receipt.TxID, BlockHeight: receipt.BlockHeight}, nil
}

// ResolveIdentity fetches the DID Document for a given DID.
func (s *IdentityService) ResolveIdentity(ctx context.Context, didStr string) (*ResolveResult, error) {
	if _, err := did.Parse(didStr); err != nil {
		return nil, fmt.Errorf("service: invalid DID: %w", err)
	}
	rec, err := s.repo.GetByDID(ctx, didStr)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: DID not found: %s", didStr)
	}
	if err != nil {
		return nil, fmt.Errorf("service: resolve: %w", err)
	}
	return &ResolveResult{Record: rec}, nil
}

// DeactivateIdentity deactivates a DID (e.g. on death or fraud detection).
// Ref: PRD §FR-001.3
func (s *IdentityService) DeactivateIdentity(ctx context.Context, didStr string, _ string) (*DeactivateResult, error) {
	if _, err := did.Parse(didStr); err != nil {
		return nil, fmt.Errorf("service: invalid DID: %w", err)
	}
	if err := s.repo.Deactivate(ctx, didStr); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("service: DID not found or already deactivated: %s", didStr)
		}
		return nil, fmt.Errorf("service: deactivate: %w", err)
	}
	receipt, err := s.chain.DeactivateDID(ctx, didStr)
	if err != nil {
		// Best-effort blockchain call — reconciler will retry.
		return &DeactivateResult{TxID: ""}, nil
	}
	return &DeactivateResult{TxID: receipt.TxID}, nil
}
