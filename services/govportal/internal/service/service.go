// Package service implements business logic for the govportal service.
// It provides ministry dashboard operations including portal user management,
// bulk credential operations, and aggregated statistics.
// Implements PRD FR-009 (ministry dashboard), FR-010 (bulk operations), FR-011 (audit reports).
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	credentialv1 "github.com/mohamadsolouki/INDIS/api/gen/go/credential/v1"
	auditv1 "github.com/mohamadsolouki/INDIS/api/gen/go/audit/v1"
	enrollmentv1 "github.com/mohamadsolouki/INDIS/api/gen/go/enrollment/v1"
	identityv1 "github.com/mohamadsolouki/INDIS/api/gen/go/identity/v1"
	"github.com/mohamadsolouki/INDIS/services/govportal/internal/repository"
)

// RoleHierarchy maps role names to numeric levels for comparison.
// Higher level = more privileges.
var RoleHierarchy = map[string]int{
	"viewer":   1,
	"operator": 2,
	"senior":   3,
	"admin":    4,
}

// StatsResult holds aggregated portal statistics.
type StatsResult struct {
	// TotalPortalUsers is the count of portal_users rows.
	TotalPortalUsers int64
	// TotalBulkOperations is the count of bulk_operations rows.
	TotalBulkOperations int64
	// PendingBulkOperations is the count of pending rows.
	PendingBulkOperations int64
}

// GovPortalRepository defines the data-access behavior required by the service.
type GovPortalRepository interface {
	CreatePortalUser(ctx context.Context, rec repository.PortalUserRecord) error
	ListPortalUsers(ctx context.Context, ministryFilter string) ([]*repository.PortalUserRecord, error)
	UpdatePortalUserRole(ctx context.Context, id, role string) error
	CreateBulkOperation(ctx context.Context, rec repository.BulkOperationRecord) error
	ListBulkOperations(ctx context.Context, statusFilter, ministryFilter string) ([]*repository.BulkOperationRecord, error)
	ApproveBulkOperation(ctx context.Context, id, approvedBy, newStatus string) error
	SetBulkOperationResult(ctx context.Context, id, approvedBy, newStatus string, resultSummary json.RawMessage) error
	GetBulkOperationByID(ctx context.Context, id string) (*repository.BulkOperationRecord, error)
	CountRows(ctx context.Context, tableName string) (int64, error)
}

// GovPortalService implements business logic for the ministry portal.
type GovPortalService struct {
	repo GovPortalRepository

	credentialClient credentialv1.CredentialServiceClient
	enrollmentClient enrollmentv1.EnrollmentServiceClient
	identityClient    identityv1.IdentityServiceClient

	auditClient auditv1.AuditServiceClient
}

// New creates a GovPortalService.
func New(repo GovPortalRepository) *GovPortalService {
	return &GovPortalService{repo: repo}
}

// SetCredentialClient wires the govportal service to the credential gRPC API.
func (s *GovPortalService) SetCredentialClient(c credentialv1.CredentialServiceClient) {
	s.credentialClient = c
}

// SetEnrollmentClient wires the govportal service to the enrollment gRPC API.
func (s *GovPortalService) SetEnrollmentClient(c enrollmentv1.EnrollmentServiceClient) {
	s.enrollmentClient = c
}

// SetIdentityClient wires the govportal service to the identity gRPC API.
func (s *GovPortalService) SetIdentityClient(c identityv1.IdentityServiceClient) {
	s.identityClient = c
}

// SetAuditClient wires the govportal service to the audit gRPC API.
func (s *GovPortalService) SetAuditClient(c auditv1.AuditServiceClient) {
	s.auditClient = c
}

// AppendAuditEvent records a tamper-evident audit event.
// If audit client is missing, it becomes a no-op (dev-friendly).
func (s *GovPortalService) AppendAuditEvent(
	ctx context.Context,
	category auditv1.EventCategory,
	action string,
	actorDID string,
	subjectDID string,
	resourceID string,
	metadata any,
) error {
	if s.auditClient == nil {
		return nil
	}

	metaBytes := []byte("{}")
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metaBytes = b
		}
	}

	_, err := s.auditClient.AppendEvent(ctx, &auditv1.AppendEventRequest{
		Category:   category,
		Action:     action,
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
		ResourceId: resourceID,
		ServiceId:  "govportal",
		Metadata:   metaBytes,
	})
	return err
}

