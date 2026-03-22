// Package handler implements gRPC handlers for the biometric service.
package handler

import (
	"context"

	biometricv1 "github.com/mohamadsolouki/INDIS/api/gen/go/biometric/v1"
	"github.com/mohamadsolouki/INDIS/services/biometric/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BiometricHandler implements biometricv1.BiometricServiceServer.
type BiometricHandler struct {
	biometricv1.UnimplementedBiometricServiceServer
	svc *service.BiometricService
}

// New creates a BiometricHandler.
func New(svc *service.BiometricService) *BiometricHandler {
	return &BiometricHandler{svc: svc}
}

func (h *BiometricHandler) StoreTemplate(ctx context.Context, req *biometricv1.StoreTemplateRequest) (*biometricv1.StoreTemplateResponse, error) {
	if req.GetEnrollmentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_id is required")
	}
	if len(req.GetTemplateData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "template_data is required")
	}
	id, err := h.svc.StoreTemplate(ctx, req.GetEnrollmentId(), int32(req.GetModality()), req.GetTemplateData(), req.GetLivenessVerified())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "store template: %v", err)
	}
	return &biometricv1.StoreTemplateResponse{TemplateId: id}, nil
}

func (h *BiometricHandler) CheckDuplicate(ctx context.Context, req *biometricv1.CheckDuplicateRequest) (*biometricv1.CheckDuplicateResponse, error) {
	if req.GetEnrollmentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_id is required")
	}
	isDup, matchedDID, ms, score, err := h.svc.CheckDuplicate(ctx, req.GetEnrollmentId(), int32(req.GetModality()), req.GetTemplateData())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check duplicate: %v", err)
	}
	return &biometricv1.CheckDuplicateResponse{
		IsDuplicate:     isDup,
		MatchedDid:      matchedDID,
		MatchScore:      score,
		DeduplicationMs: ms,
	}, nil
}

func (h *BiometricHandler) DeleteTemplate(ctx context.Context, req *biometricv1.DeleteTemplateRequest) (*biometricv1.DeleteTemplateResponse, error) {
	if req.GetTemplateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	deleted, deletedAt, err := h.svc.DeleteTemplate(ctx, req.GetTemplateId(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "delete template: %v", err)
	}
	return &biometricv1.DeleteTemplateResponse{Deleted: deleted, DeletedAt: deletedAt}, nil
}
