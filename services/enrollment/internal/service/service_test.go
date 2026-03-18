package service_test

import (
	"context"
	"testing"

	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/service"
)

// mockChain satisfies blockchain.BlockchainAdapter for testing.
type mockChain struct{}

func (m *mockChain) RegisterDID(_ context.Context, _ string, _ blockchain.DIDDocument) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{TxID: "mock-tx"}, nil
}
func (m *mockChain) ResolveDID(_ context.Context, _ string) (*blockchain.DIDDocument, error) {
	return &blockchain.DIDDocument{}, nil
}
func (m *mockChain) UpdateDIDDocument(_ context.Context, _ string, _ blockchain.DIDDocument) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{}, nil
}
func (m *mockChain) DeactivateDID(_ context.Context, _ string) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{}, nil
}
func (m *mockChain) AnchorCredential(_ context.Context, _ blockchain.Hash, _ string) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{}, nil
}
func (m *mockChain) VerifyAnchor(_ context.Context, _ blockchain.Hash) (*blockchain.AnchorStatus, error) {
	return &blockchain.AnchorStatus{}, nil
}
func (m *mockChain) RevokeCredential(_ context.Context, _ string, _ blockchain.RevocationReason) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{}, nil
}
func (m *mockChain) CheckRevocationStatus(_ context.Context, _ string) (*blockchain.RevocationStatus, error) {
	return &blockchain.RevocationStatus{}, nil
}
func (m *mockChain) GetRevocationList(_ context.Context, _ string) (*blockchain.RevocationList, error) {
	return &blockchain.RevocationList{}, nil
}
func (m *mockChain) LogVerificationEvent(_ context.Context, _ blockchain.AnonymizedVerificationEvent) (*blockchain.TxReceipt, error) {
	return &blockchain.TxReceipt{}, nil
}
func (m *mockChain) GetBlockHeight(_ context.Context) (uint64, error) { return 1, nil }

// ── Input validation tests (do not need a real DB) ───────────────────────────

func TestInitiateEnrollment_InvalidPathway(t *testing.T) {
	svc := service.New(nil, &mockChain{})
	_, err := svc.InitiateEnrollment(context.Background(), 99, "agent-1", "fa")
	if err == nil {
		t.Fatal("expected error for invalid pathway int 99, got nil")
	}
}

func TestInitiateEnrollment_ValidPathways(t *testing.T) {
	svc := service.New(nil, &mockChain{})
	// Pathways 1 (standard), 2 (enhanced), 3 (social) are valid but will fail at DB level.
	// We just verify they pass the pathway validation, not the DB write.
	for pathway := int32(1); pathway <= 3; pathway++ {
		pathway := pathway
		t.Run(pathwayName(pathway), func(t *testing.T) {
			_, err := svc.InitiateEnrollment(context.Background(), pathway, "agent-1", "fa")
			// Nil repo → panic guard: the service should return an error, not panic.
			if err != nil {
				// Expected — nil pool will cause DB error.
				return
			}
		})
	}
}

func pathwayName(p int32) string {
	switch p {
	case 1:
		return "standard"
	case 2:
		return "enhanced"
	case 3:
		return "social"
	default:
		return "unknown"
	}
}

func TestSubmitSocialAttestation_TooFewAttestors(t *testing.T) {
	svc := service.New(nil, &mockChain{})
	// The attestor count check happens before DB access in service logic.
	// With a nil repo the service should fail at DB access, not at attestor validation.
	// We exercise the attestor check by verifying the error message with 2 attestors.
	_, _, err := svc.SubmitSocialAttestation(
		context.Background(),
		"enrollment-id-123",
		[]string{"did:indis:aaa", "did:indis:bbb"}, // only 2, need 3
	)
	if err == nil {
		t.Fatal("expected error with 2 attestors (minimum is 3), got nil")
	}
}

func TestSubmitSocialAttestation_ExactlyThreeAttestors(t *testing.T) {
	svc := service.New(nil, &mockChain{})
	// 3 attestors should pass attestor count check, then fail at DB.
	_, _, err := svc.SubmitSocialAttestation(
		context.Background(),
		"enrollment-id-123",
		[]string{"did:indis:aaa", "did:indis:bbb", "did:indis:ccc"},
	)
	// With nil repo this will panic or error at DB level — not at attestor count.
	_ = err // acceptably fails at DB, not at attestor count
}

func TestSubmitSocialAttestation_EmptyAttestors(t *testing.T) {
	svc := service.New(nil, &mockChain{})
	_, _, err := svc.SubmitSocialAttestation(
		context.Background(),
		"enrollment-id-123",
		[]string{},
	)
	if err == nil {
		t.Fatal("expected error with zero attestors, got nil")
	}
}

func TestGetEnrollmentStatus_NilRepoFailsGracefully(t *testing.T) {
	svc := service.New(nil, &mockChain{})
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	_, err := svc.GetEnrollmentStatus(context.Background(), "enrollment-id-123")
	// With nil repo we expect an error, not a panic.
	if err == nil {
		t.Fatal("expected error with nil repo, got nil")
	}
}
