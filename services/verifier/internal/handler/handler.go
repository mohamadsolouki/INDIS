// Package handler implements the gRPC handler for the verifier service.
package handler

import (
	"context"
	"time"

	verifierv1 "github.com/mohamadsolouki/INDIS/api/gen/go/verifier/v1"
	"github.com/mohamadsolouki/INDIS/services/verifier/internal/repository"
	"github.com/mohamadsolouki/INDIS/services/verifier/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VerifierHandler implements verifierv1.VerifierServiceServer.
type VerifierHandler struct {
	verifierv1.UnimplementedVerifierServiceServer
	svc *service.VerifierService
}

// New creates a VerifierHandler wrapping the given service.
func New(svc *service.VerifierService) *VerifierHandler {
	return &VerifierHandler{svc: svc}
}

// RegisterVerifier creates a new verifier organization and issues a certificate.
func (h *VerifierHandler) RegisterVerifier(ctx context.Context, req *verifierv1.RegisterVerifierRequest) (*verifierv1.RegisterVerifierResponse, error) {
	if req.GetOrgName() == "" {
		return nil, status.Error(codes.InvalidArgument, "org_name is required")
	}
	if req.GetOrgType() == "" {
		return nil, status.Error(codes.InvalidArgument, "org_type is required")
	}

	result, err := h.svc.RegisterVerifier(
		ctx,
		req.GetOrgName(),
		req.GetOrgType(),
		req.GetAuthorizedCredentialTypes(),
		req.GetGeographicScope(),
		req.GetMaxVerificationsPerDay(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "register verifier: %v", err)
	}

	return &verifierv1.RegisterVerifierResponse{
		VerifierId:    result.VerifierID,
		CertificateId: result.CertificateID,
		PublicKeyHex:  result.PublicKeyHex,
	}, nil
}

// GetVerifier retrieves a verifier record by ID.
func (h *VerifierHandler) GetVerifier(ctx context.Context, req *verifierv1.GetVerifierRequest) (*verifierv1.GetVerifierResponse, error) {
	if req.GetVerifierId() == "" {
		return nil, status.Error(codes.InvalidArgument, "verifier_id is required")
	}

	rec, err := h.svc.GetVerifier(ctx, req.GetVerifierId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get verifier: %v", err)
	}

	return verifierRecordToProto(rec), nil
}

// ListVerifiers lists all verifiers, optionally filtered by status.
func (h *VerifierHandler) ListVerifiers(ctx context.Context, req *verifierv1.ListVerifiersRequest) (*verifierv1.ListVerifiersResponse, error) {
	recs, err := h.svc.ListVerifiers(ctx, req.GetStatusFilter())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list verifiers: %v", err)
	}

	resp := &verifierv1.ListVerifiersResponse{}
	for _, rec := range recs {
		resp.Verifiers = append(resp.Verifiers, verifierRecordToProto(rec))
	}
	return resp, nil
}

// SuspendVerifier sets a verifier's status to 'suspended' (or 'revoked' when reason is "revoked").
func (h *VerifierHandler) SuspendVerifier(ctx context.Context, req *verifierv1.SuspendVerifierRequest) (*verifierv1.SuspendVerifierResponse, error) {
	if req.GetVerifierId() == "" {
		return nil, status.Error(codes.InvalidArgument, "verifier_id is required")
	}

	if err := h.svc.SuspendVerifier(ctx, req.GetVerifierId(), req.GetReason()); err != nil {
		return nil, status.Errorf(codes.NotFound, "suspend verifier: %v", err)
	}

	return &verifierv1.SuspendVerifierResponse{Success: true}, nil
}

// VerifyCredential validates a ZK proof via the zkproof service and logs the event.
func (h *VerifierHandler) VerifyCredential(ctx context.Context, req *verifierv1.VerifyCredentialRequest) (*verifierv1.VerifyCredentialResponse, error) {
	if req.GetVerifierId() == "" {
		return nil, status.Error(codes.InvalidArgument, "verifier_id is required")
	}
	if req.GetCredentialType() == "" {
		return nil, status.Error(codes.InvalidArgument, "credential_type is required")
	}
	if req.GetNonce() == "" {
		return nil, status.Error(codes.InvalidArgument, "nonce is required")
	}
	if req.GetProofB64() == "" {
		return nil, status.Error(codes.InvalidArgument, "proof_b64 is required")
	}

	valid, eventID, err := h.svc.VerifyCredential(
		ctx,
		req.GetVerifierId(),
		req.GetCredentialType(),
		req.GetPredicate(),
		req.GetNonce(),
		req.GetProofSystem(),
		req.GetProofB64(),
		req.GetPublicInputsB64(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify credential: %v", err)
	}

	return &verifierv1.VerifyCredentialResponse{
		Valid:   valid,
		EventId: eventID,
	}, nil
}

// GetVerificationHistory retrieves past verification events for a verifier.
func (h *VerifierHandler) GetVerificationHistory(ctx context.Context, req *verifierv1.GetVerificationHistoryRequest) (*verifierv1.GetVerificationHistoryResponse, error) {
	if req.GetVerifierId() == "" {
		return nil, status.Error(codes.InvalidArgument, "verifier_id is required")
	}

	evts, err := h.svc.GetVerificationHistory(ctx, req.GetVerifierId(), req.GetLimit())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get verification history: %v", err)
	}

	resp := &verifierv1.GetVerificationHistoryResponse{}
	for _, evt := range evts {
		resp.Events = append(resp.Events, verificationEventToProto(evt))
	}
	return resp, nil
}

// verifierRecordToProto converts a repository VerifierRecord to its proto representation.
func verifierRecordToProto(rec *repository.VerifierRecord) *verifierv1.GetVerifierResponse {
	return &verifierv1.GetVerifierResponse{
		VerifierId:               rec.ID,
		OrgName:                  rec.OrgName,
		OrgType:                  rec.OrgType,
		AuthorizedCredentialTypes: rec.AuthorizedCredentialTypes,
		Status:                   rec.Status,
		CertificateId:            rec.CertificateID,
		RegisteredAt:             rec.RegisteredAt.UTC().Format(time.RFC3339),
	}
}

// verificationEventToProto converts a repository VerificationEventRecord to its proto representation.
func verificationEventToProto(evt *repository.VerificationEventRecord) *verifierv1.VerificationEvent {
	return &verifierv1.VerificationEvent{
		Id:             evt.ID,
		CredentialType: evt.CredentialType,
		Result:         evt.Result,
		ProofSystem:    evt.ProofSystem,
		OccurredAt:     evt.OccurredAt.UTC().Format(time.RFC3339),
	}
}
