package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	"github.com/IranProsperityProject/INDIS/services/govportal/internal/repository"
	govportalservice "github.com/IranProsperityProject/INDIS/services/govportal/internal/service"
)

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
	out := make([]*repository.PortalUserRecord, 0, len(f.users))
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

func (f *fakeRepo) CreateBulkOperation(_ context.Context, _ repository.BulkOperationRecord) error {
	return nil
}

func (f *fakeRepo) ListBulkOperations(_ context.Context, _, _ string) ([]*repository.BulkOperationRecord, error) {
	return nil, nil
}

func (f *fakeRepo) ApproveBulkOperation(_ context.Context, _, _ string, _ string) error {
	return nil
}

func (f *fakeRepo) SetBulkOperationResult(_ context.Context, _, _, _ string, _ json.RawMessage) error {
	return nil
}

func (f *fakeRepo) GetBulkOperationByID(_ context.Context, _ string) (*repository.BulkOperationRecord, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeRepo) CountRows(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestLoginEndpoint_MintsValidJWT(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	repo.users["user-1"] = &repository.PortalUserRecord{
		ID:         "user-1",
		Username:   "alice",
		Ministry:   "MOI",
		Role:       "admin",
		APIKeyHash: sha256Hex("secret"),
		CreatedAt:  time.Now().UTC(),
	}

	svc := govportalservice.New(repo)
	h := New(svc, "test-jwt-secret")

	body := `{"username":"alice","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/portal/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Token == "" {
		t.Fatalf("expected token, got empty")
	}

	authReq := httptest.NewRequest(http.MethodPost, "/v1/portal/users", nil)
	authReq.Header.Set("Authorization", "Bearer "+out.Token)
	claims, err := h.extractJWT(authReq)
	if err != nil {
		t.Fatalf("extractJWT failed: %v", err)
	}

	if claims.Sub != "user-1" {
		t.Fatalf("expected sub user-1, got %q", claims.Sub)
	}
	if claims.Ministry != "MOI" {
		t.Fatalf("expected ministry MOI, got %q", claims.Ministry)
	}
	if claims.Role != "admin" {
		t.Fatalf("expected role admin, got %q", claims.Role)
	}
	if claims.Exp < time.Now().Unix() {
		t.Fatalf("expected exp in the future, got %d", claims.Exp)
	}
}

func TestAssignRoleEndpoint_UpdatesRole(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	// Actor (admin) used for authorization.
	repo.users["actor-1"] = &repository.PortalUserRecord{
		ID:         "actor-1",
		Username:   "admin",
		Ministry:   "MOI",
		Role:       "admin",
		APIKeyHash: sha256Hex("pass"),
		CreatedAt:  time.Now().UTC(),
	}
	// Target user whose role will be updated.
	repo.users["target-1"] = &repository.PortalUserRecord{
		ID:         "target-1",
		Username:   "bob",
		Ministry:   "MOI",
		Role:       "viewer",
		APIKeyHash: sha256Hex("unused"),
		CreatedAt:  time.Now().UTC(),
	}

	svc := govportalservice.New(repo)
	h := New(svc, "test-jwt-secret")

	// Login to mint JWT token.
	loginReq := httptest.NewRequest(http.MethodPost, "/v1/portal/auth/login", strings.NewReader(`{"username":"admin","password":"pass"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}

	var loginOut struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginOut); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/v1/portal/users/target-1/role", strings.NewReader(`{"role":"operator"}`))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+loginOut.Token)

	putRec := httptest.NewRecorder()
	h.ServeHTTP(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	if repo.users["target-1"].Role != "operator" {
		t.Fatalf("expected updated role operator, got %q", repo.users["target-1"].Role)
	}
}

// Compile-time guards for any unused imports in this test file.
var (
	_ auditv1.AuditServiceClient
	_ credentialv1.CredentialServiceClient
	_ enrollmentv1.EnrollmentServiceClient
	_ identityv1.IdentityServiceClient
)

