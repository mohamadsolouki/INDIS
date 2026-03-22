// Package handler implements gRPC handlers for the notification service.
package handler

import (
	"context"

	notificationv1 "github.com/mohamadsolouki/INDIS/api/gen/go/notification/v1"
	"github.com/mohamadsolouki/INDIS/services/notification/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotificationHandler implements notificationv1.NotificationServiceServer.
type NotificationHandler struct {
	notificationv1.UnimplementedNotificationServiceServer
	svc *service.NotificationService
}

// New creates a NotificationHandler.
func New(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) Send(ctx context.Context, req *notificationv1.SendRequest) (*notificationv1.SendResponse, error) {
	if req.GetRecipientDid() == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_did is required")
	}
	id, st, err := h.svc.Send(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send: %v", err)
	}
	return &notificationv1.SendResponse{NotificationId: id, Status: st}, nil
}

func (h *NotificationHandler) ScheduleExpiryAlert(ctx context.Context, req *notificationv1.ScheduleExpiryAlertRequest) (*notificationv1.ScheduleExpiryAlertResponse, error) {
	if req.GetRecipientDid() == "" || req.GetExpiresAt() == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_did and expires_at are required")
	}
	ids, err := h.svc.ScheduleExpiryAlert(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schedule expiry alert: %v", err)
	}
	return &notificationv1.ScheduleExpiryAlertResponse{AlertIds: ids}, nil
}

func (h *NotificationHandler) CancelAlert(ctx context.Context, req *notificationv1.CancelAlertRequest) (*notificationv1.CancelAlertResponse, error) {
	if req.GetAlertId() == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}
	cancelled, err := h.svc.CancelAlert(ctx, req.GetAlertId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "cancel alert: %v", err)
	}
	return &notificationv1.CancelAlertResponse{Cancelled: cancelled}, nil
}
