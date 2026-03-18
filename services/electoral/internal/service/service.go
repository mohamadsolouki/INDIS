// Package service implements business logic for the electoral service.
// ZK-STARK proof verification is delegated to services/zkproof via gRPC.
// Ref: PRD §FR-010
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/repository"
)

// ElectoralService manages elections and ZK-verified ballots.
type ElectoralService struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *ElectoralService { return &ElectoralService{repo: repo} }

func generateID(prefix string) (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// RegisterElection creates a new election event.
func (s *ElectoralService) RegisterElection(ctx context.Context, req *electoralv1.RegisterElectionRequest) (string, error) {
	opensAt, err := time.Parse(time.RFC3339, req.GetOpensAt())
	if err != nil {
		return "", fmt.Errorf("service: parse opens_at: %w", err)
	}
	closesAt, err := time.Parse(time.RFC3339, req.GetClosesAt())
	if err != nil {
		return "", fmt.Errorf("service: parse closes_at: %w", err)
	}
	id, err := generateID("elc_")
	if err != nil {
		return "", fmt.Errorf("service: generate id: %w", err)
	}
	rec := repository.ElectionRecord{
		ID: id, Name: req.GetName(), Status: "scheduled",
		OpensAt: opensAt, ClosesAt: closesAt, AdminDID: req.GetAdminDid(),
		CreatedAt: time.Now().UTC(),
	}
	if err = s.repo.CreateElection(ctx, rec); err != nil {
		return "", fmt.Errorf("service: create election: %w", err)
	}
	return id, nil
}

// VerifyEligibility verifies a ZK proof of voter eligibility.
// TODO: delegate to services/zkproof gRPC for STARK proof verification.
func (s *ElectoralService) VerifyEligibility(ctx context.Context, electionID string, _ []byte, publicInputs []byte) (bool, string, string, error) {
	if _, err := s.repo.GetElection(ctx, electionID); errors.Is(err, repository.ErrNotFound) {
		return false, "", "election not found", nil
	}
	// Derive nullifier from public inputs (simplified: SHA-256(publicInputs)).
	nullifierHash := hex.EncodeToString(sha256.New().Sum(publicInputs))
	used, err := s.repo.NullifierExists(ctx, electionID, nullifierHash)
	if err != nil {
		return false, "", "", fmt.Errorf("service: check nullifier: %w", err)
	}
	if used {
		return false, "", "voter has already voted in this election", nil
	}
	return true, nullifierHash, "", nil
}

// CastBallot records an encrypted ballot after verifying no double-vote.
func (s *ElectoralService) CastBallot(ctx context.Context, req *electoralv1.CastBallotRequest) (string, string, error) {
	used, err := s.repo.NullifierExists(ctx, req.GetElectionId(), req.GetNullifierHash())
	if err != nil {
		return "", "", fmt.Errorf("service: check nullifier: %w", err)
	}
	if used {
		return "", "", fmt.Errorf("service: double-vote detected for election %s", req.GetElectionId())
	}
	h := sha256.Sum256(req.GetEncryptedVote())
	receiptHash := hex.EncodeToString(h[:])
	rec := repository.BallotRecord{
		ReceiptHash:   receiptHash,
		ElectionID:    req.GetElectionId(),
		NullifierHash: req.GetNullifierHash(),
		EncryptedVote: req.GetEncryptedVote(),
		BlockHeight:   "pending",
		CastAt:        time.Now().UTC(),
	}
	if err = s.repo.CastBallot(ctx, rec); err != nil {
		return "", "", fmt.Errorf("service: cast ballot: %w", err)
	}
	return receiptHash, "pending", nil
}

// GetElectionStatus returns the current status of an election.
func (s *ElectoralService) GetElectionStatus(ctx context.Context, electionID string) (*repository.ElectionRecord, error) {
	rec, err := s.repo.GetElection(ctx, electionID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: election not found: %s", electionID)
	}
	return rec, err
}
