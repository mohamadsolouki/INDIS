// Package service implements business logic for the USSD/SMS gateway service.
//
// It manages USSD session state machines and SMS OTP lifecycle. All personally
// identifiable data (phone numbers, national ID fragments) is stored only as
// SHA-256 hashes. Session state_data is wiped on session end per FR-015.6.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/IranProsperityProject/INDIS/services/ussd/internal/repository"
)

// sessionTTL is the maximum idle lifetime of a USSD session.
const sessionTTL = 5 * time.Minute

// otpTTL is the lifetime of a generated SMS OTP.
const otpTTL = 5 * time.Minute

// messages is a two-level map: locale → message key → translated text.
// Locales: "fa" (Farsi), "en" (English), "ku" (Kurdish), "ar" (Arabic), "az" (Azerbaijani).
var messages = map[string]map[string]string{
	"fa": {
		"welcome":          "به سیستم هویت ملی خوش آمدید",
		"enter_id":         "کد ملی خود را وارد کنید",
		"enter_pin":        "رمز عبور خود را وارد کنید",
		"voter_yes":        "واجد شرایط رای‌دهی هستید",
		"voter_no":         "واجد شرایط رای‌دهی نیستید",
		"pension_active":   "مستمری شما فعال است",
		"pension_inactive": "مستمری شما فعال نیست",
		"cred_valid":       "گواهینامه معتبر است",
		"cred_revoked":     "گواهینامه باطل شده است",
		"cred_expired":     "گواهینامه منقضی شده است",
		"enter_cred_id":    "شناسه گواهینامه را وارد کنید",
		"session_expired":  "جلسه منقضی شده است",
		"invalid_input":    "ورودی نامعتبر است",
		"menu_voter":       "۱. بررسی صلاحیت رای‌دهی",
		"menu_pension":     "۲. بررسی وضعیت مستمری",
		"menu_cred":        "۳. بررسی وضعیت گواهینامه",
		"select_option":    "گزینه مورد نظر را انتخاب کنید",
	},
	"en": {
		"welcome":          "Welcome to INDIS",
		"enter_id":         "Enter your national ID",
		"enter_pin":        "Enter your PIN",
		"voter_yes":        "You are eligible to vote",
		"voter_no":         "You are not eligible to vote",
		"pension_active":   "Your pension is active",
		"pension_inactive": "Your pension is inactive",
		"cred_valid":       "Credential is VALID",
		"cred_revoked":     "Credential is REVOKED",
		"cred_expired":     "Credential is EXPIRED",
		"enter_cred_id":    "Enter your credential ID",
		"session_expired":  "Session has expired",
		"invalid_input":    "Invalid input",
		"menu_voter":       "1. Voter eligibility check",
		"menu_pension":     "2. Pension status check",
		"menu_cred":        "3. Credential status check",
		"select_option":    "Select an option",
	},
	"ku": {
		"welcome":          "Xêr hatî bo INDIS",
		"enter_id":         "Koda neteweyî ya xwe binivîse",
		"enter_pin":        "Şîfreya xwe binivîse",
		"voter_yes":        "Tu mafê dengdanê dî",
		"voter_no":         "Tu mafê dengdanê nînî",
		"pension_active":   "Teqawidiya te çalak e",
		"pension_inactive": "Teqawidiya te ne çalak e",
		"cred_valid":       "Belgename derbasdar e",
		"cred_revoked":     "Belgename betal bûye",
		"cred_expired":     "Belgename derbas bûye",
		"enter_cred_id":    "Nasnameyê belgenama xwe binivîse",
		"session_expired":  "Danişîn qediya",
		"invalid_input":    "Têketina nerast",
		"menu_voter":       "1. Kontrola mafê dengdanê",
		"menu_pension":     "2. Kontrola rewşa teqawidiyê",
		"menu_cred":        "3. Kontrola rewşa belgenameyê",
		"select_option":    "Bijarteyekê hilbijêre",
	},
	"ar": {
		"welcome":          "مرحبا بك في INDIS",
		"enter_id":         "أدخل رقمك الوطني",
		"enter_pin":        "أدخل رمز PIN",
		"voter_yes":        "أنت مؤهل للتصويت",
		"voter_no":         "أنت غير مؤهل للتصويت",
		"pension_active":   "معاشك التقاعدي نشط",
		"pension_inactive": "معاشك التقاعدي غير نشط",
		"cred_valid":       "الوثيقة صالحة",
		"cred_revoked":     "تم إلغاء الوثيقة",
		"cred_expired":     "انتهت صلاحية الوثيقة",
		"enter_cred_id":    "أدخل معرّف الوثيقة",
		"session_expired":  "انتهت الجلسة",
		"invalid_input":    "مدخل غير صالح",
		"menu_voter":       "1. التحقق من أهلية التصويت",
		"menu_pension":     "2. التحقق من حالة المعاش",
		"menu_cred":        "3. التحقق من حالة الوثيقة",
		"select_option":    "اختر خياراً",
	},
	"az": {
		"welcome":          "INDIS-ə xoş gəldiniz",
		"enter_id":         "Şəxsiyyət kodunuzu daxil edin",
		"enter_pin":        "PIN kodunuzu daxil edin",
		"voter_yes":        "Seçki hüququnuz var",
		"voter_no":         "Seçki hüququnuz yoxdur",
		"pension_active":   "Pensiyanız aktivdir",
		"pension_inactive": "Pensiyanız aktiv deyil",
		"cred_valid":       "Sertifikat etibarlıdır",
		"cred_revoked":     "Sertifikat ləğv edilib",
		"cred_expired":     "Sertifikatın müddəti bitib",
		"enter_cred_id":    "Sertifikat ID-ni daxil edin",
		"session_expired":  "Sessiya başa çatıb",
		"invalid_input":    "Yanlış giriş",
		"menu_voter":       "1. Seçki hüququnun yoxlanması",
		"menu_pension":     "2. Pensiya vəziyyətinin yoxlanması",
		"menu_cred":        "3. Sertifikat vəziyyətinin yoxlanması",
		"select_option":    "Bir seçim edin",
	},
}

