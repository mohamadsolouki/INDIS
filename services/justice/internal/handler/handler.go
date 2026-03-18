// Package handler implements gRPC handlers for the justice service.
package handler

import (
	"context"

	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	"github.com/IranProsperityProject/INDIS/services/justice/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// JusticeHandler implements justicev1.JusticeServiceServer.
type JusticeHandler struct {
	justicev1.UnimplementedJusticeServiceServer
	svc *service.JusticeService
}

func New(svc *service.JusticeService) *JusticeHandler { return &JusticeHandler{svc: svc} }

func (h *JusticeHandler) SubmitTestimony(ctx context.Context, req *justicev1.SubmitTestimonyRequest) (*justicev1.SubmitTestimonyResponse, error) {
	if len(req.GetEncryptedTestimony()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "encrypted_testimony is required")
	}
	token, caseID, submittedAt, err := h.svc.SubmitTestimony(ctx,
		req.GetZkCitizenshipProof(), req.GetEncryptedTestimony(), req.GetCategory(), req.GetLocale())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "submit testimony: %v", err)
	}
	return &justicev1.SubmitTestimonyResponse{ReceiptToken: token, CaseId: caseID, SubmittedAt: submittedAt}, nil
}

func (h *JusticeHandler) LinkTestimony(ctx context.Context, req *justicev1.LinkTestimonyRequest) (*justicev1.LinkTestimonyResponse, error) {
	if req.GetReceiptToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "receipt_token is required")
	}
	caseID, linkedAt, err := h.svc.LinkTestimony(ctx, req.GetReceiptToken(), req.GetEncryptedTestimony(), req.GetLocale())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "link testimony: %v", err)
	}
	return &justicev1.LinkTestimonyResponse{CaseId: caseID, LinkedAt: linkedAt}, nil
}

func (h *JusticeHandler) InitiateAmnesty(ctx context.Context, req *justicev1.InitiateAmnestyRequest) (*justicev1.InitiateAmnestyResponse, error) {
	if req.GetApplicantDid() == "" {
		return nil, status.Error(codes.InvalidArgument, "applicant_did is required")
	}
	caseID, receipt, submittedAt, err := h.svc.InitiateAmnesty(ctx,
		req.GetApplicantDid(), req.GetEncryptedDeclaration(), req.GetCategory())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "initiate amnesty: %v", err)
	}
	return &justicev1.InitiateAmnestyResponse{CaseId: caseID, Receipt: receipt, SubmittedAt: submittedAt}, nil
}

func (h *JusticeHandler) GetCaseStatus(ctx context.Context, req *justicev1.GetCaseStatusRequest) (*justicev1.GetCaseStatusResponse, error) {
	if req.GetCaseId() == "" && req.GetReceiptToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "case_id or receipt_token is required")
	}
	caseID, st, updatedAt, err := h.svc.GetCaseStatus(ctx, req.GetCaseId(), req.GetReceiptToken())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get case status: %v", err)
	}
	statusEnum := justicev1.CaseStatus_CASE_STATUS_RECEIVED
	switch st {
	case "under_review":
		statusEnum = justicev1.CaseStatus_CASE_STATUS_UNDER_REVIEW
	case "referred":
		statusEnum = justicev1.CaseStatus_CASE_STATUS_REFERRED
	case "closed":
		statusEnum = justicev1.CaseStatus_CASE_STATUS_CLOSED
	}
	return &justicev1.GetCaseStatusResponse{CaseId: caseID, Status: statusEnum, UpdatedAt: updatedAt}, nil
}
