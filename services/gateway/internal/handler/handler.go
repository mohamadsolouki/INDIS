// Package handler implements HTTP handlers for the INDIS API gateway.
//
// Route table:
//
//	GET  /health                              → health check (pings all backends)
//	POST /v1/identity/register                → IdentityService.RegisterIdentity
//	GET  /v1/identity/{did}                   → IdentityService.ResolveIdentity
//	POST /v1/identity/{did}/deactivate        → IdentityService.DeactivateIdentity
//	POST /v1/credential/issue                 → CredentialService.IssueCredential
//	GET  /v1/credential/{id}                  → CredentialService.CheckRevocationStatus
//	POST /v1/credential/{id}/revoke           → CredentialService.RevokeCredential
//	POST /v1/enrollment/initiate              → EnrollmentService.InitiateEnrollment
//	POST /v1/enrollment/{id}/biometrics       → EnrollmentService.SubmitBiometrics
//	POST /v1/enrollment/{id}/attestation      → EnrollmentService.SubmitSocialAttestation
//	POST /v1/enrollment/{id}/complete         → EnrollmentService.CompleteEnrollment
//	GET  /v1/enrollment/{id}                  → EnrollmentService.GetEnrollmentStatus
//	POST /v1/electoral/elections              → ElectoralService.RegisterElection
//	POST /v1/electoral/verify                 → ElectoralService.VerifyEligibility
//	POST /v1/electoral/ballot                 → ElectoralService.CastBallot
//	GET  /v1/electoral/elections/{id}         → ElectoralService.GetElectionStatus
//	POST /v1/justice/testimony                → JusticeService.SubmitTestimony
//	POST /v1/justice/testimony/link           → JusticeService.LinkTestimony
//	POST /v1/justice/amnesty                  → JusticeService.InitiateAmnesty
//	GET  /v1/justice/cases/{id}               → JusticeService.GetCaseStatus
//	GET  /v1/notification/status/{id}         → 501 Not Implemented (proto has no GetStatus)
//	POST /v1/notification/send                → NotificationService.Send
//	POST /v1/notification/alert               → NotificationService.ScheduleExpiryAlert
//	GET  /v1/biometric/deduplicate-status/{id} → 200 stub (mobile polling)
//	POST /v1/audit/events                     → AuditService.AppendEvent  (API-key only)
//	GET  /v1/audit/events                     → AuditService.QueryEvents  (ministry role)
//	POST /v1/verifier/register                → proxy to VERIFIER_HTTP_URL
//	GET  /v1/verifier/{id}                    → proxy to VERIFIER_HTTP_URL
//	POST /v1/verifier/verify                  → proxy to VERIFIER_HTTP_URL (public)
//	GET  /v1/card/{did}                       → proxy to CARD_HTTP_URL
//	POST /v1/card/generate                    → proxy to CARD_HTTP_URL
//	GET  /v1/privacy/history                  → audit QueryEvents filtered by subject_did
//	GET  /v1/privacy/sharing                  → audit QueryEvents for credential.verify actions
//	POST /v1/privacy/consent                  → create consent rule
//	GET  /v1/privacy/consent                  → list consent rules
//	DELETE /v1/privacy/consent/{id}           → delete consent rule
//	POST /v1/privacy/data-export              → request data export
//	GET  /v1/privacy/data-export/{id}         → check data-export status
package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	notificationv1 "github.com/IranProsperityProject/INDIS/api/gen/go/notification/v1"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/auth"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/circuitbreaker"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/proxy"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/ratelimit"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/repository"
)

const rpcTimeout = 10 * time.Second

// Gateway is the HTTP handler for the INDIS API gateway.
type Gateway struct {
	clients        *proxy.Clients
	limiter        *ratelimit.Limiter
	repo           *repository.Repository
	verifierHTTPURL string
	cardHTTPURL     string
	govPortalHTTPURL string
	mux            *http.ServeMux
}

