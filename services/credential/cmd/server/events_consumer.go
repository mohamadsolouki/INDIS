package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IranProsperityProject/INDIS/pkg/events"
	"github.com/IranProsperityProject/INDIS/services/credential/internal/service"
)

func runEnrollmentCompletedConsumer(ctx context.Context, brokers []string, groupID string, svc *service.CredentialService) error {
	consumer, err := events.NewConsumer(brokers, groupID)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := consumer.Close(); cerr != nil {
			log.Printf("kafka consumer close: %v", cerr)
		}
	}()

	consumer.Subscribe(events.TopicEnrollmentCompleted, func(handlerCtx context.Context, _ string, data []byte) error {
		var event events.EnrollmentCompletedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		if event.DID == "" {
			return nil
		}

		baseAttrs := map[string]string{
			"pathway_type": event.PathwayType,
		}
		if event.DistrictCode != "" {
			baseAttrs["district_code"] = event.DistrictCode
		}

		if _, err := svc.IssueCredential(handlerCtx, event.DID, 1, baseAttrs); err != nil {
			log.Printf("event issue citizenship credential failed did=%s enrollment_id=%s err=%v", event.DID, event.EnrollmentID, err)
		}
		if _, err := svc.IssueCredential(handlerCtx, event.DID, 2, map[string]string{"source": "enrollment_completed"}); err != nil {
			log.Printf("event issue age-range credential failed did=%s enrollment_id=%s err=%v", event.DID, event.EnrollmentID, err)
		}
		if _, err := svc.IssueCredential(handlerCtx, event.DID, 3, baseAttrs); err != nil {
			log.Printf("event issue voter-eligibility credential failed did=%s enrollment_id=%s err=%v", event.DID, event.EnrollmentID, err)
		}

		return nil
	})

	return consumer.Run(ctx)
}
