package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/IranProsperityProject/INDIS/services/govportal/internal/repository"
)

// fakeRepo is an in-memory implementation of GovPortalRepository.
type fakeRepo struct {
	users      map[string]*repository.PortalUserRecord
	operations map[string]*repository.BulkOperationRecord
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:      make(map[string]*repository.PortalUserRecord),
		operations: make(map[string]*repository.BulkOperationRecord),
	}
}

func (f *fakeRepo) CreatePortalUser(_ context.Context, rec repository.PortalUserRecord) error {
	cp := rec
	f.users[rec.ID] = &cp
	return nil
}

func (f *fakeRepo) ListPortalUsers(_ context.Context, ministryFilter string) ([]*repository.PortalUserRecord, error) {
	var out []*repository.PortalUserRecord
	for _, u := range f.users {
		if ministryFilter == "" || u.Ministry == ministryFilter {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdatePortalUserRole(_ context.Context, id, role string) error {
	u, ok := f.users[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.Role = role
	return nil
}

func (f *fakeRepo) CreateBulkOperation(_ context.Context, rec repository.BulkOperationRecord) error {
	cp := rec
	f.operations[rec.ID] = &cp
	return nil
}

func (f *fakeRepo) ListBulkOperations(_ context.Context, statusFilter, ministryFilter string) ([]*repository.BulkOperationRecord, error) {
	var out []*repository.BulkOperationRecord
	for _, op := range f.operations {
		if statusFilter != "" && op.Status != statusFilter {
			continue
		}
		if ministryFilter != "" && op.Ministry != ministryFilter {
			continue
		}
		cp := *op
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRepo) ApproveBulkOperation(_ context.Context, id, approvedBy, newStatus string) error {
	op, ok := f.operations[id]
	if !ok {
		return repository.ErrNotFound
	}
	op.ApprovedBy = approvedBy
	op.Status = newStatus
	return nil
}

func (f *fakeRepo) GetBulkOperationByID(_ context.Context, id string) (*repository.BulkOperationRecord, error) {
	op, ok := f.operations[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *op
	return &cp, nil
}

func (f *fakeRepo) CountRows(_ context.Context, tableName string) (int64, error) {
	switch tableName {
	case "portal_users":
		return int64(len(f.users)), nil
	case "bulk_operations":
		return int64(len(f.operations)), nil
	}
	return 0, nil
}

// helpers

func createUser(t *testing.T, svc *GovPortalService, username, ministry, role string) *repository.PortalUserRecord {
	t.Helper()
	u, err := svc.CreatePortalUser(context.Background(), username, ministry, role, "apikey123")
	if err != nil {
		t.Fatalf("CreatePortalUser failed: %v", err)
	}
	return u
}

// Tests

func TestCreatePortalUser_Success(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	u, err := svc.CreatePortalUser(context.Background(), "alice", "MOI", "operator", "plainkey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Error("expected user ID to be set")
	}
	// APIKeyHash should be sha256 of "plainkey" → 64 hex chars
	if len(u.APIKeyHash) != 64 {
		t.Errorf("expected APIKeyHash length 64, got %d", len(u.APIKeyHash))
	}
}

func TestCreatePortalUser_InvalidRole(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	_, err := svc.CreatePortalUser(context.Background(), "bob", "MOI", "superuser", "key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid role") {
		t.Errorf("expected 'invalid role' in error, got: %v", err)
	}
}

func TestCreatePortalUser_MissingUsername(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	_, err := svc.CreatePortalUser(context.Background(), "", "MOI", "viewer", "key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Errorf("expected 'username' in error, got: %v", err)
	}
}

func TestCreatePortalUser_MissingMinistry(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	_, err := svc.CreatePortalUser(context.Background(), "alice", "", "viewer", "key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ministry") {
		t.Errorf("expected 'ministry' in error, got: %v", err)
	}
}

func TestListPortalUsers_Empty(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	users, err := svc.ListPortalUsers(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected empty list, got %d", len(users))
	}
}

func TestListPortalUsers_WithFilter(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	createUser(t, svc, "alice", "MOI", "viewer")
	createUser(t, svc, "bob", "MOE", "viewer")

	users, err := svc.ListPortalUsers(context.Background(), "MOI")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
	if users[0].Ministry != "MOI" {
		t.Errorf("expected ministry MOI, got %s", users[0].Ministry)
	}
}

func TestAssignRole_Success(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo)
	u := createUser(t, svc, "alice", "MOI", "viewer")

	if err := svc.AssignRole(context.Background(), u.ID, "admin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	users, err := svc.ListPortalUsers(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found bool
	for _, usr := range users {
		if usr.ID == u.ID {
			found = true
			if usr.Role != "admin" {
				t.Errorf("expected role admin, got %s", usr.Role)
			}
		}
	}
	if !found {
		t.Error("user not found in list")
	}
}

func TestAssignRole_InvalidRole(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	if err := svc.AssignRole(context.Background(), "any-id", "superuser"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAssignRole_NotFound(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	err := svc.AssignRole(context.Background(), "nonexistent-id", "admin")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestCreateBulkOperation_Success(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	op, err := svc.CreateBulkOperation(context.Background(), "issue_credential", "MOI", "user-1", []string{"did:indis:a", "did:indis:b"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Status != "pending" {
		t.Errorf("expected status 'pending', got %s", op.Status)
	}
}

func TestCreateBulkOperation_EmptyTargetDIDs(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	_, err := svc.CreateBulkOperation(context.Background(), "issue_credential", "MOI", "user-1", []string{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "target_dids") {
		t.Errorf("expected 'target_dids' in error, got: %v", err)
	}
}

func TestApproveBulkOperation_Success(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	op, err := svc.CreateBulkOperation(context.Background(), "issue_credential", "MOI", "user-1", []string{"did:indis:a"}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	approved, err := svc.ApproveBulkOperation(context.Background(), op.ID, "approver-1")
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if approved.Status != "approved" {
		t.Errorf("expected status 'approved', got %s", approved.Status)
	}
}

func TestApproveBulkOperation_NotFound(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	_, err := svc.ApproveBulkOperation(context.Background(), "nonexistent-id", "approver-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetStats_Empty(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo())
	stats, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalPortalUsers != 0 {
		t.Errorf("expected 0 portal users, got %d", stats.TotalPortalUsers)
	}
	if stats.TotalBulkOperations != 0 {
		t.Errorf("expected 0 bulk operations, got %d", stats.TotalBulkOperations)
	}
	if stats.PendingBulkOperations != 0 {
		t.Errorf("expected 0 pending bulk operations, got %d", stats.PendingBulkOperations)
	}
}
