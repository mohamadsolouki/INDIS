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

	electoralv1 "github.com/mohamadsolouki/INDIS/api/gen/go/electoral/v1"
	"github.com/mohamadsolouki/INDIS/services/electoral/internal/repository"
)

// ElectoralService manages elections and ZK-verified ballots.
type ElectoralService struct {
	repo              ElectoralRepository
	zkVerifier        ZKVerifier
	nonceReplayWindow time.Duration
}

// ElectoralRepository defines the repository behavior used by the service.
type ElectoralRepository interface {
	CreateElection(ctx context.Context, rec repository.ElectionRecord) error
	GetElection(ctx context.Context, id string) (*repository.ElectionRecord, error)
	NullifierExists(ctx context.Context, electionID, nullifierHash string) (bool, error)
	TransportNonceExistsSince(ctx context.Context, electionID, nonceHash string, since time.Time) (bool, error)
	CastBallot(ctx context.Context, rec repository.BallotRecord) error
	UpdateElectionStatus(ctx context.Context, id, newStatus string) error
}

// ZKVerifier defines verification behavior for voter eligibility proofs.
type ZKVerifier interface {
	VerifyEligibility(ctx context.Context, electionID string, zkProof, publicInputs []byte) (bool, string, error)
}

func New(repo ElectoralRepository, zkProofURL string) *ElectoralService {
	return NewWithNonceReplayWindow(repo, zkProofURL, 60*time.Minute)
}

func NewWithNonceReplayWindow(repo ElectoralRepository, zkProofURL string, nonceReplayWindow time.Duration) *ElectoralService {
	if nonceReplayWindow <= 0 {
		nonceReplayWindow = 60 * time.Minute
	}
	return &ElectoralService{
		repo:              repo,
		zkVerifier:        newHTTPZKVerifier(zkProofURL),
		nonceReplayWindow: nonceReplayWindow,
	}
}

