// Package handler implements gRPC and HTTP handlers for the enrollment service.
package handler

import (
	"context"

	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	"github.com/IranProsperityProject/INDIS/services/enrollment/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EnrollmentHandler implements enrollmentv1.EnrollmentServiceServer.
type EnrollmentHandler struct {
	enrollmentv1.UnimplementedEnrollmentServiceServer
	svc *service.EnrollmentService
}

// New creates an EnrollmentHandler wrapping the given service.
func New(svc *service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{svc: svc}
}

// InitiateEnrollment begins the enrollment process.
func (h *EnrollmentHandler) InitiateEnrollment(ctx context.Context, req *enrollmentv1.InitiateEnrollmentRequest) (*enrollmentv1.InitiateEnrollmentResponse, error) {
	result, err := h.svc.InitiateEnrollment(ctx, int32(req.GetPathway()), req.GetAgentId(), req.GetLocale())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "initiate enrollment: %v", err)
	}
	return &enrollmentv1.InitiateEnrollmentResponse{
		EnrollmentId:               result.EnrollmentID,
		TemporaryReceiptCredential: result.TemporaryReceiptCredential,
	}, nil
}

// SubmitBiometrics uploads biometric data for deduplication.
func (h *EnrollmentHandler) SubmitBiometrics(ctx context.Context, req *enrollmentv1.SubmitBiometricsRequest) (*enrollmentv1.SubmitBiometricsResponse, error) {
	if req.GetEnrollmentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_id is required")
	}
	passed, ms, err := h.svc.SubmitBiometrics(ctx,
		req.GetEnrollmentId(),
		req.GetFacialData(),
		req.GetFingerprintData(),
		req.GetIrisData(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "submit biometrics: %v", err)
	}
	return &enrollmentv1.SubmitBiometricsResponse{
		DeduplicationPassed: passed,
		DeduplicationTimeMs: ms,
	}, nil
}

// SubmitSocialAttestation submits community co-attestation (3+ attestors).
func (h *EnrollmentHandler) SubmitSocialAttestation(ctx context.Context, req *enrollmentv1.SubmitSocialAttestationRequest) (*enrollmentv1.SubmitSocialAttestationResponse, error) {
	if req.GetEnrollmentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_id is required")
	}
	attestorDIDs := make([]string, len(req.GetAttestors()))
	for i, a := range req.GetAttestors() {
		attestorDIDs[i] = a.GetAttestorDid()
	}
	accepted, count, err := h.svc.SubmitSocialAttestation(ctx, req.GetEnrollmentId(), attestorDIDs)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "submit attestation: %v", err)
	}
	return &enrollmentv1.SubmitSocialAttestationResponse{Accepted: accepted, AttestorCount: count}, nil
}

// CompleteEnrollment finalizes enrollment after all checks pass.
func (h *EnrollmentHandler) CompleteEnrollment(ctx context.Context, req *enrollmentv1.CompleteEnrollmentRequest) (*enrollmentv1.CompleteEnrollmentResponse, error) {
	if req.GetEnrollmentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_id is required")
	}
	result, err := h.svc.CompleteEnrollment(ctx, req.GetEnrollmentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "complete enrollment: %v", err)
	}
	return &enrollmentv1.CompleteEnrollmentResponse{
		Did:               result.DID,
		IssuedCredentials: result.IssuedCredentials,
	}, nil
}

// GetEnrollmentStatus checks enrollment progress.
func (h *EnrollmentHandler) GetEnrollmentStatus(ctx context.Context, req *enrollmentv1.GetEnrollmentStatusRequest) (*enrollmentv1.GetEnrollmentStatusResponse, error) {
	if req.GetEnrollmentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_id is required")
	}
	statusStr, err := h.svc.GetEnrollmentStatus(ctx, req.GetEnrollmentId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get enrollment status: %v", err)
	}
	return &enrollmentv1.GetEnrollmentStatusResponse{
		Status:       statusStr,
		EnrollmentId: req.GetEnrollmentId(),
	}, nil
}
