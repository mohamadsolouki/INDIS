// Package handler implements gRPC handlers for the electoral service.
package handler

import (
	"context"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ElectoralHandler implements electoralv1.ElectoralServiceServer.
type ElectoralHandler struct {
	electoralv1.UnimplementedElectoralServiceServer
	svc *service.ElectoralService
}

func New(svc *service.ElectoralService) *ElectoralHandler { return &ElectoralHandler{svc: svc} }

func (h *ElectoralHandler) RegisterElection(ctx context.Context, req *electoralv1.RegisterElectionRequest) (*electoralv1.RegisterElectionResponse, error) {
	if req.GetName() == "" || req.GetOpensAt() == "" || req.GetClosesAt() == "" {
		return nil, status.Error(codes.InvalidArgument, "name, opens_at, closes_at are required")
	}
	id, err := h.svc.RegisterElection(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "register election: %v", err)
	}
	return &electoralv1.RegisterElectionResponse{ElectionId: id}, nil
}

func (h *ElectoralHandler) VerifyEligibility(ctx context.Context, req *electoralv1.VerifyEligibilityRequest) (*electoralv1.VerifyEligibilityResponse, error) {
	if req.GetElectionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "election_id is required")
	}
	eligible, nullifier, reason, err := h.svc.VerifyEligibility(ctx, req.GetElectionId(), req.GetZkProof(), req.GetPublicInputs())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify eligibility: %v", err)
	}
	return &electoralv1.VerifyEligibilityResponse{Eligible: eligible, NullifierHash: nullifier, Reason: reason}, nil
}

func (h *ElectoralHandler) CastBallot(ctx context.Context, req *electoralv1.CastBallotRequest) (*electoralv1.CastBallotResponse, error) {
	if req.GetElectionId() == "" || req.GetNullifierHash() == "" {
		return nil, status.Error(codes.InvalidArgument, "election_id and nullifier_hash are required")
	}
	receipt, blockHeight, err := h.svc.CastBallot(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cast ballot: %v", err)
	}
	return &electoralv1.CastBallotResponse{ReceiptHash: receipt, BlockHeight: blockHeight}, nil
}

func (h *ElectoralHandler) GetElectionStatus(ctx context.Context, req *electoralv1.GetElectionStatusRequest) (*electoralv1.GetElectionStatusResponse, error) {
	if req.GetElectionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "election_id is required")
	}
	rec, err := h.svc.GetElectionStatus(ctx, req.GetElectionId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get election status: %v", err)
	}
	return &electoralv1.GetElectionStatusResponse{
		ElectionId:  rec.ID,
		Name:        rec.Name,
		Status:      electoralv1.ElectionStatus(electoralv1.ElectionStatus_value["ELECTION_STATUS_"+stringToEnum(rec.Status)]),
		OpensAt:     rec.OpensAt.UTC().Format("2006-01-02T15:04:05Z"),
		ClosesAt:    rec.ClosesAt.UTC().Format("2006-01-02T15:04:05Z"),
		BallotCount: rec.BallotCount,
	}, nil
}

func stringToEnum(s string) string {
	switch s {
	case "scheduled":
		return "SCHEDULED"
	case "open":
		return "OPEN"
	case "closed":
		return "CLOSED"
	case "tallied":
		return "TALLIED"
	default:
		return "UNSPECIFIED"
	}
}
