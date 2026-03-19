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
	"time"

	"github.com/IranProsperityProject/INDIS/services/govportal/internal/repository"
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

// GovPortalService implements business logic for the ministry portal.
type GovPortalService struct {
	repo *repository.Repository
}

// New creates a GovPortalService.
func New(repo *repository.Repository) *GovPortalService {
	return &GovPortalService{repo: repo}
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