// New creates a new Gateway with all routes registered.
// repo may be nil when the gateway is run without a local database (disables privacy APIs).
func New(clients *proxy.Clients, limiter *ratelimit.Limiter, repo *repository.Repository, verifierHTTPURL, cardHTTPURL, govPortalHTTPURL string) *Gateway {
	g := &Gateway{
		clients:        clients,
		limiter:        limiter,
		repo:           repo,
		verifierHTTPURL: verifierHTTPURL,
		cardHTTPURL:     cardHTTPURL,
		govPortalHTTPURL: govPortalHTTPURL,
		mux:            http.NewServeMux(),
	}
	g.routes()
	return g
}

// ServeHTTP implements http.Handler.
// It enforces per-IP rate limiting and applies security headers before dispatching.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !g.limiter.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	applySecurityHeaders(w)
	g.mux.ServeHTTP(w, r)
}

// applySecurityHeaders sets OWASP-recommended defensive HTTP headers on every response.
func applySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// routes registers all HTTP routes.
func (g *Gateway) routes() {
	g.mux.HandleFunc("/health", g.handleHealth)

	// Identity
	g.mux.HandleFunc("/v1/identity/", g.handleIdentity)

	// Credential
	g.mux.HandleFunc("/v1/credential/", g.handleCredential)

	// Enrollment
	g.mux.HandleFunc("/v1/enrollment/", g.handleEnrollment)

	// Electoral
	g.mux.HandleFunc("/v1/electoral/", g.handleElectoral)

	// Justice
	g.mux.HandleFunc("/v1/justice/", g.handleJustice)

	// Notification
	g.mux.HandleFunc("/v1/notification/", g.handleNotification)

	// Biometric (stub for mobile polling)
	g.mux.HandleFunc("/v1/biometric/", g.handleBiometric)

	// Audit
	g.mux.HandleFunc("/v1/audit/", g.handleAudit)

	// Verifier (HTTP proxy)
	g.mux.HandleFunc("/v1/verifier/", g.handleVerifier)

	// Card (HTTP proxy)
	g.mux.HandleFunc("/v1/card/", g.handleCard)

	// Privacy Control Center (PRD §FR-008)
	g.mux.HandleFunc("/v1/privacy/", g.handlePrivacy)

	// Gov portal (REST + minimal GraphQL)
	g.mux.HandleFunc("/graphql", g.handleGovPortalGraphQL)
	g.mux.HandleFunc("/v1/portal/", g.handleGovPortal)
}

// ── Health ────────────────────────────────────────────────────────────────────

func (g *Gateway) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── Identity ──────────────────────────────────────────────────────────────────

