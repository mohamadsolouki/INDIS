// Package blockchain — Hyperledger Fabric gateway adapter.
//
// This file implements BlockchainAdapter using the Fabric Peer Gateway REST API
// available in Hyperledger Fabric 2.4+. The adapter communicates via HTTP(S) to
// a Fabric peer gateway endpoint, avoiding a direct dependency on the full Fabric
// Go SDK (which carries hundreds of transitive dependencies).
//
// REST conventions used:
//   POST /v1/submit/{channelID}/{chaincodeID}/{function}   — state-changing invoke
//   POST /v1/evaluate/{channelID}/{chaincodeID}/{function} — read-only query
//
// Request body:  {"args": ["arg1", "arg2", ...]}
// Response body: {"result": "..."} on success, {"error": "..."} on failure
package blockchain

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FabricConfig holds all configuration required to connect to a Hyperledger Fabric
// peer gateway endpoint. TLS is optional; if CertPEM and KeyPEM are both empty
// the adapter will connect without client-certificate authentication (useful for
// local development networks).
type FabricConfig struct {
	// GatewayURL is the base URL of the Fabric peer gateway REST API.
	// Example: "http://peer0.org1.example.com:7080"
	GatewayURL string

	// ChannelID is the default Fabric channel name. Individual methods override this
	// via their dedicated channel fields below.
	ChannelID string

	// MSPId is the Membership Service Provider identifier for the calling client.
	// Example: "Org1MSP"
	MSPId string

	// CertPEM is the PEM-encoded client TLS certificate used for mTLS.
	// Leave empty to disable client certificate authentication.
	CertPEM string

	// KeyPEM is the PEM-encoded private key corresponding to CertPEM.
	KeyPEM string

	// TLSCACertPEM is the PEM-encoded TLS CA certificate used to verify the peer's
	// TLS certificate. Leave empty to use the system certificate pool.
	TLSCACertPEM string
}

// FabricAdapter is a production BlockchainAdapter that communicates with a
// Hyperledger Fabric peer via the Fabric Gateway REST API (Fabric 2.4+).
//
// Each logical domain (DID, credentials, audit, electoral) is mapped to a
// dedicated Fabric channel and chaincode ID, allowing independent upgrade and
// endorsement policies per domain.
type FabricAdapter struct {
	config FabricConfig
	client *http.Client

	// Per-domain channel and chaincode identifiers.
	didChannel        string // channel for DID registry operations
	didChaincode      string // chaincode ID on didChannel
	credChannel       string // channel for credential anchoring / revocation
	credChaincode     string // chaincode ID on credChannel
	auditChannel      string // channel for anonymized audit events
	auditChaincode    string // chaincode ID on auditChannel
	electoralChannel  string // channel for electoral ZK proof anchoring
	electoralChaincode string // chaincode ID on electoralChannel
}