// USSDRequest contains the fields posted by a telecom USSD gateway.
type USSDRequest struct {
	SessionID   string
	ServiceCode string
	PhoneNumber string
	Text        string
	NetworkCode string
}

// USSDResponse is the plain-text response returned to the USSD gateway.
type USSDResponse struct {
	// Continue is true when more user input is expected (CON prefix).
	// False means the session is over (END prefix).
	Continue bool
	Message  string
}

// OTPSendResult is returned after generating an OTP.
type OTPSendResult struct {
	OTPID string
	// OTPPlain is the cleartext OTP that the caller must deliver via SMS.
	// In production this is handed to an SMS dispatcher; it is never stored.
	OTPPlain string
}

// ErrSessionExpired is returned when a session has timed out.
var ErrSessionExpired = errors.New("service: session expired")

// Service implements USSD session management and SMS OTP operations.
type Service struct {
	repo       *repository.Repository
	gatewayURL string
}

// New creates a Service.
func New(repo *repository.Repository, gatewayURL string) *Service {
	return &Service{repo: repo, gatewayURL: gatewayURL}
}

// msg looks up a localized string, falling back to English when the key or locale is missing.
func msg(locale, key string) string {
	if lm, ok := messages[locale]; ok {
		if v, ok := lm[key]; ok {
			return v
		}
	}
	if v, ok := messages["en"][key]; ok {
		return v
	}
	return key
}

// hashString returns the hex-encoded SHA-256 digest of s.
func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// generateID produces a random URL-safe identifier with the given prefix.
func generateID(prefix string) (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// detectLocale infers locale from the network code.
// In production this would be a more sophisticated lookup table.
func detectLocale(networkCode string) string {
	switch networkCode {
	case "IRKU":
		return "ku"
	case "IRAZ":
		return "az"
	case "IRAR":
		return "ar"
	default:
		return "fa"
	}
}

// detectFlowType maps service codes to flow types.
func detectFlowType(serviceCode string) (string, bool) {
	switch serviceCode {
	case "*ID#", "*463#":
		return "voter", true
	case "*PENSION#", "*736746#":
		return "pension", true
	case "*CRED#", "*2733#":
		return "credential", true
	}
	return "", false
}

// HandleUSSD processes an incoming USSD request from a telecom gateway.
// It creates a new session on first contact (empty Text) or advances the state
// machine for existing sessions.
func (s *Service) HandleUSSD(ctx context.Context, req USSDRequest) (*USSDResponse, error) {
	// Try to load an existing session.
	sess, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("service: get session: %w", err)
	}

	if errors.Is(err, repository.ErrNotFound) || sess == nil {
		// New session — create it.
		return s.createSession(ctx, req)
	}

	// Guard: if already ended, treat as new.
	if sess.EndedAt != nil {
		return s.createSession(ctx, req)
	}

	// Guard: session TTL check.
	if time.Since(sess.LastActiveAt) > sessionTTL {
		// End the expired session and return END message.
		_ = s.repo.EndSession(ctx, sess.SessionID)
		return &USSDResponse{
			Continue: false,
			Message:  msg(sess.Locale, "session_expired"),
		}, nil
	}

	return s.advanceSession(ctx, sess, req.Text)
}