func (g *Gateway) handleIdentity(w http.ResponseWriter, r *http.Request) {
	// Strip /v1/identity/
	path := strings.TrimPrefix(r.URL.Path, "/v1/identity/")
	parts := strings.SplitN(path, "/", 2)

	switch {
	case path == "" || path == "register":
		// POST /v1/identity/register
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req identityv1.RegisterIdentityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *identityv1.RegisterIdentityResponse
		if !cbCall(w, g.clients.CBIdentity, func() error {
			var err error
			resp, err = g.clients.Identity.RegisterIdentity(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case len(parts) == 1 && r.Method == http.MethodGet:
		// GET /v1/identity/{did}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *identityv1.ResolveIdentityResponse
		if !cbCall(w, g.clients.CBIdentity, func() error {
			var err error
			resp, err = g.clients.Identity.ResolveIdentity(ctx, &identityv1.ResolveIdentityRequest{Did: parts[0]})
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case len(parts) == 2 && parts[1] == "deactivate" && r.Method == http.MethodPost:
		// POST /v1/identity/{did}/deactivate
		var req identityv1.DeactivateIdentityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.Did = parts[0]
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *identityv1.DeactivateIdentityResponse
		if !cbCall(w, g.clients.CBIdentity, func() error {
			var err error
			resp, err = g.clients.Identity.DeactivateIdentity(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown identity route")
	}
}

// ── Credential ────────────────────────────────────────────────────────────────

func (g *Gateway) handleCredential(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/credential/")
	parts := strings.SplitN(path, "/", 2)

	switch {
	case path == "issue" && r.Method == http.MethodPost:
		var req credentialv1.IssueCredentialRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *credentialv1.IssueCredentialResponse
		if !cbCall(w, g.clients.CBCredential, func() error {
			var err error
			resp, err = g.clients.Credential.IssueCredential(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case len(parts) == 1 && r.Method == http.MethodGet:
		// GET /v1/credential/{id} → check revocation status
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *credentialv1.CheckRevocationStatusResponse
		if !cbCall(w, g.clients.CBCredential, func() error {
			var err error
			resp, err = g.clients.Credential.CheckRevocationStatus(ctx, &credentialv1.CheckRevocationStatusRequest{CredentialId: parts[0]})
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case len(parts) == 2 && parts[1] == "revoke" && r.Method == http.MethodPost:
		var req credentialv1.RevokeCredentialRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.CredentialId = parts[0]
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *credentialv1.RevokeCredentialResponse
		if !cbCall(w, g.clients.CBCredential, func() error {
			var err error
			resp, err = g.clients.Credential.RevokeCredential(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown credential route")
	}
}

// ── Enrollment ────────────────────────────────────────────────────────────────

func (g *Gateway) handleEnrollment(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/enrollment/")
	parts := strings.SplitN(path, "/", 2)

	switch {
	case path == "initiate" && r.Method == http.MethodPost:
		var req enrollmentv1.InitiateEnrollmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *enrollmentv1.InitiateEnrollmentResponse
		if !cbCall(w, g.clients.CBEnrollment, func() error {
			var err error
			resp, err = g.clients.Enrollment.InitiateEnrollment(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case len(parts) == 1 && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *enrollmentv1.GetEnrollmentStatusResponse
		if !cbCall(w, g.clients.CBEnrollment, func() error {
			var err error
			resp, err = g.clients.Enrollment.GetEnrollmentStatus(ctx, &enrollmentv1.GetEnrollmentStatusRequest{EnrollmentId: parts[0]})
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case len(parts) == 2 && parts[1] == "biometrics" && r.Method == http.MethodPost:
		var req enrollmentv1.SubmitBiometricsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.EnrollmentId = parts[0]
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *enrollmentv1.SubmitBiometricsResponse
		if !cbCall(w, g.clients.CBEnrollment, func() error {
			var err error
			resp, err = g.clients.Enrollment.SubmitBiometrics(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case len(parts) == 2 && parts[1] == "attestation" && r.Method == http.MethodPost:
		var req enrollmentv1.SubmitSocialAttestationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.EnrollmentId = parts[0]
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *enrollmentv1.SubmitSocialAttestationResponse
		if !cbCall(w, g.clients.CBEnrollment, func() error {
			var err error
			resp, err = g.clients.Enrollment.SubmitSocialAttestation(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case len(parts) == 2 && parts[1] == "complete" && r.Method == http.MethodPost:
		var req enrollmentv1.CompleteEnrollmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.EnrollmentId = parts[0]
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *enrollmentv1.CompleteEnrollmentResponse
		if !cbCall(w, g.clients.CBEnrollment, func() error {
			var err error
			resp, err = g.clients.Enrollment.CompleteEnrollment(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown enrollment route")
	}
}

// ── Electoral ─────────────────────────────────────────────────────────────────

func (g *Gateway) handleElectoral(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/electoral/")
	parts := strings.SplitN(path, "/", 2)

	switch {
	case path == "elections" && r.Method == http.MethodPost:
		var req electoralv1.RegisterElectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *electoralv1.RegisterElectionResponse
		if !cbCall(w, g.clients.CBElectoral, func() error {
			var err error
			resp, err = g.clients.Electoral.RegisterElection(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case path == "verify" && r.Method == http.MethodPost:
		bodyBytes, ok := validateProofSize(w, r, 100_000)
		if !ok {
			return
		}
		var req electoralv1.VerifyEligibilityRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *electoralv1.VerifyEligibilityResponse
		if !cbCall(w, g.clients.CBElectoral, func() error {
			var err error
			resp, err = g.clients.Electoral.VerifyEligibility(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case path == "ballot" && r.Method == http.MethodPost:
		bodyBytes, ok := validateProofSize(w, r, 100_000)
		if !ok {
			return
		}
		var req electoralv1.CastBallotRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *electoralv1.CastBallotResponse
		if !cbCall(w, g.clients.CBElectoral, func() error {
			var err error
			resp, err = g.clients.Electoral.CastBallot(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case parts[0] == "elections" && len(parts) == 2 && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *electoralv1.GetElectionStatusResponse
		if !cbCall(w, g.clients.CBElectoral, func() error {
			var err error
			resp, err = g.clients.Electoral.GetElectionStatus(ctx, &electoralv1.GetElectionStatusRequest{ElectionId: parts[1]})
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown electoral route")
	}
}

// ── Justice ───────────────────────────────────────────────────────────────────

func (g *Gateway) handleJustice(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/justice/")
	parts := strings.SplitN(path, "/", 2)

	switch {
	case path == "testimony" && r.Method == http.MethodPost:
		bodyBytes, ok := validateProofSize(w, r, 100_000)
		if !ok {
			return
		}
		var req justicev1.SubmitTestimonyRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *justicev1.SubmitTestimonyResponse
		if !cbCall(w, g.clients.CBJustice, func() error {
			var err error
			resp, err = g.clients.Justice.SubmitTestimony(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case path == "testimony/link" && r.Method == http.MethodPost:
		var req justicev1.LinkTestimonyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *justicev1.LinkTestimonyResponse
		if !cbCall(w, g.clients.CBJustice, func() error {
			var err error
			resp, err = g.clients.Justice.LinkTestimony(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case path == "amnesty" && r.Method == http.MethodPost:
		var req justicev1.InitiateAmnestyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *justicev1.InitiateAmnestyResponse
		if !cbCall(w, g.clients.CBJustice, func() error {
			var err error
			resp, err = g.clients.Justice.InitiateAmnesty(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case parts[0] == "cases" && len(parts) == 2 && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *justicev1.GetCaseStatusResponse
		if !cbCall(w, g.clients.CBJustice, func() error {
			var err error
			resp, err = g.clients.Justice.GetCaseStatus(ctx, &justicev1.GetCaseStatusRequest{CaseId: parts[1]})
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown justice route")
	}
}

// ── Notification ──────────────────────────────────────────────────────────────

func (g *Gateway) handleNotification(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/notification/")

	switch {
	case path == "send" && r.Method == http.MethodPost:
		var req notificationv1.SendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *notificationv1.SendResponse
		if !cbCall(w, g.clients.CBNotification, func() error {
			var err error
			resp, err = g.clients.Notification.Send(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case path == "alert" && r.Method == http.MethodPost:
		var req notificationv1.ScheduleExpiryAlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *notificationv1.ScheduleExpiryAlertResponse
		if !cbCall(w, g.clients.CBNotification, func() error {
			var err error
			resp, err = g.clients.Notification.ScheduleExpiryAlert(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case strings.HasPrefix(path, "status/") && r.Method == http.MethodGet:
		// NotificationService proto does not expose a GetStatus RPC.
		writeError(w, http.StatusNotImplemented, "notification status lookup not yet implemented")

	default:
		writeError(w, http.StatusNotFound, "unknown notification route")
	}
}

// ── Biometric ─────────────────────────────────────────────────────────────────

func (g *Gateway) handleBiometric(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/biometric/")

	switch {
	case strings.HasPrefix(path, "deduplicate-status/") && r.Method == http.MethodGet:
		// Mobile polling stub. A real implementation would query the AI deduplication service.
		id := strings.TrimPrefix(path, "deduplicate-status/")
		writeJSON(w, http.StatusOK, map[string]string{
			"id":     id,
			"status": "pending",
		})

	default:
		writeError(w, http.StatusNotFound, "unknown biometric route")
	}
}

// ── Audit ─────────────────────────────────────────────────────────────────────

func (g *Gateway) handleAudit(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/audit/")

	switch {
	case path == "events" && r.Method == http.MethodPost:
		// Internal: requires API-key authentication (service-to-service).
		if auth.RoleFromContext(r.Context()) != "service" {
			writeError(w, http.StatusForbidden, "API key required for audit append")
			return
		}
		var req auditv1.AppendEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *auditv1.AppendEventResponse
		if !cbCall(w, g.clients.CBAudit, func() error {
			var err error
			resp, err = g.clients.Audit.AppendEvent(ctx, &req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case path == "events" && r.Method == http.MethodGet:
		// Requires ministry role.
		if auth.RoleFromContext(r.Context()) == "" {
			writeError(w, http.StatusForbidden, "authentication required for audit query")
			return
		}
		q := r.URL.Query()
		req := &auditv1.QueryEventsRequest{
			ActorDid:   q.Get("actor_did"),
			SubjectDid: q.Get("subject_did"),
			FromTime:   q.Get("from"),
			ToTime:     q.Get("to"),
			PageToken:  q.Get("page_token"),
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		var resp *auditv1.QueryEventsResponse
		if !cbCall(w, g.clients.CBAudit, func() error {
			var err error
			resp, err = g.clients.Audit.QueryEvents(ctx, req)
			return err
		}) {
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown audit route")
	}
}

// ── Verifier (HTTP proxy) ─────────────────────────────────────────────────────

func (g *Gateway) handleVerifier(w http.ResponseWriter, r *http.Request) {
	// POST /v1/verifier/verify is public (ZK verification); all others require auth.
	// Auth is enforced by the auth middleware upstream; verifier/verify is in publicRoutes.
	path := strings.TrimPrefix(r.URL.Path, "/v1/verifier")
	g.proxyHTTP(w, r, g.verifierHTTPURL+"/v1/verifier"+path)
}

// ── Card (HTTP proxy) ─────────────────────────────────────────────────────────

func (g *Gateway) handleCard(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/card")
	g.proxyHTTP(w, r, g.cardHTTPURL+"/v1/card"+path)
}

// ── Gov Portal (HTTP proxy) ───────────────────────────────────────────────────

func (g *Gateway) handleGovPortal(w http.ResponseWriter, r *http.Request) {
	// /v1/portal/{...} → govportal:8200/v1/portal/{...}
	path := strings.TrimPrefix(r.URL.Path, "/v1/portal")
	target := strings.TrimRight(g.govPortalHTTPURL, "/") + "/v1/portal" + path
	g.proxyHTTP(w, r, target)
}

func (g *Gateway) handleGovPortalGraphQL(w http.ResponseWriter, r *http.Request) {
	// /graphql → govportal:8200/graphql
	target := strings.TrimRight(g.govPortalHTTPURL, "/") + r.URL.Path
	g.proxyHTTP(w, r, target)
}

// proxyHTTP forwards the incoming request to targetURL and streams the response back.
// Request headers are forwarded; X-Forwarded-For is appended.
func (g *Gateway) proxyHTTP(w http.ResponseWriter, r *http.Request, targetURL string) {
	client := &http.Client{Timeout: rpcTimeout}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("proxy: build request: %v", err))
		return
	}

	// Forward all incoming headers.
	for k, vv := range r.Header {
		for _, v := range vv {
			proxyReq.Header.Add(k, v)
		}
	}
	proxyReq.Header.Set("X-Forwarded-For", clientIP(r))

	resp, err := client.Do(proxyReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("proxy: upstream error: %v", err))
		return
	}
	defer resp.Body.Close()

	// Copy response headers.
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// ── Privacy Control Center (PRD §FR-008) ──────────────────────────────────────

func (g *Gateway) handlePrivacy(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/privacy/")
	parts := strings.SplitN(path, "/", 2)

	citizenDID := auth.DIDFromContext(r.Context())
	if citizenDID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	switch {
	// GET /v1/privacy/history
	case path == "history" && r.Method == http.MethodGet:
		g.handlePrivacyHistory(w, r, citizenDID)

	// GET /v1/privacy/sharing
	case path == "sharing" && r.Method == http.MethodGet:
		g.handlePrivacySharing(w, r, citizenDID)

	// POST /v1/privacy/consent
	case path == "consent" && r.Method == http.MethodPost:
		g.handlePrivacyConsentCreate(w, r, citizenDID)

	// GET /v1/privacy/consent
	case path == "consent" && r.Method == http.MethodGet:
		g.handlePrivacyConsentList(w, r, citizenDID)

	// DELETE /v1/privacy/consent/{id}
	case parts[0] == "consent" && len(parts) == 2 && r.Method == http.MethodDelete:
		g.handlePrivacyConsentDelete(w, r, citizenDID, parts[1])

	// POST /v1/privacy/data-export
	case path == "data-export" && r.Method == http.MethodPost:
		g.handlePrivacyDataExportCreate(w, r, citizenDID)

	// GET /v1/privacy/data-export/{id}
	case parts[0] == "data-export" && len(parts) == 2 && r.Method == http.MethodGet:
		g.handlePrivacyDataExportGet(w, r, citizenDID, parts[1])

	default:
		writeError(w, http.StatusNotFound, "unknown privacy route")
	}
}

// handlePrivacyHistory lists verification requests received by the citizen by
// querying the audit service for events where subject_did equals the citizen's DID.
func (g *Gateway) handlePrivacyHistory(w http.ResponseWriter, r *http.Request, citizenDID string) {
	q := r.URL.Query()
	req := &auditv1.QueryEventsRequest{
		SubjectDid: citizenDID,
		FromTime:   q.Get("from"),
		ToTime:     q.Get("to"),
		PageToken:  q.Get("page_token"),
	}
	ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
	defer cancel()
	var resp *auditv1.QueryEventsResponse
	if !cbCall(w, g.clients.CBAudit, func() error {
		var err error
		resp, err = g.clients.Audit.QueryEvents(ctx, req)
		return err
	}) {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePrivacySharing lists credentials shared and with whom by querying audit events
// for credential.verify actions where the actor_did is a verifier.
func (g *Gateway) handlePrivacySharing(w http.ResponseWriter, r *http.Request, citizenDID string) {
	q := r.URL.Query()
	req := &auditv1.QueryEventsRequest{
		SubjectDid: citizenDID,
		FromTime:   q.Get("from"),
		ToTime:     q.Get("to"),
		PageToken:  q.Get("page_token"),
		Category:   auditv1.EventCategory_EVENT_CATEGORY_CREDENTIAL,
	}
	ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
	defer cancel()
	var resp *auditv1.QueryEventsResponse
	if !cbCall(w, g.clients.CBAudit, func() error {
		var err error
		resp, err = g.clients.Audit.QueryEvents(ctx, req)
		return err
	}) {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// consentRuleRequest is the JSON body for POST /v1/privacy/consent.
type consentRuleRequest struct {
	VerifierCategory string `json:"verifier_category"`
	CredentialType   string `json:"credential_type"`
	Rule             string `json:"rule"` // "always" | "ask" | "never"
}

func (g *Gateway) handlePrivacyConsentCreate(w http.ResponseWriter, r *http.Request, citizenDID string) {
	if g.repo == nil {
		writeError(w, http.StatusServiceUnavailable, "privacy consent storage not configured")
		return
	}
	var body consentRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.VerifierCategory == "" || body.CredentialType == "" {
		writeError(w, http.StatusBadRequest, "verifier_category and credential_type are required")
		return
	}
	switch body.Rule {
	case "always", "ask", "never":
	default:
		writeError(w, http.StatusBadRequest, `rule must be "always", "ask", or "never"`)
		return
	}
	id, err := generateID("cr_")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate ID")
		return
	}
	rule := repository.ConsentRule{
		ID:               id,
		CitizenDID:       citizenDID,
		VerifierCategory: body.VerifierCategory,
		CredentialType:   body.CredentialType,
		Rule:             body.Rule,
		CreatedAt:        time.Now().UTC(),
	}
	if err := g.repo.InsertConsentRule(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (g *Gateway) handlePrivacyConsentList(w http.ResponseWriter, r *http.Request, citizenDID string) {
	if g.repo == nil {
		writeError(w, http.StatusServiceUnavailable, "privacy consent storage not configured")
		return
	}
	rules, err := g.repo.ListConsentRules(r.Context(), citizenDID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rules == nil {
		rules = []repository.ConsentRule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"consent_rules": rules})
}

func (g *Gateway) handlePrivacyConsentDelete(w http.ResponseWriter, r *http.Request, citizenDID, ruleID string) {
	if g.repo == nil {
		writeError(w, http.StatusServiceUnavailable, "privacy consent storage not configured")
		return
	}
	if err := g.repo.DeleteConsentRule(r.Context(), ruleID, citizenDID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "consent rule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// dataExportCreateRequest is the optional JSON body for POST /v1/privacy/data-export.
type dataExportCreateRequest struct {
	// Reserved for future filter parameters (e.g., date range, credential types).
}

func (g *Gateway) handlePrivacyDataExportCreate(w http.ResponseWriter, r *http.Request, citizenDID string) {
	if g.repo == nil {
		writeError(w, http.StatusServiceUnavailable, "privacy data-export storage not configured")
		return
	}
	id, err := generateID("dex_")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate ID")
		return
	}
	req := repository.DataExportRequest{
		ID:          id,
		CitizenDID:  citizenDID,
		Status:      "pending",
		RequestedAt: time.Now().UTC(),
	}
	if err := g.repo.InsertDataExportRequest(r.Context(), req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"request_id": id,
		"status":     "pending",
	})
}

func (g *Gateway) handlePrivacyDataExportGet(w http.ResponseWriter, r *http.Request, citizenDID, exportID string) {
	if g.repo == nil {
		writeError(w, http.StatusServiceUnavailable, "privacy data-export storage not configured")
		return
	}
	rec, err := g.repo.GetDataExportRequest(r.Context(), exportID, citizenDID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "data export request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeGRPCError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusBadGateway, err.Error())
}

// writeCircuitOpen writes an HTTP 503 response when a backend circuit is open.
func writeCircuitOpen(w http.ResponseWriter) {
	writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable")
}

// cbCall executes fn only if the circuit breaker cb allows it.
// On success it calls cb.RecordSuccess(); on failure cb.RecordFailure().
// Returns false (and writes 503) when the circuit is open.
func cbCall(w http.ResponseWriter, cb *circuitbreaker.CircuitBreaker, fn func() error) bool {
	if !cb.Allow() {
		writeCircuitOpen(w)
		return false
	}
	if err := fn(); err != nil {
		cb.RecordFailure()
		writeGRPCError(w, err)
		return false
	}
	cb.RecordSuccess()
	return true
}

// clientIP extracts the real client IP from X-Forwarded-For or RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// validateProofSize reads r.Body up to maxBytes+1 bytes. If the body fits within
// maxBytes it returns the bytes and true. If the body exceeds maxBytes it writes
// HTTP 400 and returns nil, false. The caller must not read r.Body after this call.
func validateProofSize(w http.ResponseWriter, r *http.Request, maxBytes int) ([]byte, bool) {
	limited := io.LimitReader(r.Body, int64(maxBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return nil, false
	}
	if len(data) > maxBytes {
		writeError(w, http.StatusBadRequest, "proof exceeds maximum size")
		return nil, false
	}
	// Check proof_b64 field specifically if present.
	var envelope struct {
		ProofB64 string `json:"proof_b64"`
	}
	if jsonErr := json.Unmarshal(data, &envelope); jsonErr == nil && envelope.ProofB64 != "" {
		decoded, decErr := base64.StdEncoding.DecodeString(envelope.ProofB64)
		if decErr != nil {
			// Try URL-safe variant.
			decoded, decErr = base64.RawURLEncoding.DecodeString(envelope.ProofB64)
		}
		if decErr == nil && len(decoded) > maxBytes {
			writeError(w, http.StatusBadRequest, "proof exceeds maximum size")
			return nil, false
		}
	}
	return data, true
}

// generateID produces a random URL-safe ID with the given prefix.
func generateID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
