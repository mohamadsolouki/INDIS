package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mohamadsolouki/INDIS/services/ussd/internal/repository"
)

// fakeRepo is an in-memory implementation of USSDRepository.
type fakeRepo struct {
	sessions map[string]*repository.USSDSession
	otps     map[string]*repository.SMSOtp // keyed by phone hash
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		sessions: make(map[string]*repository.USSDSession),
		otps:     make(map[string]*repository.SMSOtp),
	}
}

func (f *fakeRepo) CreateSession(_ context.Context, s repository.USSDSession) error {
	cp := s
	f.sessions[s.SessionID] = &cp
	return nil
}

func (f *fakeRepo) GetSession(_ context.Context, sessionID string) (*repository.USSDSession, error) {
	s, ok := f.sessions[sessionID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeRepo) UpdateSession(_ context.Context, s repository.USSDSession) error {
	existing, ok := f.sessions[s.SessionID]
	if !ok {
		return repository.ErrNotFound
	}
	existing.CurrentStep = s.CurrentStep
	existing.Locale = s.Locale
	existing.StateData = s.StateData
	existing.LastActiveAt = s.LastActiveAt
	return nil
}

func (f *fakeRepo) EndSession(_ context.Context, sessionID string) error {
	s, ok := f.sessions[sessionID]
	if !ok {
		return repository.ErrNotFound
	}
	now := time.Now().UTC()
	s.EndedAt = &now
	s.StateData = map[string]string{}
	return nil
}

func (f *fakeRepo) CreateOTP(_ context.Context, otp repository.SMSOtp) error {
	cp := otp
	f.otps[otp.PhoneNumberHash] = &cp
	return nil
}

func (f *fakeRepo) GetActiveOTP(_ context.Context, phoneHash string) (*repository.SMSOtp, error) {
	otp, ok := f.otps[phoneHash]
	if !ok || otp.Used || otp.ExpiresAt.Before(time.Now()) {
		return nil, repository.ErrNotFound
	}
	cp := *otp
	return &cp, nil
}

func (f *fakeRepo) MarkOTPUsed(_ context.Context, id string) error {
	for _, otp := range f.otps {
		if otp.ID == id {
			otp.Used = true
			return nil
		}
	}
	return repository.ErrNotFound
}

// Tests

func TestHandleUSSD_NewSession_VoterFlow(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-voter-1",
		ServiceCode: "*ID#",
		PhoneNumber: "+989001234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Continue {
		t.Error("expected Continue=true for new voter session")
	}
	lower := strings.ToLower(resp.Message)
	if !strings.Contains(lower, "id") && !strings.Contains(resp.Message, "کد") {
		t.Errorf("expected message to contain ID prompt, got: %s", resp.Message)
	}
}

func TestHandleUSSD_NewSession_PensionFlow(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-pension-1",
		ServiceCode: "*PENSION#",
		PhoneNumber: "+989001234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Continue {
		t.Error("expected Continue=true for new pension session")
	}
}

func TestHandleUSSD_NewSession_CredentialFlow(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-cred-1",
		ServiceCode: "*CRED#",
		PhoneNumber: "+989001234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Continue {
		t.Error("expected Continue=true for new credential session")
	}
}

func TestHandleUSSD_NewSession_UnknownCode(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-unk-1",
		ServiceCode: "*UNKNOWN#",
		PhoneNumber: "+989001234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Continue {
		t.Error("expected Continue=false for unknown service code")
	}
}

func TestHandleUSSD_VoterFlow_Step1_EnterID(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "")
	// Create session (step 0)
	_, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-voter-step1",
		ServiceCode: "*ID#",
		PhoneNumber: "+989001234567",
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// Advance: user enters their national ID
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID: "sess-voter-step1",
		Text:      "1234567890",
	})
	if err != nil {
		t.Fatalf("step 1 failed: %v", err)
	}
	if !resp.Continue {
		t.Error("expected Continue=true at step 1 (PIN prompt)")
	}
	// Should ask for PIN
	lower := strings.ToLower(resp.Message)
	if !strings.Contains(lower, "pin") && !strings.Contains(resp.Message, "رمز") {
		t.Errorf("expected PIN prompt, got: %s", resp.Message)
	}
}