// createSession starts a new USSD session for the given request.
func (s *Service) createSession(ctx context.Context, req USSDRequest) (*USSDResponse, error) {
	flowType, ok := detectFlowType(req.ServiceCode)
	if !ok {
		return &USSDResponse{Continue: false, Message: "Unknown service code"}, nil
	}

	locale := detectLocale(req.NetworkCode)
	id, err := generateID("us_")
	if err != nil {
		return nil, fmt.Errorf("service: generate session id: %w", err)
	}
	// Use provided session ID if non-empty (telecom gateway dictates it).
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = id
	}

	now := time.Now().UTC()
	sess := repository.USSDSession{
		SessionID:       sessionID,
		PhoneNumberHash: hashString(req.PhoneNumber),
		ServiceCode:     req.ServiceCode,
		CurrentStep:     0,
		FlowType:        flowType,
		Locale:          locale,
		StateData:       map[string]string{},
		StartedAt:       now,
		LastActiveAt:    now,
	}
	if err = s.repo.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("service: create session: %w", err)
	}

	return s.stepZeroGreeting(locale, flowType), nil
}

// stepZeroGreeting returns the initial menu message for a flow.
func (s *Service) stepZeroGreeting(locale, flowType string) *USSDResponse {
	greeting := msg(locale, "welcome") + "\n"
	switch flowType {
	case "voter":
		greeting += msg(locale, "enter_id")
	case "pension":
		greeting += msg(locale, "enter_id")
	case "credential":
		greeting += msg(locale, "enter_cred_id")
	}
	return &USSDResponse{Continue: true, Message: greeting}
}

// advanceSession advances the state machine for an existing session.
func (s *Service) advanceSession(ctx context.Context, sess *repository.USSDSession, input string) (*USSDResponse, error) {
	switch sess.FlowType {
	case "voter":
		return s.stepVoter(ctx, sess, input)
	case "pension":
		return s.stepPension(ctx, sess, input)
	case "credential":
		return s.stepCredential(ctx, sess, input)
	}
	return &USSDResponse{Continue: false, Message: msg(sess.Locale, "invalid_input")}, nil
}

// stepVoter drives the voter eligibility flow.
//
//	Step 0: greeting (handled in createSession)
//	Step 1: user enters national ID fragment → store hash, ask for PIN
//	Step 2: user enters PIN → verify, return result, END
func (s *Service) stepVoter(ctx context.Context, sess *repository.USSDSession, input string) (*USSDResponse, error) {
	switch sess.CurrentStep {
	case 0:
		// Record the national ID hash, advance to step 1.
		sess.StateData["id_hash"] = hashString(input)
		sess.CurrentStep = 1
		sess.LastActiveAt = time.Now().UTC()
		if err := s.repo.UpdateSession(ctx, *sess); err != nil {
			return nil, fmt.Errorf("service: voter step 0 update: %w", err)
		}
		return &USSDResponse{Continue: true, Message: msg(sess.Locale, "enter_pin")}, nil

	case 1:
		// Verify PIN via gateway (simplified: call gateway credential check).
		// Store PIN hash only for tracing; never the plaintext PIN.
		sess.StateData["pin_hash"] = hashString(input)

		eligible := s.verifyVoterEligibility(ctx, sess.StateData["id_hash"], hashString(input))

		// End session and wipe PII.
		if err := s.repo.EndSession(ctx, sess.SessionID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("service: voter end session: %w", err)
		}

		resultKey := "voter_no"
		if eligible {
			resultKey = "voter_yes"
		}
		return &USSDResponse{Continue: false, Message: msg(sess.Locale, resultKey)}, nil
	}

	return &USSDResponse{Continue: false, Message: msg(sess.Locale, "invalid_input")}, nil
}