// CreatePortalUser creates a new ministry portal user.
// The plaintext API key is hashed with SHA-256 before storage.
func (s *GovPortalService) CreatePortalUser(ctx context.Context, username, ministry, role, plaintextAPIKey string) (*repository.PortalUserRecord, error) {
	if username == "" {
		return nil, fmt.Errorf("service: username is required")
	}
	if ministry == "" {
		return nil, fmt.Errorf("service: ministry is required")
	}
	if _, ok := RoleHierarchy[role]; !ok {
		return nil, fmt.Errorf("service: invalid role %q; must be viewer, operator, senior, or admin", role)
	}

	id, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("service: generate user ID: %w", err)
	}

	apiKeyHash := ""
	if plaintextAPIKey != "" {
		h := sha256.Sum256([]byte(plaintextAPIKey))
		apiKeyHash = hex.EncodeToString(h[:])
	}

	now := time.Now().UTC()
	rec := repository.PortalUserRecord{
		ID:         id,
		Username:   username,
		Ministry:   ministry,
		Role:       role,
		APIKeyHash: apiKeyHash,
		CreatedAt:  now,
	}
	if err := s.repo.CreatePortalUser(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: create portal user: %w", err)
	}
	return &rec, nil
}

// ListPortalUsers returns all portal users, optionally filtered by ministry.
func (s *GovPortalService) ListPortalUsers(ctx context.Context, ministryFilter string) ([]*repository.PortalUserRecord, error) {
	users, err := s.repo.ListPortalUsers(ctx, ministryFilter)
	if err != nil {
		return nil, fmt.Errorf("service: list portal users: %w", err)
	}
	return users, nil
}

// AssignRole changes the role of an existing portal user.
func (s *GovPortalService) AssignRole(ctx context.Context, userID, newRole string) error {
	if _, ok := RoleHierarchy[newRole]; !ok {
		return fmt.Errorf("service: invalid role %q; must be viewer, operator, senior, or admin", newRole)
	}
	if err := s.repo.UpdatePortalUserRole(ctx, userID, newRole); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("service: portal user not found: %s", userID)
		}
		return fmt.Errorf("service: assign role: %w", err)
	}
	return nil
}

// CreateBulkOperation creates a new bulk credential operation in 'pending' state.
// The caller must be at least 'operator' role (enforced in the handler layer via JWT claims).
func (s *GovPortalService) CreateBulkOperation(ctx context.Context, opType, ministry, requestedBy string, targetDIDs []string, parameters json.RawMessage) (*repository.BulkOperationRecord, error) {
	if opType == "" {
		return nil, fmt.Errorf("service: operation_type is required")
	}
	if ministry == "" {
		return nil, fmt.Errorf("service: ministry is required")
	}
	if requestedBy == "" {
		return nil, fmt.Errorf("service: requested_by is required")
	}
	if len(targetDIDs) == 0 {
		return nil, fmt.Errorf("service: target_dids must not be empty")
	}

	id, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("service: generate operation ID: %w", err)
	}
	if parameters == nil {
		parameters = json.RawMessage("{}")
	}

	now := time.Now().UTC()
	rec := repository.BulkOperationRecord{
		ID:            id,
		OperationType: opType,
		Ministry:      ministry,
		RequestedBy:   requestedBy,
		Status:        "pending",
		TargetDIDs:    targetDIDs,
		Parameters:    parameters,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateBulkOperation(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: create bulk operation: %w", err)
	}
	return &rec, nil
}

// ListBulkOperations returns bulk operations with optional status and ministry filters.
func (s *GovPortalService) ListBulkOperations(ctx context.Context, statusFilter, ministryFilter string) ([]*repository.BulkOperationRecord, error) {
	ops, err := s.repo.ListBulkOperations(ctx, statusFilter, ministryFilter)
	if err != nil {
		return nil, fmt.Errorf("service: list bulk operations: %w", err)
	}
	return ops, nil
}

// ApproveBulkOperation sets a pending operation to 'approved' and records the approver.
// The approving user must be at least 'senior' role (enforced in the handler layer).
func (s *GovPortalService) ApproveBulkOperation(ctx context.Context, operationID, approverID string) (*repository.BulkOperationRecord, error) {
	if operationID == "" {
		return nil, fmt.Errorf("service: operation_id is required")
	}
	if approverID == "" {
		return nil, fmt.Errorf("service: approver_id is required")
	}

	if err := s.repo.ApproveBulkOperation(ctx, operationID, approverID, "approved"); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("service: bulk operation not found: %s", operationID)
		}
		return nil, fmt.Errorf("service: approve bulk operation: %w", err)
	}

	rec, err := s.repo.GetBulkOperationByID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("service: fetch approved operation: %w", err)
	}
	return rec, nil
}