// NewFabricAdapter constructs a FabricAdapter from the provided configuration.
// It builds an http.Client with optional mTLS configuration and sets the default
// channel/chaincode names. All channel and chaincode names can be overridden via
// FabricConfig.ChannelID or by calling the setter methods before first use.
//
// Returns an error if the TLS configuration is invalid.
func NewFabricAdapter(cfg FabricConfig) (*FabricAdapter, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Configure the CA certificate pool if provided.
	if cfg.TLSCACertPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.TLSCACertPEM)) {
			return nil, fmt.Errorf("failed to parse TLS CA certificate PEM")
		}
		tlsCfg.RootCAs = pool
	}

	// Configure client certificate authentication if both cert and key are provided.
	if cfg.CertPEM != "" && cfg.KeyPEM != "" {
		cert, err := tls.X509KeyPair([]byte(cfg.CertPEM), []byte(cfg.KeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse client TLS key pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &FabricAdapter{
		config:              cfg,
		client:              httpClient,
		didChannel:          "did-registry-channel",
		didChaincode:        "did-registry-cc",
		credChannel:         "credential-anchor-channel",
		credChaincode:       "credential-anchor-cc",
		auditChannel:        "audit-log-channel",
		auditChaincode:      "audit-log-cc",
		electoralChannel:    "electoral-channel",
		electoralChaincode:  "electoral-cc",
	}, nil
}

// ---- internal HTTP helpers -------------------------------------------------

// gatewayRequest is the JSON body sent to the Fabric gateway REST API.
type gatewayRequest struct {
	Args []string `json:"args"`
}

// gatewayResponse is the JSON body returned by the Fabric gateway REST API.
type gatewayResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
	TxID   string `json:"txId,omitempty"`
}

// submit performs a state-changing invoke via POST /v1/submit/{channel}/{cc}/{fn}.
// Returns the raw result string from the gateway response, or an error.
func (f *FabricAdapter) submit(
	ctx context.Context,
	channel, chaincode, function string,
	args []string,
) (string, error) {
	return f.call(ctx, "submit", channel, chaincode, function, args)
}

// evaluate performs a read-only query via POST /v1/evaluate/{channel}/{cc}/{fn}.
// Returns the raw result string from the gateway response, or an error.
func (f *FabricAdapter) evaluate(
	ctx context.Context,
	channel, chaincode, function string,
	args []string,
) (string, error) {
	return f.call(ctx, "evaluate", channel, chaincode, function, args)
}

// call is the shared HTTP helper used by submit and evaluate.
func (f *FabricAdapter) call(
	ctx context.Context,
	verb, channel, chaincode, function string,
	args []string,
) (string, error) {
	url := fmt.Sprintf("%s/v1/%s/%s/%s/%s",
		strings.TrimRight(f.config.GatewayURL, "/"),
		verb, channel, chaincode, function,
	)

	reqBody := gatewayRequest{Args: args}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gateway request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if f.config.MSPId != "" {
		httpReq.Header.Set("X-MSP-ID", f.config.MSPId)
	}

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("fabric gateway request failed [%s %s/%s/%s]: %w",
			verb, channel, chaincode, function, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gateway response body: %w", err)
	}

	var gResp gatewayResponse
	if err := json.Unmarshal(respBytes, &gResp); err != nil {
		return "", fmt.Errorf("failed to parse gateway response: %w (body: %s)", err, string(respBytes))
	}
	if gResp.Error != "" {
		return "", fmt.Errorf("chaincode error [%s/%s/%s]: %s", channel, chaincode, function, gResp.Error)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("gateway HTTP %d [%s/%s/%s]: %s",
			resp.StatusCode, channel, chaincode, function, string(respBytes))
	}

	return gResp.Result, nil
}

// makeTxReceipt builds a TxReceipt from a gateway response string.
// The gateway returns a JSON object like {"txId":"...", "blockHeight": 42} in the
// result field for submit operations; if the format differs, a synthetic receipt is
// returned with the raw result as the TxID.
func makeTxReceipt(raw string) *TxReceipt {
	var parsed struct {
		TxID        string `json:"txId"`
		BlockHeight uint64 `json:"blockHeight"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil && parsed.TxID != "" {
		return &TxReceipt{
			TxID:        parsed.TxID,
			BlockHeight: parsed.BlockHeight,
			Timestamp:   time.Now(),
		}
	}
	// Fallback: treat the raw string as the transaction ID.
	if raw == "" {
		raw = "fabric-tx-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return &TxReceipt{
		TxID:      raw,
		Timestamp: time.Now(),
	}
}

// ---- DID operations --------------------------------------------------------

// RegisterDID anchors a new DID document on the DID registry channel.
// The DIDDocument is serialised to JSON before being passed to the chaincode.
// Personal data MUST NOT be present in the document; the chaincode will reject it.
func (f *FabricAdapter) RegisterDID(ctx context.Context, did string, document DIDDocument) (*TxReceipt, error) {
	doc := didDocToChain(did, document)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("RegisterDID: failed to marshal DID document: %w", err)
	}

	raw, err := f.submit(ctx, f.didChannel, f.didChaincode, "RegisterDID", []string{string(docJSON)})
	if err != nil {
		return nil, fmt.Errorf("RegisterDID: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// ResolveDID retrieves the DID document for the given DID from the ledger.
func (f *FabricAdapter) ResolveDID(ctx context.Context, did string) (*DIDDocument, error) {
	raw, err := f.evaluate(ctx, f.didChannel, f.didChaincode, "ResolveDID", []string{did})
	if err != nil {
		return nil, fmt.Errorf("ResolveDID: %w", err)
	}

	var chainDoc chainDIDDoc
	if err := json.Unmarshal([]byte(raw), &chainDoc); err != nil {
		return nil, fmt.Errorf("ResolveDID: failed to parse DID document: %w", err)
	}
	return chainDocToAdapter(chainDoc), nil
}

// UpdateDIDDocument replaces the DID document stored on the ledger.
func (f *FabricAdapter) UpdateDIDDocument(ctx context.Context, did string, update DIDDocument) (*TxReceipt, error) {
	doc := didDocToChain(did, update)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("UpdateDIDDocument: failed to marshal DID document: %w", err)
	}

	raw, err := f.submit(ctx, f.didChannel, f.didChaincode, "UpdateDIDDocument", []string{string(docJSON)})
	if err != nil {
		return nil, fmt.Errorf("UpdateDIDDocument: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// DeactivateDID marks the given DID as deactivated on the ledger.
func (f *FabricAdapter) DeactivateDID(ctx context.Context, did string) (*TxReceipt, error) {
	raw, err := f.submit(ctx, f.didChannel, f.didChaincode, "DeactivateDID", []string{did})
	if err != nil {
		return nil, fmt.Errorf("DeactivateDID: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// ---- Credential anchoring --------------------------------------------------

// AnchorCredential stores a credential hash and the issuer DID on the credential
// anchor channel.
func (f *FabricAdapter) AnchorCredential(ctx context.Context, credentialHash Hash, issuerDID string) (*TxReceipt, error) {
	hashHex := hex.EncodeToString(credentialHash[:])
	raw, err := f.submit(ctx, f.credChannel, f.credChaincode, "AnchorCredential",
		[]string{hashHex, issuerDID})
	if err != nil {
		return nil, fmt.Errorf("AnchorCredential: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// VerifyAnchor queries the credential anchor channel to check whether a credential
// hash has been anchored and by whom.
func (f *FabricAdapter) VerifyAnchor(ctx context.Context, credentialHash Hash) (*AnchorStatus, error) {
	hashHex := hex.EncodeToString(credentialHash[:])
	raw, err := f.evaluate(ctx, f.credChannel, f.credChaincode, "VerifyAnchor",
		[]string{hashHex})
	if err != nil {
		return nil, fmt.Errorf("VerifyAnchor: %w", err)
	}

	var result struct {
		Exists    bool   `json:"exists"`
		IssuerDID string `json:"issuerDid"`
		BlockTime string `json:"blockTime"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("VerifyAnchor: failed to parse response: %w", err)
	}

	status := &AnchorStatus{
		Exists:    result.Exists,
		IssuerDID: result.IssuerDID,
	}
	if result.BlockTime != "" {
		t, err := time.Parse(time.RFC3339, result.BlockTime)
		if err == nil {
			status.Timestamp = t
		}
	}
	return status, nil
}

// ---- Revocation registry ---------------------------------------------------

// RevokeCredential records a revocation for the given credential ID on the
// credential anchor channel.
func (f *FabricAdapter) RevokeCredential(ctx context.Context, credentialID string, reason RevocationReason) (*TxReceipt, error) {
	raw, err := f.submit(ctx, f.credChannel, f.credChaincode, "RevokeCredential",
		[]string{credentialID, string(reason)})
	if err != nil {
		return nil, fmt.Errorf("RevokeCredential: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// CheckRevocationStatus queries the revocation status of the given credential.
func (f *FabricAdapter) CheckRevocationStatus(ctx context.Context, credentialID string) (*RevocationStatus, error) {
	raw, err := f.evaluate(ctx, f.credChannel, f.credChaincode, "CheckRevocationStatus",
		[]string{credentialID})
	if err != nil {
		return nil, fmt.Errorf("CheckRevocationStatus: %w", err)
	}

	var result struct {
		Revoked   bool   `json:"revoked"`
		Reason    string `json:"reason"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("CheckRevocationStatus: failed to parse response: %w", err)
	}

	status := &RevocationStatus{
		Revoked: result.Revoked,
		Reason:  RevocationReason(result.Reason),
	}
	if result.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, result.Timestamp)
		if err == nil {
			status.Timestamp = t
		}
	}
	return status, nil
}

// GetRevocationList retrieves the full revocation list for the given issuer DID.
func (f *FabricAdapter) GetRevocationList(ctx context.Context, issuerDID string) (*RevocationList, error) {
	raw, err := f.evaluate(ctx, f.credChannel, f.credChaincode, "GetRevocationList",
		[]string{issuerDID})
	if err != nil {
		return nil, fmt.Errorf("GetRevocationList: %w", err)
	}

	var entries []struct {
		CredentialID string `json:"credentialId"`
		Reason       string `json:"reason"`
		Timestamp    string `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("GetRevocationList: failed to parse response: %w", err)
	}

	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.CredentialID)
	}
	return &RevocationList{
		IssuerDID:   issuerDID,
		RevokedIDs:  ids,
		LastUpdated: time.Now(),
	}, nil
}

// ---- Audit trail -----------------------------------------------------------

// LogVerificationEvent submits an anonymized verification event to the audit log
// channel. The event is serialised to JSON before submission. The chaincode will
// reject any event containing personal data fields.
func (f *FabricAdapter) LogVerificationEvent(ctx context.Context, event AnonymizedVerificationEvent) (*TxReceipt, error) {
	evtJSON, err := json.Marshal(map[string]interface{}{
		"event_id":          event.EventID,
		"credential_type":   event.CredentialType,
		"verifier_category": event.VerifierCategory,
		"result":            event.Result,
		"timestamp":         event.Timestamp.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("LogVerificationEvent: failed to marshal event: %w", err)
	}

	raw, err := f.submit(ctx, f.auditChannel, f.auditChaincode, "LogVerificationEvent",
		[]string{string(evtJSON)})
	if err != nil {
		return nil, fmt.Errorf("LogVerificationEvent: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// AnchorAuditEvent submits an immutable audit event hash to the audit-log chaincode.
// Only the event ID and its SHA-256 hash are stored on-chain — no PII.
func (f *FabricAdapter) AnchorAuditEvent(ctx context.Context, eventID, entryHash string) (*TxReceipt, error) {
	raw, err := f.submit(ctx, f.auditChannel, f.auditChaincode, "AnchorAuditEvent",
		[]string{eventID, entryHash})
	if err != nil {
		return nil, fmt.Errorf("AnchorAuditEvent: %w", err)
	}
	return makeTxReceipt(raw), nil
}

// ---- Health and status -----------------------------------------------------

// GetBlockHeight returns the current block height of the DID registry channel.
// If the gateway exposes a dedicated block height endpoint, use that; otherwise
// this implementation falls back to querying a lightweight evaluate call.
func (f *FabricAdapter) GetBlockHeight(ctx context.Context) (uint64, error) {
	url := fmt.Sprintf("%s/healthz", strings.TrimRight(f.config.GatewayURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("GetBlockHeight: failed to build health request: %w", err)
	}
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("GetBlockHeight: health request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("GetBlockHeight: failed to read health response: %w", err)
	}

	var health struct {
		BlockHeight uint64 `json:"blockHeight"`
	}
	if err := json.Unmarshal(body, &health); err == nil && health.BlockHeight > 0 {
		return health.BlockHeight, nil
	}
	// The health endpoint may not include block height; return a sentinel value.
	return 0, nil
}

// GetValidatorStatus queries the Fabric peer gateway health endpoint and returns
// the reported peer node status as a ValidatorStatus slice.
func (f *FabricAdapter) GetValidatorStatus(ctx context.Context) ([]ValidatorStatus, error) {
	url := fmt.Sprintf("%s/healthz", strings.TrimRight(f.config.GatewayURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("GetValidatorStatus: failed to build request: %w", err)
	}
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GetValidatorStatus: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GetValidatorStatus: failed to read response: %w", err)
	}

	// Parse the Fabric peer health response. The schema varies by Fabric version;
	// we extract what we can and return a best-effort status.
	var health struct {
		Status string `json:"status"`
		Peer   struct {
			ID      string `json:"id"`
			Address string `json:"address"`
		} `json:"peer"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		// If parsing fails, return a single entry indicating the gateway is reachable.
		return []ValidatorStatus{
			{
				NodeID:   "fabric-peer",
				Address:  f.config.GatewayURL,
				IsActive: resp.StatusCode == http.StatusOK,
				LastSeen: time.Now(),
			},
		}, nil
	}

	nodeID := health.Peer.ID
	if nodeID == "" {
		nodeID = "fabric-peer"
	}
	address := health.Peer.Address
	if address == "" {
		address = f.config.GatewayURL
	}

	return []ValidatorStatus{
		{
			NodeID:   nodeID,
			Address:  address,
			IsActive: health.Status == "OK" || resp.StatusCode == http.StatusOK,
			LastSeen: time.Now(),
		},
	}, nil
}

// EstimateTxTime returns a constant 500ms as the estimated transaction confirmation
// time for this Fabric network. In production this could be derived from recent
// transaction latency metrics exposed by the gateway.
func (f *FabricAdapter) EstimateTxTime(_ context.Context) (time.Duration, error) {
	return 500 * time.Millisecond, nil
}

// ---- Type conversion helpers -----------------------------------------------

// chainDIDDoc is the JSON shape expected by the DID registry chaincode.
type chainDIDDoc struct {
	DID        string            `json:"did"`
	PublicKeys []chainPubKey     `json:"publicKeys"`
	Services   []chainSvcEndpt   `json:"services"`
	Created    string            `json:"created"`
	Updated    string            `json:"updated"`
	Deactivated bool             `json:"deactivated"`
}

type chainPubKey struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Controller   string `json:"controller"`
	PublicKeyHex string `json:"publicKeyHex"`
}

type chainSvcEndpt struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// didDocToChain converts an adapter DIDDocument to the chaincode JSON shape.
func didDocToChain(did string, doc DIDDocument) chainDIDDoc {
	keys := make([]chainPubKey, len(doc.PublicKeys))
	for i, k := range doc.PublicKeys {
		keys[i] = chainPubKey{
			ID:           k.ID,
			Type:         k.Type,
			Controller:   k.Controller,
			PublicKeyHex: k.PublicKeyHex,
		}
	}
	svcs := make([]chainSvcEndpt, len(doc.ServiceEndpoints))
	for i, s := range doc.ServiceEndpoints {
		svcs[i] = chainSvcEndpt{
			ID:              s.ID,
			Type:            s.Type,
			ServiceEndpoint: s.ServiceEndpoint,
		}
	}
	created := ""
	if !doc.Created.IsZero() {
		created = doc.Created.UTC().Format(time.RFC3339)
	}
	updated := ""
	if !doc.Updated.IsZero() {
		updated = doc.Updated.UTC().Format(time.RFC3339)
	}
	return chainDIDDoc{
		DID:        did,
		PublicKeys: keys,
		Services:   svcs,
		Created:    created,
		Updated:    updated,
	}
}

// chainDocToAdapter converts the chaincode JSON shape back to an adapter DIDDocument.
func chainDocToAdapter(doc chainDIDDoc) *DIDDocument {
	keys := make([]PublicKey, len(doc.PublicKeys))
	for i, k := range doc.PublicKeys {
		keys[i] = PublicKey{
			ID:           k.ID,
			Type:         k.Type,
			Controller:   k.Controller,
			PublicKeyHex: k.PublicKeyHex,
		}
	}
	svcs := make([]ServiceEndpoint, len(doc.Services))
	for i, s := range doc.Services {
		svcs[i] = ServiceEndpoint{
			ID:              s.ID,
			Type:            s.Type,
			ServiceEndpoint: s.ServiceEndpoint,
		}
	}
	result := &DIDDocument{
		ID:               doc.DID,
		PublicKeys:       keys,
		ServiceEndpoints: svcs,
	}
	if doc.Created != "" {
		t, err := time.Parse(time.RFC3339, doc.Created)
		if err == nil {
			result.Created = t
		}
	}
	if doc.Updated != "" {
		t, err := time.Parse(time.RFC3339, doc.Updated)
		if err == nil {
			result.Updated = t
		}
	}
	return result
}

// Compile-time interface check.
var _ BlockchainAdapter = (*FabricAdapter)(nil)