func generateID(prefix string) (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// computeElectionStatus derives the effective election status from stored status and timestamps.
// Returns: scheduled | open | closed | tallied
func computeElectionStatus(stored string, opensAt, closesAt time.Time) string {
	now := time.Now().UTC()
	// "tallied" is an explicit admin action; always honour it.
	if stored == "tallied" {
		return "tallied"
	}
	if now.After(closesAt) {
		return "closed"
	}
	if now.After(opensAt) {
		return "open"
	}
	return "scheduled"
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
	rec, err := s.repo.GetElection(ctx, electionID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, "", "election not found", nil
	}
	if err != nil {
		return false, "", "", fmt.Errorf("service: get election: %w", err)
	}
	effectiveStatus := computeElectionStatus(rec.Status, rec.OpensAt, rec.ClosesAt)
	if effectiveStatus != "open" {
		return false, "", fmt.Sprintf("election is not open for voting (status: %s)", effectiveStatus), nil
	}

	valid, reason, err := s.zkVerifier.VerifyEligibility(ctx, electionID, zkProof, publicInputs)
	if err != nil {
		return false, "", "", fmt.Errorf("service: verify proof via zk service: %w", err)
	}
	if !valid {
		return false, "", reason, nil
	}

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
	if len(req.GetZkProof()) == 0 {
		return "", "", fmt.Errorf("service: zk_proof is required")
	}

	rec, err := s.repo.GetElection(ctx, req.GetElectionId())
	if errors.Is(err, repository.ErrNotFound) {
		return "", "", fmt.Errorf("service: election not found: %s", req.GetElectionId())
	}
	if err != nil {
		return "", "", fmt.Errorf("service: get election: %w", err)
	}
	if effectiveStatus := computeElectionStatus(rec.Status, rec.OpensAt, rec.ClosesAt); effectiveStatus != "open" {
		return "", "", fmt.Errorf("service: election is not open for voting (status: %s)", effectiveStatus)
	}

	valid, reason, err := s.zkVerifier.VerifyEligibility(ctx, req.GetElectionId(), req.GetZkProof(), []byte(req.GetNullifierHash()))
	if err != nil {
		return "", "", fmt.Errorf("service: verify ballot proof via zk service: %w", err)
	}
	if !valid {
		return "", "", fmt.Errorf("service: invalid ballot proof: %s", reason)
	}

	used, err := s.repo.NullifierExists(ctx, req.GetElectionId(), req.GetNullifierHash())
	if err != nil {
		return "", "", fmt.Errorf("service: check nullifier: %w", err)
	}
	if used {
		return "", "", fmt.Errorf("service: double-vote detected for election %s", req.GetElectionId())
	}
	h := sha256.Sum256(req.GetEncryptedVote())
	receiptHash := hex.EncodeToString(h[:])
	ballotRec := repository.BallotRecord{
		ReceiptHash:   receiptHash,
		ElectionID:    req.GetElectionId(),
		NullifierHash: req.GetNullifierHash(),
		EncryptedVote: req.GetEncryptedVote(),
		ZKProof:       req.GetZkProof(),
		BlockHeight:   "pending",
		CastAt:        time.Now().UTC(),
	}
	if err = s.repo.CastBallot(ctx, ballotRec); err != nil {
		return "", "", fmt.Errorf("service: cast ballot: %w", err)
	}
	return receiptHash, "pending", nil
}

// SubmitRemoteBallot records a remote ballot after validating integrity metadata.
func (s *ElectoralService) SubmitRemoteBallot(ctx context.Context, req *electoralv1.SubmitRemoteBallotRequest) (string, string, string, error) {
	if req.GetElectionId() == "" || req.GetNullifierHash() == "" {
		return "", "", "", fmt.Errorf("service: election_id and nullifier_hash are required")
	}
	if len(req.GetEncryptedVote()) == 0 {
		return "", "", "", fmt.Errorf("service: encrypted_vote is required")
	}

	elecRec, err := s.repo.GetElection(ctx, req.GetElectionId())
	if errors.Is(err, repository.ErrNotFound) {
		return "", "", "", fmt.Errorf("service: election not found: %s", req.GetElectionId())
	}
	if err != nil {
		return "", "", "", fmt.Errorf("service: get election: %w", err)
	}
	if effectiveStatus := computeElectionStatus(elecRec.Status, elecRec.OpensAt, elecRec.ClosesAt); effectiveStatus != "open" {
		return "", "", "", fmt.Errorf("service: election is not open for remote voting (status: %s)", effectiveStatus)
	}

	clientSubmittedAt, err := time.Parse(time.RFC3339, req.GetSubmittedAt())
	if err != nil {
		return "", "", "", fmt.Errorf("service: invalid submitted_at: %w", err)
	}
	now := time.Now().UTC()
	if clientSubmittedAt.After(now.Add(2 * time.Minute)) {
		return "", "", "", fmt.Errorf("service: remote ballot timestamp is too far in the future")
	}

	// Basic replay window control for remote submissions.
	if now.Sub(clientSubmittedAt) > 10*time.Minute {
		return "", "", "", fmt.Errorf("service: remote ballot timestamp expired")
	}
	if len(req.GetClientAttestation()) == 0 || len(req.GetTransportNonce()) == 0 {
		return "", "", "", fmt.Errorf("service: client_attestation and transport_nonce are required")
	}

	integrityHash := sha256.Sum256(bytes.Join([][]byte{
		req.GetClientAttestation(),
		req.GetTransportNonce(),
		[]byte(req.GetSubmittedAt()),
		[]byte(req.GetNetwork()),
	}, []byte("|")))

	remotePayload := append([]byte{}, req.GetEncryptedVote()...)
	remotePayload = append(remotePayload, integrityHash[:]...)

	attestationHash := sha256.Sum256(req.GetClientAttestation())
	attestationHashHex := hex.EncodeToString(attestationHash[:])
	nonceHash := sha256.Sum256(req.GetTransportNonce())
	nonceHashHex := hex.EncodeToString(nonceHash[:])
	network := req.GetNetwork()
	acceptedAt := now

	nonceCutoff := acceptedAt.Add(-s.nonceReplayWindow)
	nonceUsed, err := s.repo.TransportNonceExistsSince(ctx, req.GetElectionId(), nonceHashHex, nonceCutoff)
	if err != nil {
		return "", "", "", fmt.Errorf("service: check transport nonce: %w", err)
	}
	if nonceUsed {
		return "", "", "", fmt.Errorf("service: replayed remote ballot nonce for election %s", req.GetElectionId())
	}

	used, err := s.repo.NullifierExists(ctx, req.GetElectionId(), req.GetNullifierHash())
	if err != nil {
		return "", "", "", fmt.Errorf("service: check nullifier: %w", err)
	}
	if used {
		return "", "", "", fmt.Errorf("service: double-vote detected for election %s", req.GetElectionId())
	}

	if len(req.GetZkProof()) == 0 {
		return "", "", "", fmt.Errorf("service: zk_proof is required")
	}
	valid, reason, err := s.zkVerifier.VerifyEligibility(ctx, req.GetElectionId(), req.GetZkProof(), []byte(req.GetNullifierHash()))
	if err != nil {
		return "", "", "", fmt.Errorf("service: verify ballot proof via zk service: %w", err)
	}
	if !valid {
		return "", "", "", fmt.Errorf("service: invalid ballot proof: %s", reason)
	}

	receiptDigest := sha256.Sum256(remotePayload)
	receiptHash := hex.EncodeToString(receiptDigest[:])
	rec := repository.BallotRecord{
		ReceiptHash:           receiptHash,
		ElectionID:            req.GetElectionId(),
		NullifierHash:         req.GetNullifierHash(),
		EncryptedVote:         remotePayload,
		ZKProof:               req.GetZkProof(),
		BlockHeight:           "pending",
		RemoteNetwork:         &network,
		ClientAttestationHash: &attestationHashHex,
		TransportNonceHash:    &nonceHashHex,
		ClientSubmittedAt:     &clientSubmittedAt,
		AcceptedAt:            &acceptedAt,
		CastAt:                acceptedAt,
	}
	if err := s.repo.CastBallot(ctx, rec); err != nil {
		return "", "", "", fmt.Errorf("service: cast ballot: %w", err)
	}

	return receiptHash, "pending", acceptedAt.Format(time.RFC3339), nil
}

// GetElectionStatus returns the current status of an election.
func (s *ElectoralService) GetElectionStatus(ctx context.Context, electionID string) (*repository.ElectionRecord, error) {
	rec, err := s.repo.GetElection(ctx, electionID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: election not found: %s", electionID)
	}
	if err != nil {
		return nil, err
	}
	// Compute effective status from timestamps; only "tallied" uses stored status.
	rec.Status = computeElectionStatus(rec.Status, rec.OpensAt, rec.ClosesAt)
	return rec, nil
}

// FinalizeElection transitions a closed election to tallied state.
// Only callable once voting has closed. Ref: PRD §FR-010.
func (s *ElectoralService) FinalizeElection(ctx context.Context, electionID, adminDID string) error {
	rec, err := s.repo.GetElection(ctx, electionID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("service: election not found: %s", electionID)
	}
	if err != nil {
		return fmt.Errorf("service: get election: %w", err)
	}
	effective := computeElectionStatus(rec.Status, rec.OpensAt, rec.ClosesAt)
	if effective != "closed" {
		return fmt.Errorf("service: can only finalize a closed election (current status: %s)", effective)
	}
	if err := s.repo.UpdateElectionStatus(ctx, electionID, "tallied"); err != nil {
		return fmt.Errorf("service: update election status to tallied: %w", err)
	}
	return nil
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