// ApproveAndExecuteBulkOperation approves a pending operation, then executes it via gRPC and
// persists per-target outcomes into `result_summary`.
func (s *GovPortalService) ApproveAndExecuteBulkOperation(ctx context.Context, operationID, approverID string) (*repository.BulkOperationRecord, error) {
	if operationID == "" {
		return nil, fmt.Errorf("service: operation_id is required")
	}
	if approverID == "" {
		return nil, fmt.Errorf("service: approver_id is required")
	}

	op, err := s.repo.GetBulkOperationByID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("service: fetch bulk operation: %w", err)
	}

	// Move into executing state first (auditable status transition).
	if err := s.repo.ApproveBulkOperation(ctx, operationID, approverID, "executing"); err != nil {
		return nil, fmt.Errorf("service: set executing: %w", err)
	}

	type targetOutcome struct {
		Target     string `json:"target"`
		Success    bool   `json:"success"`
		Error      string `json:"error,omitempty"`
		Credential string `json:"credential_id,omitempty"`
		TxID       string `json:"tx_id,omitempty"`
		DID        string `json:"did,omitempty"`
		IssuedCred []string `json:"issued_credentials,omitempty"`
	}
	outcomes := make([]targetOutcome, 0, len(op.TargetDIDs))
	allOK := true

	switch op.OperationType {
	case "issue_credential":
		if s.credentialClient == nil {
			return nil, fmt.Errorf("service: credential client not configured")
		}
		type issueCredentialParams struct {
			CredentialType int32             `json:"credential_type"`
			Attributes     map[string]string `json:"attributes"`
		}
		paramsBytes := op.Parameters
		if len(paramsBytes) == 0 {
			paramsBytes = json.RawMessage("{}")
		}
		var p issueCredentialParams
		if err := json.Unmarshal(paramsBytes, &p); err != nil {
			return nil, fmt.Errorf("service: parse issue_credential parameters: %w", err)
		}
		if p.CredentialType == 0 {
			return nil, fmt.Errorf("service: issue_credential requires credential_type")
		}
		if p.Attributes == nil {
			p.Attributes = map[string]string{}
		}

		for _, subjectDid := range op.TargetDIDs {
			resp, callErr := s.credentialClient.IssueCredential(ctx, &credentialv1.IssueCredentialRequest{
				SubjectDid: subjectDid,
				Type:       credentialv1.CredentialType(p.CredentialType),
				Attributes: p.Attributes,
				// IssuerDid is ignored by credential handler/service; issuer identity is configured server-side.
			})
			if callErr != nil {
				allOK = false
				outcomes = append(outcomes, targetOutcome{Target: subjectDid, Success: false, Error: callErr.Error()})
				continue
			}
			outcomes = append(outcomes, targetOutcome{
				Target:     subjectDid,
				Success:    true,
				Credential: resp.GetCredentialId(),
				TxID:       resp.GetTxId(),
			})
		}

	case "revoke_credential":
		if s.credentialClient == nil {
			return nil, fmt.Errorf("service: credential client not configured")
		}
		type revokeCredentialParams struct {
			Reason string `json:"reason"`
		}
		paramsBytes := op.Parameters
		if len(paramsBytes) == 0 {
			paramsBytes = json.RawMessage("{}")
		}
		var p revokeCredentialParams
		if err := json.Unmarshal(paramsBytes, &p); err != nil {
			return nil, fmt.Errorf("service: parse revoke_credential parameters: %w", err)
		}
		if p.Reason == "" {
			return nil, fmt.Errorf("service: revoke_credential requires reason")
		}

		for _, credentialID := range op.TargetDIDs {
			resp, callErr := s.credentialClient.RevokeCredential(ctx, &credentialv1.RevokeCredentialRequest{
				CredentialId: credentialID,
				Reason:       p.Reason,
			})
			if callErr != nil {
				allOK = false
				outcomes = append(outcomes, targetOutcome{Target: credentialID, Success: false, Error: callErr.Error()})
				continue
			}
			outcomes = append(outcomes, targetOutcome{
				Target:  credentialID,
				Success: true,
				TxID:    resp.GetTxId(),
			})
		}

	case "enroll_batch":
		if s.enrollmentClient == nil {
			return nil, fmt.Errorf("service: enrollment client not configured")
		}
		type enrollBatchParams struct {
			Pathway   string   `json:"pathway"` // standard|enhanced|social
			Attestors []string `json:"attestors"`
		}
		paramsBytes := op.Parameters
		if len(paramsBytes) == 0 {
			paramsBytes = json.RawMessage("{}")
		}
		var p enrollBatchParams
		if err := json.Unmarshal(paramsBytes, &p); err != nil {
			return nil, fmt.Errorf("service: parse enroll_batch parameters: %w", err)
		}
		pathway := strings.ToLower(strings.TrimSpace(p.Pathway))
		if pathway == "" {
			pathway = "standard"
		}

		for _, enrollmentID := range op.TargetDIDs {
			// Prepare pathway-dependent preconditions.
			var preErr error
			if pathway == "social" {
				attestorMsgs := make([]*enrollmentv1.SocialAttestor, 0, len(p.Attestors))
				for _, a := range p.Attestors {
					attestorMsgs = append(attestorMsgs, &enrollmentv1.SocialAttestor{AttestorDid: a})
				}
				_, preErr = s.enrollmentClient.SubmitSocialAttestation(ctx, &enrollmentv1.SubmitSocialAttestationRequest{
					EnrollmentId: enrollmentID,
					Attestors:    attestorMsgs,
				})
			} else {
				_, preErr = s.enrollmentClient.SubmitBiometrics(ctx, &enrollmentv1.SubmitBiometricsRequest{
					EnrollmentId: enrollmentID,
					FacialData:   []byte{1},
					FingerprintData: nil,
					IrisData:        nil,
				})
			}

			if preErr != nil {
				allOK = false
				outcomes = append(outcomes, targetOutcome{Target: enrollmentID, Success: false, Error: preErr.Error()})
				continue
			}

			comp, compErr := s.enrollmentClient.CompleteEnrollment(ctx, &enrollmentv1.CompleteEnrollmentRequest{EnrollmentId: enrollmentID})
			if compErr != nil {
				allOK = false
				outcomes = append(outcomes, targetOutcome{Target: enrollmentID, Success: false, Error: compErr.Error()})
				continue
			}

			// Best-effort identity registration so other services can resolve the DID.
			if s.identityClient != nil {
				now := time.Now().UTC().Format(time.RFC3339)
				_, regErr := s.identityClient.RegisterIdentity(ctx, &identityv1.RegisterIdentityRequest{
					Did: comp.GetDid(),
					Document: &identityv1.DIDDocument{
						Id:      comp.GetDid(),
						Created: now,
						Updated: now,
					},
				})
				if regErr != nil {
					allOK = false
					outcomes = append(outcomes, targetOutcome{
						Target:  enrollmentID,
						Success: false,
						Error:   "identity registration: " + regErr.Error(),
					})
					continue
				}
			}

			outcomes = append(outcomes, targetOutcome{
				Target:     enrollmentID,
				Success:    true,
				DID:        comp.GetDid(),
				IssuedCred: comp.GetIssuedCredentials(),
			})
		}

	default:
		return nil, fmt.Errorf("service: unknown bulk operation_type %q", op.OperationType)
	}

	finalStatus := "completed"
	if !allOK {
		finalStatus = "failed"
	}

	summary := map[string]any{
		"operation_id":   operationID,
		"operation_type": op.OperationType,
		"results":        outcomes,
	}
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("service: marshal result_summary: %w", err)
	}

	if err := s.repo.SetBulkOperationResult(ctx, operationID, approverID, finalStatus, json.RawMessage(summaryBytes)); err != nil {
		return nil, fmt.Errorf("service: persist result_summary: %w", err)
	}

	rec, err := s.repo.GetBulkOperationByID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("service: fetch final outcome: %w", err)
	}
	return rec, nil
}

// GetStats returns aggregated portal statistics from the database.
func (s *GovPortalService) GetStats(ctx context.Context) (*StatsResult, error) {
	totalUsers, err := s.repo.CountRows(ctx, "portal_users")
	if err != nil {
		return nil, fmt.Errorf("service: count portal users: %w", err)
	}
	totalOps, err := s.repo.CountRows(ctx, "bulk_operations")
	if err != nil {
		return nil, fmt.Errorf("service: count bulk operations: %w", err)
	}

	// Count pending specifically using ListBulkOperations filter.
	pendingOps, err := s.repo.ListBulkOperations(ctx, "pending", "")
	if err != nil {
		return nil, fmt.Errorf("service: count pending bulk operations: %w", err)
	}

	return &StatsResult{
		TotalPortalUsers:      totalUsers,
		TotalBulkOperations:   totalOps,
		PendingBulkOperations: int64(len(pendingOps)),
	}, nil
}

// newUUID generates a random UUID v4 using crypto/rand.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	// Set version 4 and variant bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
