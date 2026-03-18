// Package service implements business logic for the electoral service.
// ZK-STARK proof verification is delegated to services/zkproof via gRPC.
// Ref: PRD §FR-010
package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/repository"
)

// ElectoralService manages elections and ZK-verified ballots.
type ElectoralService struct {
	repo       ElectoralRepository
	zkVerifier ZKVerifier
}

// ElectoralRepository defines the repository behavior used by the service.
type ElectoralRepository interface {
	CreateElection(ctx context.Context, rec repository.ElectionRecord) error
	GetElection(ctx context.Context, id string) (*repository.ElectionRecord, error)
	NullifierExists(ctx context.Context, electionID, nullifierHash string) (bool, error)
	CastBallot(ctx context.Context, rec repository.BallotRecord) error
}

// ZKVerifier defines verification behavior for voter eligibility proofs.
type ZKVerifier interface {
	VerifyEligibility(ctx context.Context, electionID string, zkProof, publicInputs []byte) (bool, string, error)
}

func New(repo ElectoralRepository, zkProofURL string) *ElectoralService {
	return &ElectoralService{
		repo:       repo,
		zkVerifier: newHTTPZKVerifier(zkProofURL),
	}
}

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
func (s *ElectoralService) VerifyEligibility(ctx context.Context, electionID string, zkProof []byte, publicInputs []byte) (bool, string, string, error) {
	if _, err := s.repo.GetElection(ctx, electionID); errors.Is(err, repository.ErrNotFound) {
		return false, "", "election not found", nil
	}

	valid, reason, err := s.zkVerifier.VerifyEligibility(ctx, electionID, zkProof, publicInputs)
	if err != nil {
		return false, "", "", fmt.Errorf("service: verify proof via zk service: %w", err)
	}
	if !valid {
		return false, "", reason, nil
	}

	// Derive a deterministic nullifier candidate from public inputs.
	h := sha256.Sum256(publicInputs)
	nullifierHash := hex.EncodeToString(h[:])
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

type zkVerifyRequest struct {
	ElectionID      string `json:"election_id"`
	ProofSystem     string `json:"proof_system"`
	ProofB64        string `json:"proof_b64"`
	PublicInputsB64 string `json:"public_inputs_b64"`
}

type zkVerifyResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

type httpZKVerifier struct {
	baseURL string
	client  *http.Client
}

func newHTTPZKVerifier(baseURL string) ZKVerifier {
	trimmed := strings.TrimRight(baseURL, "/")
	if trimmed == "" {
		return &httpZKVerifier{client: &http.Client{Timeout: 5 * time.Second}}
	}
	return &httpZKVerifier{
		baseURL: trimmed,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (v *httpZKVerifier) VerifyEligibility(ctx context.Context, electionID string, zkProof, publicInputs []byte) (bool, string, error) {
	if v.baseURL == "" {
		return false, "", fmt.Errorf("zk proof URL is not configured")
	}

	payload := zkVerifyRequest{
		ElectionID:      electionID,
		ProofSystem:     "stark",
		ProofB64:        base64.StdEncoding.EncodeToString(zkProof),
		PublicInputsB64: base64.StdEncoding.EncodeToString(publicInputs),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/verify", bytes.NewReader(body))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("zk verify status: %s", resp.Status)
	}

	var out zkVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "", err
	}

	return out.Valid, out.Reason, nil
}
