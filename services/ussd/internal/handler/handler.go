// Package handler implements HTTP handlers for the USSD/SMS gateway service.
//
// Route table:
//
//	POST /ussd               → USSD gateway callback (plain-text response)
//	POST /v1/sms/otp/send    → generate and send OTP (JSON)
//	POST /v1/sms/otp/verify  → verify OTP (JSON)
//	GET  /health             → health check (JSON)
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IranProsperityProject/INDIS/services/ussd/internal/service"
)

// Handler is the HTTP handler for the USSD/SMS service.
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
	h.mux.HandleFunc("/ussd", h.handleUSSD)
	h.mux.HandleFunc("/v1/sms/otp/send", h.handleOTPSend)
	h.mux.HandleFunc("/v1/sms/otp/verify", h.handleOTPVerify)
}

// handleHealth responds with a JSON status object.
func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleUSSD processes a USSD callback from a telecom gateway.
// The request body may be form-encoded or JSON. The response is plain text
// prefixed with "CON " (continue) or "END " (terminate).
func (h *Handler) handleUSSD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	req, err := parseUSSDRequest(r)
	if err != nil {
		writeUSSDError(w, fmt.Sprintf("END bad request: %v", err))
		return
	}

	if req.SessionID == "" || req.ServiceCode == "" {
		writeUSSDError(w, "END sessionId and serviceCode are required")
		return
	}

	resp, err := h.svc.HandleUSSD(r.Context(), req)
	if err != nil {
		writeUSSDError(w, "END internal error")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	prefix := "END"
	if resp.Continue {
		prefix = "CON"
	}
	fmt.Fprintf(w, "%s %s", prefix, resp.Message)
}

// parseUSSDRequest extracts USSD fields from either a JSON body or form values.
func parseUSSDRequest(r *http.Request) (service.USSDRequest, error) {
	ct := r.Header.Get("Content-Type")
	var req service.USSDRequest

	switch {
	case ct == "application/json" || ct == "application/json; charset=utf-8":
		var body struct {
			SessionID   string `json:"sessionId"`
			ServiceCode string `json:"serviceCode"`
			PhoneNumber string `json:"phoneNumber"`
			Text        string `json:"text"`
			NetworkCode string `json:"networkCode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return req, fmt.Errorf("decode JSON: %w", err)
		}
		req = service.USSDRequest{
			SessionID:   body.SessionID,
			ServiceCode: body.ServiceCode,
			PhoneNumber: body.PhoneNumber,
			Text:        body.Text,
			NetworkCode: body.NetworkCode,
		}
	default:
		// Assume application/x-www-form-urlencoded (standard telecom gateway format).
		if err := r.ParseForm(); err != nil {
			return req, fmt.Errorf("parse form: %w", err)
		}
		req = service.USSDRequest{
			SessionID:   r.FormValue("sessionId"),
			ServiceCode: r.FormValue("serviceCode"),
			PhoneNumber: r.FormValue("phoneNumber"),
			Text:        r.FormValue("text"),
			NetworkCode: r.FormValue("networkCode"),
		}
	}
	return req, nil
}

// otpSendRequest is the JSON body for POST /v1/sms/otp/send.
type otpSendRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// otpSendResponse is the JSON response for POST /v1/sms/otp/send.
type otpSendResponse struct {
	OTPID string `json:"otp_id"`
	// OTPPlain is included for testing/development. In production the caller
	// delivers it to the SMS gateway and discards it — never logged.
	OTPPlain string `json:"otp_plain,omitempty"`
}

// handleOTPSend handles POST /v1/sms/otp/send.
func (h *Handler) handleOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var body otpSendRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.PhoneNumber == "" {
		writeError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	result, err := h.svc.SendOTP(r.Context(), body.PhoneNumber)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate OTP")
		return
	}

	writeJSON(w, http.StatusCreated, otpSendResponse{
		OTPID:    result.OTPID,
		OTPPlain: result.OTPPlain,
	})
}

// otpVerifyRequest is the JSON body for POST /v1/sms/otp/verify.
type otpVerifyRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
}

// otpVerifyResponse is the JSON response for POST /v1/sms/otp/verify.
type otpVerifyResponse struct {
	Valid bool `json:"valid"`
}

// handleOTPVerify handles POST /v1/sms/otp/verify.
func (h *Handler) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var body otpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.PhoneNumber == "" || body.OTP == "" {
		writeError(w, http.StatusBadRequest, "phone_number and otp are required")
		return
	}

	valid, err := h.svc.VerifyOTP(r.Context(), body.PhoneNumber, body.OTP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "OTP verification failed")
		return
	}

	writeJSON(w, http.StatusOK, otpVerifyResponse{Valid: valid})
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

// writeUSSDError writes a plain-text END error message as a USSD response.
func writeUSSDError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) // USSD gateways always expect HTTP 200.
	fmt.Fprint(w, msg)
}
