// Package events defines Kafka topic names and event payload types for INDIS
// inter-service messaging.
package events

import "time"

// Kafka topic constants used across INDIS services.
const (
	TopicEnrollmentCompleted = "indis.enrollment.completed"
	TopicCredentialRevoked   = "indis.credential.revoked"
	TopicIdentityDeactivated = "indis.identity.deactivated"
)

// EnrollmentCompletedEvent is published to TopicEnrollmentCompleted when an
// enrollment workflow finishes successfully.
type EnrollmentCompletedEvent struct {
	EnrollmentID string    `json:"enrollment_id"`
	DID          string    `json:"did"`
	SubjectName  string    `json:"subject_name"`
	DistrictCode string    `json:"district_code"`
	PathwayType  string    `json:"pathway_type"` // standard|enhanced|social
	OccurredAt   time.Time `json:"occurred_at"`
}

// CredentialRevokedEvent is published to TopicCredentialRevoked when a
// Verifiable Credential is revoked.
type CredentialRevokedEvent struct {
	CredentialID   string    `json:"credential_id"`
	SubjectDID     string    `json:"subject_did"`
	CredentialType string    `json:"credential_type"`
	RevokedBy      string    `json:"revoked_by"`
	Reason         string    `json:"reason"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// IdentityDeactivatedEvent is published to TopicIdentityDeactivated when a
// DID subject's identity is deactivated.
type IdentityDeactivatedEvent struct {
	DID           string    `json:"did"`
	DeactivatedBy string    `json:"deactivated_by"`
	Reason        string    `json:"reason"`
	OccurredAt    time.Time `json:"occurred_at"`
}
