package handler

import (
	"context"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitRemoteBallot records a remote encrypted ballot with integrity metadata.
func (h *ElectoralHandler) SubmitRemoteBallot(ctx context.Context, req *electoralv1.SubmitRemoteBallotRequest) (*electoralv1.SubmitRemoteBallotResponse, error) {
	if req.GetElectionId() == "" || req.GetNullifierHash() == "" {
		return nil, status.Error(codes.InvalidArgument, "election_id and nullifier_hash are required")
	}
	if req.GetSubmittedAt() == "" {
		return nil, status.Error(codes.InvalidArgument, "submitted_at is required")
	}

	receipt, blockHeight, acceptedAt, err := h.svc.SubmitRemoteBallot(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "submit remote ballot: %v", err)
	}

	return &electoralv1.SubmitRemoteBallotResponse{
		ReceiptHash: receipt,
		BlockHeight: blockHeight,
		AcceptedAt:  acceptedAt,
	}, nil
}
