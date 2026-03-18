// Package handler implements gRPC and HTTP handlers for the audit service.
package handler

import (
	"context"
	"time"

	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuditHandler implements auditv1.AuditServiceServer.
type AuditHandler struct {
	auditv1.UnimplementedAuditServiceServer
	svc *service.AuditService
}

// New creates an AuditHandler.
func New(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// AppendEvent writes a new audit event.
func (h *AuditHandler) AppendEvent(ctx context.Context, req *auditv1.AppendEventRequest) (*auditv1.AppendEventResponse, error) {
	if req.GetAction() == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}
	eventID, prevHash, ts, err := h.svc.AppendEvent(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}
	return &auditv1.AppendEventResponse{EventId: eventID, PrevHash: prevHash, Timestamp: ts}, nil
}

// QueryEvents retrieves audit events matching the given filter.
func (h *AuditHandler) QueryEvents(ctx context.Context, req *auditv1.QueryEventsRequest) (*auditv1.QueryEventsResponse, error) {
	recs, nextToken, err := h.svc.QueryEvents(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query events: %v", err)
	}
	events := make([]*auditv1.AuditEvent, len(recs))
	for i, r := range recs {
		events[i] = recordToProto(r)
	}
	return &auditv1.QueryEventsResponse{Events: events, NextPageToken: nextToken}, nil
}

// GetEventByID retrieves a single event by ID.
func (h *AuditHandler) GetEventByID(ctx context.Context, req *auditv1.GetEventByIDRequest) (*auditv1.GetEventByIDResponse, error) {
	if req.GetEventId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}
	rec, err := h.svc.GetEventByID(ctx, req.GetEventId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get event: %v", err)
	}
	return &auditv1.GetEventByIDResponse{Event: recordToProto(*rec)}, nil
}

func recordToProto(r repository.EventRecord) *auditv1.AuditEvent {
	return &auditv1.AuditEvent{
		EventId:    r.EventID,
		Category:   auditv1.EventCategory(r.Category),
		Action:     r.Action,
		ActorDid:   r.ActorDID,
		SubjectDid: r.SubjectDID,
		ResourceId: r.ResourceID,
		ServiceId:  r.ServiceID,
		Metadata:   r.Metadata,
		PrevHash:   r.PrevHash,
		EntryHash:  r.EntryHash,
		Timestamp:  r.Timestamp.UTC().Format(time.RFC3339),
	}
}
