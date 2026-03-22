package main

import (
	"context"
	"encoding/json"
	"log"

	auditv1 "github.com/mohamadsolouki/INDIS/api/gen/go/audit/v1"
	"github.com/mohamadsolouki/INDIS/pkg/events"
	"github.com/mohamadsolouki/INDIS/services/audit/internal/service"
)

func runCredentialRevokedConsumer(ctx context.Context, brokers []string, groupID string, svc *service.AuditService) error {
	consumer, err := events.NewConsumer(brokers, groupID)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := consumer.Close(); cerr != nil {
			log.Printf("kafka consumer close: %v", cerr)
		}
	}()

	consumer.Subscribe(events.TopicCredentialRevoked, func(handlerCtx context.Context, _ string, data []byte) error {
		var event events.CredentialRevokedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}

		meta, _ := json.Marshal(map[string]string{
			"credential_type": event.CredentialType,
			"reason":          event.Reason,
			"revoked_by":      event.RevokedBy,
		})
		_, _, _, err := svc.AppendEvent(handlerCtx, &auditv1.AppendEventRequest{
			Category:   auditv1.EventCategory_EVENT_CATEGORY_CREDENTIAL,
			Action:     "credential.revoke",
			ActorDid:   event.RevokedBy,
			SubjectDid: event.SubjectDID,
			ResourceId: event.CredentialID,
			ServiceId:  "credential",
			Metadata:   meta,
		})
		if err != nil {
			log.Printf("audit append for revoked credential failed credential_id=%s err=%v", event.CredentialID, err)
		}
		return nil
	})

	return consumer.Run(ctx)
}
