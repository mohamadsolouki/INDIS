// Package handler implements gRPC and HTTP handlers for the credential service.
package handler

import (
	"context"

	credentialv1 "github.com/mohamadsolouki/INDIS/api/gen/go/credential/v1"
	"github.com/mohamadsolouki/INDIS/services/credential/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CredentialHandler implements credentialv1.CredentialServiceServer.
type CredentialHandler struct {
	credentialv1.UnimplementedCredentialServiceServer
	svc *service.CredentialService
}

// New creates a CredentialHandler wrapping the given service.
func New(svc *service.CredentialService) *CredentialHandler {
	return &CredentialHandler{svc: svc}
}

// IssueCredential issues a new verifiable credential to a citizen.
func (h *CredentialHandler) IssueCredential(ctx context.Context, req *credentialv1.IssueCredentialRequest) (*credentialv1.IssueCredentialResponse, error) {
	if req.GetSubjectDid() == "" {
		return nil, status.Error(codes.InvalidArgument, "subject_did is required")
	}
	if req.GetType() == credentialv1.CredentialType_CREDENTIAL_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "credential type is required")
	}

	result, err := h.svc.IssueCredential(ctx, req.GetSubjectDid(), int32(req.GetType()), req.GetAttributes())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "issue credential: %v", err)
	}
	return &credentialv1.IssueCredentialResponse{
		CredentialId:   result.CredentialID,
		TxId:           result.TxID,
		CredentialData: result.CredentialData,
	}, nil
}

// VerifyCredential verifies a credential proof.
func (h *CredentialHandler) VerifyCredential(ctx context.Context, req *credentialv1.VerifyCredentialRequest) (*credentialv1.VerifyCredentialResponse, error) {
	valid, reason := h.svc.VerifyCredential(ctx, req.GetProof(), req.GetVerificationKey())
	return &credentialv1.VerifyCredentialResponse{Valid: valid, Reason: reason}, nil
}

// RevokeCredential revokes a previously issued credential.
func (h *CredentialHandler) RevokeCredential(ctx context.Context, req *credentialv1.RevokeCredentialRequest) (*credentialv1.RevokeCredentialResponse, error) {
	if req.GetCredentialId() == "" {
		return nil, status.Error(codes.InvalidArgument, "credential_id is required")
	}
	txID, err := h.svc.RevokeCredential(ctx, req.GetCredentialId(), req.GetReason(), req.GetRevokerDid())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "revoke credential: %v", err)
	}
	return &credentialv1.RevokeCredentialResponse{TxId: txID}, nil
}

// CheckRevocationStatus checks if a credential has been revoked.
func (h *CredentialHandler) CheckRevocationStatus(ctx context.Context, req *credentialv1.CheckRevocationStatusRequest) (*credentialv1.CheckRevocationStatusResponse, error) {
	if req.GetCredentialId() == "" {
		return nil, status.Error(codes.InvalidArgument, "credential_id is required")
	}
	result, err := h.svc.CheckRevocationStatus(ctx, req.GetCredentialId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "check revocation: %v", err)
	}
	return &credentialv1.CheckRevocationStatusResponse{
		Revoked:   result.Revoked,
		Reason:    result.Reason,
		RevokedAt: result.RevokedAt,
	}, nil
}
