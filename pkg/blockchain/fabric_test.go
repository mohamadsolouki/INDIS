// Package blockchain — unit tests for the Fabric gateway adapter.
//
// All tests use httptest.NewServer to spin up a local HTTP server that mimics
// the Fabric peer gateway REST API. No real Fabric network is required.
package blockchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fabricTestServer creates a test HTTP server that records requests and returns
// configurable responses. The caller can inspect capturedPath and capturedBody
// after each call to verify the adapter sent the correct request.
type fabricTestServer struct {
	server       *httptest.Server
	handler      http.HandlerFunc
}

// newFabricTestAdapter builds a FabricAdapter wired to a test HTTP server.
// The provided handler controls what responses are returned.
func newFabricTestAdapter(t *testing.T, handler http.HandlerFunc) (*FabricAdapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	adapter, err := NewFabricAdapter(FabricConfig{
		GatewayURL: srv.URL,
		ChannelID:  "test-channel",
		MSPId:      "TestMSP",
	})
	if err != nil {
		t.Fatalf("NewFabricAdapter: %v", err)
	}
	return adapter, srv
}

// submitResponse builds a complete gateway JSON response body for a successful submit.
// The result field contains a JSON object string (TxID + blockHeight) as returned by
// the real Fabric gateway.
func submitResponse(txID string) string {
	inner := fmt.Sprintf(`{"txId":"%s","blockHeight":42}`, txID)
	b, _ := json.Marshal(inner) // properly escape the inner JSON as a string value
	return fmt.Sprintf(`{"result":%s}`, string(b))
}

// evaluateResponse wraps a result value in the gateway JSON response envelope.
// The result string is JSON-marshalled so that it is correctly embedded as a
// JSON string value inside the outer envelope object.
func evaluateResponse(result string) string {
	b, _ := json.Marshal(result)
	return fmt.Sprintf(`{"result":%s}`, string(b))
}

// errorResponse returns a gateway error response.
func errorResponse(msg string) string {
	return fmt.Sprintf(`{"error":"%s"}`, msg)
}

// ---- NewFabricAdapter tests -------------------------------------------------

func TestNewFabricAdapter_DefaultChannels(t *testing.T) {
	adapter, err := NewFabricAdapter(FabricConfig{GatewayURL: "http://localhost:7080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.didChannel != "did-registry-channel" {
		t.Errorf("didChannel = %q, want %q", adapter.didChannel, "did-registry-channel")
	}
	if adapter.credChannel != "credential-anchor-channel" {
		t.Errorf("credChannel = %q, want %q", adapter.credChannel, "credential-anchor-channel")
	}
	if adapter.auditChannel != "audit-log-channel" {
		t.Errorf("auditChannel = %q, want %q", adapter.auditChannel, "audit-log-channel")
	}
	if adapter.electoralChannel != "electoral-channel" {
		t.Errorf("electoralChannel = %q, want %q", adapter.electoralChannel, "electoral-channel")
	}
}

func TestNewFabricAdapter_InvalidCertPair(t *testing.T) {
	_, err := NewFabricAdapter(FabricConfig{
		GatewayURL: "http://localhost:7080",
		CertPEM:    "not-a-cert",
		KeyPEM:     "not-a-key",
	})
	if err == nil {
		t.Error("expected error for invalid cert/key pair, got nil")
	}
}

func TestNewFabricAdapter_InvalidCACert(t *testing.T) {
	_, err := NewFabricAdapter(FabricConfig{
		GatewayURL:   "http://localhost:7080",
		TLSCACertPEM: "garbage-pem",
	})
	if err == nil {
		t.Error("expected error for unparseable CA cert PEM, got nil")
	}
}

// ---- RegisterDID tests -----------------------------------------------------

func TestRegisterDID_Success(t *testing.T) {
	var capturedPath string
	var capturedArgs []string

	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		var req gatewayRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedArgs = req.Args
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, submitResponse("tx-001"))
	})

	doc := DIDDocument{
		ID: "did:indis:test123",
		PublicKeys: []PublicKey{
			{ID: "did:indis:test123#key-1", Type: "Ed25519VerificationKey2020", Controller: "did:indis:test123", PublicKeyHex: "abcdef"},
		},
	}
	receipt, err := adapter.RegisterDID(context.Background(), "did:indis:test123", doc)
	if err != nil {
		t.Fatalf("RegisterDID: %v", err)
	}
	if receipt == nil {
		t.Fatal("expected non-nil receipt")
	}
	if !strings.Contains(capturedPath, "submit") {
		t.Errorf("expected submit path, got %q", capturedPath)
	}
	if !strings.Contains(capturedPath, "did-registry-channel") {
		t.Errorf("expected did-registry-channel in path, got %q", capturedPath)
	}
	if !strings.Contains(capturedPath, "RegisterDID") {
		t.Errorf("expected RegisterDID in path, got %q", capturedPath)
	}
	if len(capturedArgs) != 1 {
		t.Errorf("expected 1 argument, got %d", len(capturedArgs))
	}
}

