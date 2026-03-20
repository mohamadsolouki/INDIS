package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/repository"
)

// MockElectoralRepository implements ElectoralRepository for testing.
type MockElectoralRepository struct {
	elections map[string]*repository.ElectionRecord
	ballots   map[string]*repository.BallotRecord
}

func NewMockElectoralRepository() *MockElectoralRepository {
	return &MockElectoralRepository{
		elections: make(map[string]*repository.ElectionRecord),
		ballots:   make(map[string]*repository.BallotRecord),
	}
}

func (m *MockElectoralRepository) CreateElection(ctx context.Context, rec repository.ElectionRecord) error {
	m.elections[rec.ID] = &rec
	return nil
}

func (m *MockElectoralRepository) GetElection(ctx context.Context, id string) (*repository.ElectionRecord, error) {
	if el, ok := m.elections[id]; ok {
		return el, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockElectoralRepository) NullifierExists(ctx context.Context, electionID, nullifierHash string) (bool, error) {
	for _, ballot := range m.ballots {
		if ballot.ElectionID == electionID && ballot.NullifierHash == nullifierHash {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockElectoralRepository) TransportNonceExistsSince(ctx context.Context, electionID, nonceHash string, since time.Time) (bool, error) {
	_ = since
	for _, ballot := range m.ballots {
		if ballot.ElectionID == electionID && ballot.TransportNonceHash != nil && *ballot.TransportNonceHash == nonceHash {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockElectoralRepository) CastBallot(ctx context.Context, rec repository.BallotRecord) error {
	m.ballots[rec.ReceiptHash] = &rec
	return nil
}

func (m *MockElectoralRepository) UpdateElectionStatus(_ context.Context, id, newStatus string) error {
	if el, ok := m.elections[id]; ok {
		el.Status = newStatus
	}
	return nil
}

// TestElectoralServiceWithZKVerification tests the electoral service integration with ZK verification.
func TestElectoralServiceWithZKVerification(t *testing.T) {
	// Create a mock ZK server
	zkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/verify" {
			var req zkVerifyRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Always return valid for the test
			resp := zkVerifyResponse{Valid: true, Reason: "test proof verified"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer zkServer.Close()

	// Create service
	repo := NewMockElectoralRepository()
	service := New(repo, zkServer.URL)

	// Test data
	opensAt := time.Now().UTC()
	closesAt := opensAt.Add(24 * time.Hour)

	// Register an election
	regResp, err := service.RegisterElection(context.Background(), &electoralv1.RegisterElectionRequest{
		Name:     "Test Election 2026",
		OpensAt:  opensAt.Format(time.RFC3339),
		ClosesAt: closesAt.Format(time.RFC3339),
		AdminDid: "did:indis:admin123",
	})
	if err != nil {
		t.Fatalf("RegisterElection failed: %v", err)
	}

	if regResp == "" {
		t.Fatal("RegisterElection returned empty ID")
	}

	// Test VerifyEligibility with ZK proof
	publicInputsData := []byte("voter_eligibility_claim")
	proofData := []byte("dummy_proof_data_for_testing")

	valid, nullifierHash, reason, err := service.VerifyEligibility(
		context.Background(),
		regResp,
		proofData,
		publicInputsData,
	)

	if err != nil {
		t.Fatalf("VerifyEligibility failed: %v", err)
	}

	if !valid {
		t.Fatalf("VerifyEligibility returned valid=false, reason: %s", reason)
	}

	if nullifierHash == "" {
		t.Fatal("VerifyEligibility returned empty nullifier hash")
	}

	t.Logf("✓ Electoral service ZK verification test passed. Nullifier: %s", nullifierHash)
}

// MockZKProofService implements ZKProofService for testing.
type MockZKProofService struct {
	shouldProveSucceed bool
}

func (m *MockZKProofService) VerifyEligibility(ctx context.Context, electionID string, zkProof, publicInputs []byte) (bool, string, error) {
	// Always return valid for test
	return true, "mock proof verified", nil
}

// TestZKVerificationEndpointPayload verifies the exact payload format sent to /verify.
func TestZKVerificationEndpointPayload(t *testing.T) {
	var capturedRequest zkVerifyRequest

	zkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/verify" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedRequest)

			resp := zkVerifyResponse{Valid: true, Reason: "verified"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer zkServer.Close()

	repo := NewMockElectoralRepository()
	service := New(repo, zkServer.URL)

	// Create election
	opensAt := time.Now().UTC()
	closesAt := opensAt.Add(24 * time.Hour)
	registeredElectionID, _ := service.RegisterElection(context.Background(), &electoralv1.RegisterElectionRequest{
		Name:     "Payload Test Election",
		OpensAt:  opensAt.Format(time.RFC3339),
		ClosesAt: closesAt.Format(time.RFC3339),
		AdminDid: "did:indis:admin456",
	})

	// Verify eligibility
	proofData := []byte("test_proof")
	publicInputsData := []byte("public_inputs")

	service.VerifyEligibility(
		context.Background(),
		registeredElectionID,
		proofData,
		publicInputsData,
	)

	// Verify the payload sent to /verify
	expectedProofB64 := base64.StdEncoding.EncodeToString(proofData)
	expectedPublicInputsB64 := base64.StdEncoding.EncodeToString(publicInputsData)

	if capturedRequest.ProofSystem != "stark" {
		t.Errorf("Expected proof_system=stark, got %s", capturedRequest.ProofSystem)
	}
	if capturedRequest.ElectionID != registeredElectionID {
		t.Errorf("Expected election_id=%s, got %s", registeredElectionID, capturedRequest.ElectionID)
	}
	if capturedRequest.ProofB64 != expectedProofB64 {
		t.Errorf("Expected proof_b64=%s, got %s", expectedProofB64, capturedRequest.ProofB64)
	}
	if capturedRequest.PublicInputsB64 != expectedPublicInputsB64 {
		t.Errorf("Expected public_inputs_b64=%s, got %s", expectedPublicInputsB64, capturedRequest.PublicInputsB64)
	}

	t.Log("✓ ZK verification endpoint payload test passed")
}

// TestZKVerificationWithInvalidProof tests handling of invalid ZK proofs.
func TestZKVerificationWithInvalidProof(t *testing.T) {
	zkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/verify" {
			resp := zkVerifyResponse{Valid: false, Reason: "proof signature invalid"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer zkServer.Close()

	repo := NewMockElectoralRepository()
	service := New(repo, zkServer.URL)

	// Register election
	opensAt := time.Now().UTC()
	closesAt := opensAt.Add(24 * time.Hour)
	electionID, _ := service.RegisterElection(context.Background(), &electoralv1.RegisterElectionRequest{
		Name:     "Invalid Proof Test",
		OpensAt:  opensAt.Format(time.RFC3339),
		ClosesAt: closesAt.Format(time.RFC3339),
		AdminDid: "did:indis:admin789",
	})

	// Attempt verification with invalid proof
	valid, _, reason, err := service.VerifyEligibility(
		context.Background(),
		electionID,
		[]byte("invalid_proof"),
		[]byte("public_inputs"),
	)

	if valid {
		t.Fatal("Expected valid=false for invalid proof")
	}

	if reason != "proof signature invalid" {
		t.Fatalf("Expected reason='proof signature invalid', got '%s'", reason)
	}

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	t.Log("✓ Invalid proof handling test passed")
}

// TestZKVerificationWithUnavailableService tests graceful handling when ZK service is unavailable.
func TestZKVerificationWithUnavailableService(t *testing.T) {
	// Use a non-existent URL
	repo := NewMockElectoralRepository()
	service := New(repo, "http://localhost:19999")

	// Register election
	opensAt := time.Now().UTC()
	closesAt := opensAt.Add(24 * time.Hour)
	electionID, _ := service.RegisterElection(context.Background(), &electoralv1.RegisterElectionRequest{
		Name:     "Unavailable Service Test",
		OpensAt:  opensAt.Format(time.RFC3339),
		ClosesAt: closesAt.Format(time.RFC3339),
		AdminDid: "did:indis:admin999",
	})

	// Attempt verification with unavailable service
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	valid, _, _, err := service.VerifyEligibility(
		ctx,
		electionID,
		[]byte("proof"),
		[]byte("inputs"),
	)

	if err == nil {
		t.Fatal("Expected error when ZK service is unavailable")
	}

	if valid {
		t.Fatal("Expected valid=false when ZK service is unavailable")
	}

	t.Logf("✓ Unavailable service handling test passed. Error: %v", err)
}