// stepPension drives the pension eligibility flow.
//
//	Step 0: user enters national ID fragment → store hash, return result, END
func (s *Service) stepPension(ctx context.Context, sess *repository.USSDSession, input string) (*USSDResponse, error) {
	switch sess.CurrentStep {
	case 0:
		idHash := hashString(input)
		active := s.verifyPensionStatus(ctx, idHash)

		// End session and wipe PII.
		if err := s.repo.EndSession(ctx, sess.SessionID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("service: pension end session: %w", err)
		}

		resultKey := "pension_inactive"
		if active {
			resultKey = "pension_active"
		}
		return &USSDResponse{Continue: false, Message: msg(sess.Locale, resultKey)}, nil
	}

	return &USSDResponse{Continue: false, Message: msg(sess.Locale, "invalid_input")}, nil
}

// stepCredential drives the credential status check flow.
//
//	Step 0: user enters credential ID fragment → store hash, return status, END
func (s *Service) stepCredential(ctx context.Context, sess *repository.USSDSession, input string) (*USSDResponse, error) {
	switch sess.CurrentStep {
	case 0:
		credHash := hashString(input)
		status := s.checkCredentialStatus(ctx, credHash)

		// End session and wipe PII.
		if err := s.repo.EndSession(ctx, sess.SessionID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("service: credential end session: %w", err)
		}

		var msgKey string
		switch status {
		case "valid":
			msgKey = "cred_valid"
		case "revoked":
			msgKey = "cred_revoked"
		default:
			msgKey = "cred_expired"
		}
		return &USSDResponse{Continue: false, Message: msg(sess.Locale, msgKey)}, nil
	}

	return &USSDResponse{Continue: false, Message: msg(sess.Locale, "invalid_input")}, nil
}

// verifyVoterEligibility calls the gateway to verify voter eligibility.
// Returns true when the holder is eligible. In this scaffold it always returns
// false; production code performs an HTTP call to the gateway credential endpoint.
func (s *Service) verifyVoterEligibility(_ context.Context, _ string, _ string) bool {
	// TODO(production): POST to s.gatewayURL/v1/electoral/verify with ZK proof.
	// The gateway verifier returns only a boolean claim (ZK-first design).
	return false
}

// verifyPensionStatus calls the gateway to check pension payment status.
// Returns true when an active pension record exists.
func (s *Service) verifyPensionStatus(_ context.Context, _ string) bool {
	// TODO(production): POST to s.gatewayURL/v1/credential/{hash} for pension credential.
	return false
}

// checkCredentialStatus calls the gateway to determine if a credential is
// valid, revoked, or expired.
func (s *Service) checkCredentialStatus(_ context.Context, _ string) string {
	// TODO(production): GET s.gatewayURL/v1/credential/{hash} → parse revocation response.
	return "expired"
}

// SendOTP generates a 6-digit OTP, hashes it, persists the hash, and returns
// the plain OTP for delivery via SMS. The plain OTP is never stored.
func (s *Service) SendOTP(ctx context.Context, phoneNumber string) (*OTPSendResult, error) {
	otp, err := generateNumericOTP(6)
	if err != nil {
		return nil, fmt.Errorf("service: generate otp: %w", err)
	}

	id, err := generateID("otp_")
	if err != nil {
		return nil, fmt.Errorf("service: generate otp id: %w", err)
	}

	now := time.Now().UTC()
	rec := repository.SMSOtp{
		ID:              id,
		PhoneNumberHash: hashString(phoneNumber),
		OTPHash:         hashString(otp),
		ExpiresAt:       now.Add(otpTTL),
		Used:            false,
		CreatedAt:       now,
	}
	if err = s.repo.CreateOTP(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: create otp: %w", err)
	}

	return &OTPSendResult{OTPID: id, OTPPlain: otp}, nil
}

// VerifyOTP checks the provided OTP against the stored hash for a phone number.
// Returns true and marks the OTP used when it matches and is still valid.
func (s *Service) VerifyOTP(ctx context.Context, phoneNumber, otpPlain string) (bool, error) {
	phoneHash := hashString(phoneNumber)
	rec, err := s.repo.GetActiveOTP(ctx, phoneHash)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("service: get active otp: %w", err)
	}

	if rec.OTPHash != hashString(otpPlain) {
		return false, nil
	}

	if err = s.repo.MarkOTPUsed(ctx, rec.ID); err != nil {
		return false, fmt.Errorf("service: mark otp used: %w", err)
	}
	return true, nil
}

// generateNumericOTP creates a cryptographically random decimal string of length n.
func generateNumericOTP(n int) (string, error) {
	digits := make([]byte, n)
	for i := range digits {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0') + byte(v.Int64())
	}
	return string(digits), nil
}