func TestRegisterDID_GatewayError(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, errorResponse("DID already registered"))
	})

	_, err := adapter.RegisterDID(context.Background(), "did:indis:dup", DIDDocument{ID: "did:indis:dup"})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "DID already registered") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---- ResolveDID tests -------------------------------------------------------

func TestResolveDID_Success(t *testing.T) {
	chainDoc := chainDIDDoc{
		DID: "did:indis:abc",
		PublicKeys: []chainPubKey{
			{ID: "did:indis:abc#key-1", Type: "Ed25519VerificationKey2020", Controller: "did:indis:abc", PublicKeyHex: "deadbeef"},
		},
		Created: "2026-01-01T00:00:00Z",
		Updated: "2026-01-02T00:00:00Z",
	}
	chainDocJSON, _ := json.Marshal(chainDoc)

	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, evaluateResponse(string(chainDocJSON)))
	})

	doc, err := adapter.ResolveDID(context.Background(), "did:indis:abc")
	if err != nil {
		t.Fatalf("ResolveDID: %v", err)
	}
	if doc.ID != "did:indis:abc" {
		t.Errorf("doc.ID = %q, want %q", doc.ID, "did:indis:abc")
	}
	if len(doc.PublicKeys) != 1 {
		t.Errorf("len(PublicKeys) = %d, want 1", len(doc.PublicKeys))
	}
	if doc.PublicKeys[0].PublicKeyHex != "deadbeef" {
		t.Errorf("PublicKeyHex = %q, want %q", doc.PublicKeys[0].PublicKeyHex, "deadbeef")
	}
}

func TestResolveDID_NotFound(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, errorResponse("DID did:indis:nope not found"))
	})

	_, err := adapter.ResolveDID(context.Background(), "did:indis:nope")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---- DeactivateDID tests ---------------------------------------------------

func TestDeactivateDID_Success(t *testing.T) {
	var capturedPath string
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, submitResponse("tx-deact"))
	})

	receipt, err := adapter.DeactivateDID(context.Background(), "did:indis:old")
	if err != nil {
		t.Fatalf("DeactivateDID: %v", err)
	}
	if receipt == nil {
		t.Fatal("expected non-nil receipt")
	}
	if !strings.Contains(capturedPath, "DeactivateDID") {
		t.Errorf("expected DeactivateDID in path, got %q", capturedPath)
	}
}

// ---- AnchorCredential tests ------------------------------------------------

func TestAnchorCredential_Success(t *testing.T) {
	var capturedArgs []string
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		var req gatewayRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedArgs = req.Args
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, submitResponse("tx-anchor"))
	})

	var h Hash
	copy(h[:], []byte("test-hash-32-bytes-long-padding!!"))
	receipt, err := adapter.AnchorCredential(context.Background(), h, "did:indis:issuer1")
	if err != nil {
		t.Fatalf("AnchorCredential: %v", err)
	}
	if receipt == nil {
		t.Fatal("expected non-nil receipt")
	}
	if len(capturedArgs) != 2 {
		t.Fatalf("expected 2 args, got %d", len(capturedArgs))
	}
	// First arg must be hex encoding of the hash.
	expectedHex := hex.EncodeToString(h[:])
	if capturedArgs[0] != expectedHex {
		t.Errorf("arg[0] = %q, want %q", capturedArgs[0], expectedHex)
	}
	if capturedArgs[1] != "did:indis:issuer1" {
		t.Errorf("arg[1] = %q, want %q", capturedArgs[1], "did:indis:issuer1")
	}
}

// ---- VerifyAnchor tests ----------------------------------------------------

func TestVerifyAnchor_Exists(t *testing.T) {
	anchorResp := `{"exists":true,"issuerDid":"did:indis:issuer1","blockTime":"2026-03-01T10:00:00Z"}`
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, evaluateResponse(anchorResp))
	})

	var h Hash
	status, err := adapter.VerifyAnchor(context.Background(), h)
	if err != nil {
		t.Fatalf("VerifyAnchor: %v", err)
	}
	if !status.Exists {
		t.Error("expected Exists=true")
	}
	if status.IssuerDID != "did:indis:issuer1" {
		t.Errorf("IssuerDID = %q, want %q", status.IssuerDID, "did:indis:issuer1")
	}
	if status.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

