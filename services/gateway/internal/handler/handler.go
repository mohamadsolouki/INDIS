// Package handler implements HTTP handlers for the INDIS API gateway.
//
// Route table:
//
//	GET  /health                              → health check (pings all backends)
//	POST /v1/identity/register                → IdentityService.RegisterIdentity
//	GET  /v1/identity/{did}                   → IdentityService.ResolveIdentity
//	POST /v1/identity/{did}/deactivate        → IdentityService.DeactivateIdentity
//	POST /v1/credential/issue                 → CredentialService.IssueCredential
//	GET  /v1/credential/{id}                  → CredentialService.VerifyCredential
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
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/proxy"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/ratelimit"
)

const rpcTimeout = 10 * time.Second

// Gateway is the HTTP handler for the INDIS API gateway.
type Gateway struct {
	clients *proxy.Clients
	limiter *ratelimit.Limiter
	mux     *http.ServeMux
}

// New creates a new Gateway with all routes registered.
func New(clients *proxy.Clients, limiter *ratelimit.Limiter) *Gateway {
	g := &Gateway{
		clients: clients,
		limiter: limiter,
		mux:     http.NewServeMux(),
	}
	g.routes()
	return g
}

// ServeHTTP implements http.Handler.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !g.limiter.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	g.mux.ServeHTTP(w, r)
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
		resp, err := g.clients.Identity.RegisterIdentity(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case len(parts) == 1 && r.Method == http.MethodGet:
		// GET /v1/identity/{did}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Identity.ResolveIdentity(ctx, &identityv1.ResolveIdentityRequest{Did: parts[0]})
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Identity.DeactivateIdentity(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Credential.IssueCredential(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case len(parts) == 1 && r.Method == http.MethodGet:
		// GET /v1/credential/{id}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Credential.VerifyCredential(ctx, &credentialv1.VerifyCredentialRequest{CredentialId: parts[0]})
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Credential.RevokeCredential(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Enrollment.InitiateEnrollment(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case len(parts) == 1 && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Enrollment.GetEnrollmentStatus(ctx, &enrollmentv1.GetEnrollmentStatusRequest{EnrollmentId: parts[0]})
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Enrollment.SubmitBiometrics(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Enrollment.SubmitSocialAttestation(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Enrollment.CompleteEnrollment(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Electoral.RegisterElection(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case path == "verify" && r.Method == http.MethodPost:
		var req electoralv1.VerifyEligibilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Electoral.VerifyEligibility(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case path == "ballot" && r.Method == http.MethodPost:
		var req electoralv1.CastBallotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Electoral.CastBallot(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case parts[0] == "elections" && len(parts) == 2 && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Electoral.GetElectionStatus(ctx, &electoralv1.GetElectionStatusRequest{ElectionId: parts[1]})
		if err != nil {
			writeGRPCError(w, err)
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
		var req justicev1.SubmitTestimonyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Justice.SubmitTestimony(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Justice.LinkTestimony(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
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
		resp, err := g.clients.Justice.InitiateAmnesty(ctx, &req)
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, resp)

	case parts[0] == "cases" && len(parts) == 2 && r.Method == http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
		defer cancel()
		resp, err := g.clients.Justice.GetCaseStatus(ctx, &justicev1.GetCaseStatusRequest{CaseId: parts[1]})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusNotFound, "unknown justice route")
	}
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
