package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	notificationv1 "github.com/IranProsperityProject/INDIS/api/gen/go/notification/v1"
	"github.com/IranProsperityProject/INDIS/pkg/events"
	"github.com/IranProsperityProject/INDIS/services/notification/internal/service"
)

func runCredentialRevokedConsumer(ctx context.Context, brokers []string, groupID string, svc *service.NotificationService) error {
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
		if event.SubjectDID == "" {
			return nil
		}

		_, _, err := svc.Send(handlerCtx, &notificationv1.SendRequest{
			RecipientDid: event.SubjectDID,
			Channel:      notificationv1.Channel_CHANNEL_PUSH,
			Type:         notificationv1.NotificationType_NOTIFICATION_TYPE_CREDENTIAL_REVOKED,
			Locale:       "fa",
			Subject:      "Credential Revoked",
			Body:         fmt.Sprintf("Your %s credential has been revoked.", event.CredentialType),
		})
		if err != nil {
			log.Printf("credential revoked notification failed credential_id=%s err=%v", event.CredentialID, err)
		}
		return nil
	})

	return consumer.Run(ctx)
}