func TestVerifyAnchor_NotFound(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, evaluateResponse(`{"exists":false}`))
	})

	var h Hash
	status, err := adapter.VerifyAnchor(context.Background(), h)
	if err != nil {
		t.Fatalf("VerifyAnchor: %v", err)
	}
	if status.Exists {
		t.Error("expected Exists=false")
	}
}

// ---- RevokeCredential tests ------------------------------------------------

func TestRevokeCredential_Success(t *testing.T) {
	var capturedArgs []string
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		var req gatewayRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedArgs = req.Args
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, submitResponse("tx-revoke"))
	})

	receipt, err := adapter.RevokeCredential(context.Background(), "cred-001", RevocationReasonCompromised)
	if err != nil {
		t.Fatalf("RevokeCredential: %v", err)
	}
	if receipt == nil {
		t.Fatal("expected non-nil receipt")
	}
	if len(capturedArgs) != 2 {
		t.Fatalf("expected 2 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0] != "cred-001" {
		t.Errorf("arg[0] = %q, want %q", capturedArgs[0], "cred-001")
	}
	if capturedArgs[1] != string(RevocationReasonCompromised) {
		t.Errorf("arg[1] = %q, want %q", capturedArgs[1], RevocationReasonCompromised)
	}
}

// ---- CheckRevocationStatus tests -------------------------------------------

func TestCheckRevocationStatus_NotRevoked(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, evaluateResponse(`{"revoked":false}`))
	})

	status, err := adapter.CheckRevocationStatus(context.Background(), "cred-999")
	if err != nil {
		t.Fatalf("CheckRevocationStatus: %v", err)
	}
	if status.Revoked {
		t.Error("expected Revoked=false")
	}
}

func TestCheckRevocationStatus_Revoked(t *testing.T) {
	revokedResp := `{"revoked":true,"reason":"compromised","timestamp":"2026-02-15T12:00:00Z"}`
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, evaluateResponse(revokedResp))
	})

	status, err := adapter.CheckRevocationStatus(context.Background(), "cred-bad")
	if err != nil {
		t.Fatalf("CheckRevocationStatus: %v", err)
	}
	if !status.Revoked {
		t.Error("expected Revoked=true")
	}
	if status.Reason != RevocationReasonCompromised {
		t.Errorf("Reason = %q, want %q", status.Reason, RevocationReasonCompromised)
	}
	if status.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

// ---- GetRevocationList tests -----------------------------------------------

func TestGetRevocationList_Empty(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, evaluateResponse(`[]`))
	})

	list, err := adapter.GetRevocationList(context.Background(), "did:indis:issuer1")
	if err != nil {
		t.Fatalf("GetRevocationList: %v", err)
	}
	if list.IssuerDID != "did:indis:issuer1" {
		t.Errorf("IssuerDID = %q, want %q", list.IssuerDID, "did:indis:issuer1")
	}
	if len(list.RevokedIDs) != 0 {
		t.Errorf("expected empty list, got %v", list.RevokedIDs)
	}
}

func TestGetRevocationList_WithEntries(t *testing.T) {
	listResp := `[{"credentialId":"cred-1","reason":"expired","timestamp":"2026-01-01T00:00:00Z"},{"credentialId":"cred-2","reason":"compromised","timestamp":"2026-01-02T00:00:00Z"}]`
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, evaluateResponse(listResp))
	})

	list, err := adapter.GetRevocationList(context.Background(), "did:indis:issuer1")
	if err != nil {
		t.Fatalf("GetRevocationList: %v", err)
	}
	if len(list.RevokedIDs) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list.RevokedIDs))
	}
	if list.RevokedIDs[0] != "cred-1" {
		t.Errorf("RevokedIDs[0] = %q, want %q", list.RevokedIDs[0], "cred-1")
	}
}

// ---- LogVerificationEvent tests --------------------------------------------

func TestLogVerificationEvent_Success(t *testing.T) {
	var capturedArgs []string
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		var req gatewayRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedArgs = req.Args
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, submitResponse("tx-audit"))
	})

	event := AnonymizedVerificationEvent{
		EventID:          "evt-001",
		CredentialType:   "AgeCredential",
		VerifierCategory: "pharmacy",
		Result:           true,
		Timestamp:        time.Now(),
	}
	receipt, err := adapter.LogVerificationEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("LogVerificationEvent: %v", err)
	}
	if receipt == nil {
		t.Fatal("expected non-nil receipt")
	}
	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(capturedArgs))
	}

	// Verify the event JSON contains no personal data fields.
	evtJSON := capturedArgs[0]
	forbidden := []string{"name", "national_id", "address", "phone", "email"}
	for _, f := range forbidden {
		if strings.Contains(evtJSON, `"`+f+`"`) {
			t.Errorf("event JSON contains forbidden field %q: %s", f, evtJSON)
		}
	}
}