func TestHandleUSSD_VoterFlow_Step2_Complete(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "")

	// Step 0: create session
	if _, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-voter-step2",
		ServiceCode: "*ID#",
		PhoneNumber: "+989001234567",
	}); err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// Step 1: enter national ID
	if _, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID: "sess-voter-step2",
		Text:      "1234567890",
	}); err != nil {
		t.Fatalf("step 1 failed: %v", err)
	}

	// Step 2: enter PIN — session should end
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID: "sess-voter-step2",
		Text:      "1234",
	})
	if err != nil {
		t.Fatalf("step 2 failed: %v", err)
	}
	if resp.Continue {
		t.Error("expected Continue=false after PIN entry (voter flow complete)")
	}
}

func TestHandleUSSD_SessionExpired(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "")

	// Manually insert an expired session.
	old := time.Now().UTC().Add(-10 * time.Minute) // past TTL of 5 minutes
	repo.sessions["sess-expired"] = &repository.USSDSession{
		SessionID:       "sess-expired",
		PhoneNumberHash: hashString("+989001234567"),
		ServiceCode:     "*ID#",
		CurrentStep:     0,
		FlowType:        "voter",
		Locale:          "fa",
		StateData:       map[string]string{},
		StartedAt:       old,
		LastActiveAt:    old,
	}

	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID: "sess-expired",
		Text:      "input",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Continue {
		t.Error("expected Continue=false for expired session")
	}
	if !strings.Contains(resp.Message, "expired") && !strings.Contains(resp.Message, "منقضی") {
		t.Errorf("expected 'expired' in message, got: %s", resp.Message)
	}
}

func TestHandleUSSD_ExistingSession_Ended(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := New(repo, "")

	// Create a session and end it.
	now := time.Now().UTC()
	ended := now.Add(-1 * time.Minute)
	repo.sessions["sess-ended"] = &repository.USSDSession{
		SessionID:    "sess-ended",
		FlowType:     "voter",
		Locale:       "fa",
		StateData:    map[string]string{},
		LastActiveAt: ended,
		EndedAt:      &ended,
	}

	// Sending a new request on the same session ID should start fresh.
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-ended",
		ServiceCode: "*ID#",
		PhoneNumber: "+989001234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// createSession is called, returns Continue=true greeting
	if !resp.Continue {
		t.Error("expected Continue=true for fresh session after ended session")
	}
}

func TestSendOTP_Success(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	res, err := svc.SendOTP(context.Background(), "+989001234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OTPID == "" {
		t.Error("expected OTPID to be set")
	}
	if len(res.OTPPlain) != 6 {
		t.Errorf("expected 6-digit OTP, got length %d: %s", len(res.OTPPlain), res.OTPPlain)
	}
	for _, c := range res.OTPPlain {
		if c < '0' || c > '9' {
			t.Errorf("OTP contains non-digit character: %c", c)
		}
	}
}

func TestVerifyOTP_CorrectOTP(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	phone := "+989001234567"
	res, err := svc.SendOTP(context.Background(), phone)
	if err != nil {
		t.Fatalf("send OTP failed: %v", err)
	}

	ok, err := svc.VerifyOTP(context.Background(), phone, res.OTPPlain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected VerifyOTP to return true for correct OTP")
	}
}

func TestVerifyOTP_WrongOTP(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	phone := "+989001234567"
	if _, err := svc.SendOTP(context.Background(), phone); err != nil {
		t.Fatalf("send OTP failed: %v", err)
	}

	ok, err := svc.VerifyOTP(context.Background(), phone, "000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected VerifyOTP to return false for wrong OTP")
	}
}

func TestVerifyOTP_NotFound(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	ok, err := svc.VerifyOTP(context.Background(), "+989001111111", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected VerifyOTP to return false for unknown phone")
	}
}

func TestLocalization_Persian(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	// Empty NetworkCode → defaults to "fa" (Persian)
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-fa",
		ServiceCode: "*ID#",
		PhoneNumber: "+989001234567",
		NetworkCode: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Persian welcome message should contain Persian characters.
	if !strings.Contains(resp.Message, "خوش") && !strings.Contains(resp.Message, "هویت") {
		t.Errorf("expected Persian text in message, got: %s", resp.Message)
	}
}

func TestLocalization_Kurdish(t *testing.T) {
	t.Parallel()
	svc := New(newFakeRepo(), "")
	resp, err := svc.HandleUSSD(context.Background(), USSDRequest{
		SessionID:   "sess-ku",
		ServiceCode: "*ID#",
		PhoneNumber: "+989001234567",
		NetworkCode: "IRKU",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Kurdish welcome message: "Xêr hatî bo INDIS"
	if !strings.Contains(resp.Message, "INDIS") && !strings.Contains(resp.Message, "Xêr") {
		t.Errorf("expected Kurdish text in message, got: %s", resp.Message)
	}
}
