// Package handler implements HTTP handlers for the card service.
//
// Route table:
//
//	POST /v1/card/generate        → generate card data for a DID
//	GET  /v1/card/{did}           → get card data for a DID
//	POST /v1/card/{did}/invalidate → invalidate a card (lost/stolen)
//	GET  /v1/card/{did}/verify    → verify card authenticity
//	GET  /health                  → health check
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mohamadsolouki/INDIS/services/card/internal/repository"
	"github.com/mohamadsolouki/INDIS/services/card/internal/service"
)

// Handler is the HTTP handler for the card service.
type Handler struct {
	svc *service.Service
	mux *http.ServeMux
}

// New creates a Handler and registers all routes.
func New(svc *service.Service) *Handler {
	h := &Handler{
		svc: svc,
		mux: http.NewServeMux(),
	}
	h.routes()
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// routes registers all HTTP endpoints.
func (h *Handler) routes() {
	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/v1/card/", h.handleCard)
}

// handleHealth responds with a JSON status object.
func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleCard dispatches all /v1/card/* routes.
func (h *Handler) handleCard(w http.ResponseWriter, r *http.Request) {
	// Strip /v1/card/
	path := strings.TrimPrefix(r.URL.Path, "/v1/card/")
	// Remove trailing slash.
	path = strings.TrimRight(path, "/")

	switch {
	case path == "generate" && r.Method == http.MethodPost:
		h.handleGenerate(w, r)

	case path != "" && !strings.Contains(path, "/") && r.Method == http.MethodGet:
		// GET /v1/card/{did}
		h.handleGet(w, r, path)

	case strings.HasSuffix(path, "/invalidate") && r.Method == http.MethodPost:
		did := strings.TrimSuffix(path, "/invalidate")
		h.handleInvalidate(w, r, did)

	case strings.HasSuffix(path, "/verify") && r.Method == http.MethodGet:
		did := strings.TrimSuffix(path, "/verify")
		h.handleVerify(w, r, did)

	default:
		writeError(w, http.StatusNotFound, "unknown card route")
	}
}

// generateRequest is the JSON body for POST /v1/card/generate.
type generateRequest struct {
	DID         string `json:"did"`
	HolderName  string `json:"holder_name"`
	DateOfBirth string `json:"date_of_birth"` // YYMMDD
	ExpiryDate  string `json:"expiry_date"`   // YYMMDD
}

// handleGenerate handles POST /v1/card/generate.
func (h *Handler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var body generateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.DID == "" {
		writeError(w, http.StatusBadRequest, "did is required")
		return
	}
	if body.HolderName == "" {
		writeError(w, http.StatusBadRequest, "holder_name is required")
		return
	}
	if len(body.DateOfBirth) != 6 {
		writeError(w, http.StatusBadRequest, "date_of_birth must be YYMMDD (6 digits)")
		return
	}
	if len(body.ExpiryDate) != 6 {
		writeError(w, http.StatusBadRequest, "expiry_date must be YYMMDD (6 digits)")
		return
	}

	req := service.GenerateRequest{
		DID:         body.DID,
		HolderName:  body.HolderName,
		DateOfBirth: body.DateOfBirth,
		ExpiryDate:  body.ExpiryDate,
	}

	card, err := h.svc.GenerateCard(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, card)
}

// handleGet handles GET /v1/card/{did}.
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request, did string) {
	if did == "" {
		writeError(w, http.StatusBadRequest, "did is required")
		return
	}
	card, err := h.svc.GetCard(r.Context(), did)
	if errors.Is(err, repository.ErrNotFound) || (err != nil && isNotFound(err)) {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, card)
}

// invalidateRequest is the optional JSON body for POST /v1/card/{did}/invalidate.
type invalidateRequest struct {
	Reason string `json:"reason"`
}

// handleInvalidate handles POST /v1/card/{did}/invalidate.
func (h *Handler) handleInvalidate(w http.ResponseWriter, r *http.Request, did string) {
	if did == "" {
		writeError(w, http.StatusBadRequest, "did is required")
		return
	}

	var body invalidateRequest
	// Body is optional; ignore decode errors.
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := body.Reason
	if reason == "" {
		reason = "unspecified"
	}

	if err := h.svc.InvalidateCard(r.Context(), did, reason); err != nil {
		if errors.Is(err, repository.ErrNotFound) || isNotFound(err) {
			writeError(w, http.StatusNotFound, "card not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "invalidated"})
}

// verifyResponse is the JSON body for GET /v1/card/{did}/verify.
type verifyResponse struct {
	Valid bool `json:"valid"`
}

// handleVerify handles GET /v1/card/{did}/verify.
func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request, did string) {
	if did == "" {
		writeError(w, http.StatusBadRequest, "did is required")
		return
	}
	valid, err := h.svc.VerifyCard(r.Context(), did)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || isNotFound(err) {
			writeError(w, http.StatusNotFound, "card not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, verifyResponse{Valid: valid})
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

// isNotFound returns true when the error message contains "not found" — used
// to unwrap wrapped repository errors that cross package boundaries.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