func TestLogVerificationEvent_PathUsesAuditChannel(t *testing.T) {
	var capturedPath string
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, submitResponse("tx-a"))
	})

	event := AnonymizedVerificationEvent{
		EventID:        "evt-002",
		CredentialType: "CitizenshipCredential",
		Timestamp:      time.Now(),
	}
	adapter.LogVerificationEvent(context.Background(), event)

	if !strings.Contains(capturedPath, "audit-log-channel") {
		t.Errorf("expected audit-log-channel in path, got %q", capturedPath)
	}
}

// ---- EstimateTxTime tests --------------------------------------------------

func TestEstimateTxTime(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {})
	d, err := adapter.EstimateTxTime(context.Background())
	if err != nil {
		t.Fatalf("EstimateTxTime: %v", err)
	}
	if d != 500*time.Millisecond {
		t.Errorf("EstimateTxTime = %v, want 500ms", d)
	}
}

// ---- GetBlockHeight / GetValidatorStatus tests ----------------------------

func TestGetBlockHeight_FromHealthEndpoint(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "healthz") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"blockHeight": 1234, "status": "OK"}`)
		}
	})

	height, err := adapter.GetBlockHeight(context.Background())
	if err != nil {
		t.Fatalf("GetBlockHeight: %v", err)
	}
	if height != 1234 {
		t.Errorf("blockHeight = %d, want 1234", height)
	}
}

func TestGetValidatorStatus_Success(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"OK","peer":{"id":"peer0.org1","address":"peer0.org1:7051"}}`)
	})

	statuses, err := adapter.GetValidatorStatus(context.Background())
	if err != nil {
		t.Fatalf("GetValidatorStatus: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if !statuses[0].IsActive {
		t.Error("expected IsActive=true")
	}
	if statuses[0].NodeID != "peer0.org1" {
		t.Errorf("NodeID = %q, want %q", statuses[0].NodeID, "peer0.org1")
	}
}

func TestGetValidatorStatus_GatewayUnreachable(t *testing.T) {
	adapter, err := NewFabricAdapter(FabricConfig{
		GatewayURL: "http://127.0.0.1:1", // nothing listening here
	})
	if err != nil {
		t.Fatalf("NewFabricAdapter: %v", err)
	}
	adapter.client.Timeout = 100 * time.Millisecond

	_, err = adapter.GetValidatorStatus(context.Background())
	if err == nil {
		t.Error("expected error for unreachable gateway, got nil")
	}
}

// ---- Context cancellation test ---------------------------------------------

func TestRegisterDID_ContextCancelled(t *testing.T) {
	adapter, _ := newFabricTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		// Slow response — context should cancel before this completes.
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, `{"result":"ok"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := adapter.RegisterDID(ctx, "did:indis:slow", DIDDocument{ID: "did:indis:slow"})
	if err == nil {
		t.Error("expected context deadline error, got nil")
	}
}

// ---- makeTxReceipt tests ---------------------------------------------------

func TestMakeTxReceipt_WellFormed(t *testing.T) {
	raw := `{"txId":"abc123","blockHeight":99}`
	r := makeTxReceipt(raw)
	if r.TxID != "abc123" {
		t.Errorf("TxID = %q, want %q", r.TxID, "abc123")
	}
	if r.BlockHeight != 99 {
		t.Errorf("BlockHeight = %d, want 99", r.BlockHeight)
	}
}

func TestMakeTxReceipt_Fallback(t *testing.T) {
	r := makeTxReceipt("plain-tx-string")
	if r.TxID != "plain-tx-string" {
		t.Errorf("TxID = %q, want %q", r.TxID, "plain-tx-string")
	}
}

func TestMakeTxReceipt_Empty(t *testing.T) {
	r := makeTxReceipt("")
	if r.TxID == "" {
		t.Error("expected non-empty synthetic TxID for empty input")
	}
}

// ---- Interface compliance test ---------------------------------------------

func TestFabricAdapter_ImplementsInterface(t *testing.T) {
	adapter, err := NewFabricAdapter(FabricConfig{GatewayURL: "http://localhost:7080"})
	if err != nil {
		t.Fatalf("NewFabricAdapter: %v", err)
	}
	var _ BlockchainAdapter = adapter // compile-time check surfaced at test time
}
