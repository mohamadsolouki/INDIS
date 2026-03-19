// Package hsm — vault.go
// HashiCorp Vault Transit engine KeyManager implementation.
//
// All key material remains inside Vault; this client never handles raw private
// key bytes. Signing and decryption are performed server-side by Vault.
//
// Ref: https://www.vaultproject.io/docs/secrets/transit
// Ref: INDIS PRD §4.3 — FIPS 140-2 Level 3 HSM
package hsm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VaultKeyManager implements KeyManager using HashiCorp Vault's Transit
// secrets engine. All cryptographic operations are delegated to Vault; no
// private key material is ever held in process memory.
type VaultKeyManager struct {
	addr      string      // Vault server address, e.g. "http://vault:8200"
	token     string      // Vault token (production: use AppRole or k8s auth)
	mountPath string      // Transit mount path, default: "transit"
	client    *http.Client
}

// NewVaultKeyManager creates a VaultKeyManager that communicates with the
// Vault Transit engine at addr, authenticating with token, using mountPath as
// the secrets engine mount point.
//
// If mountPath is empty it defaults to "transit".
// The embedded http.Client uses a 30-second timeout; callers that need
// different timeout behaviour should replace client via the Client field after
// construction.
func NewVaultKeyManager(addr, token, mountPath string) *VaultKeyManager {
	if mountPath == "" {
		mountPath = "transit"
	}
	return &VaultKeyManager{
		addr:      strings.TrimRight(addr, "/"),
		token:     token,
		mountPath: mountPath,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// vaultKeyTypeName maps INDIS KeyType values to the Vault Transit key type
// names accepted by the API.
func vaultKeyTypeName(kt KeyType) (string, error) {
	switch kt {
	case KeyTypeEd25519:
		return "ed25519", nil
	case KeyTypeECDSAP256:
		return "ecdsa-p256", nil
	case KeyTypeDilithium3:
		// Vault does not yet natively support Dilithium; fall back to ed25519
		// until a Vault plugin provides FIPS 204 support.
		return "ed25519", nil
	case KeyTypeAES256:
		return "aes256-gcm96", nil
	default:
		return "", fmt.Errorf("hsm: vault: unsupported key type %q", kt)
	}
}

// GenerateKey generates a new key in the Vault Transit engine.
// POST /v1/{mount}/keys/{name}
func (v *VaultKeyManager) GenerateKey(ctx context.Context, name string, keyType KeyType) error {
	vType, err := vaultKeyTypeName(keyType)
	if err != nil {
		return err
	}
	body := map[string]any{"type": vType}
	_, err = v.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/%s/keys/%s", v.mountPath, name), body)
	if err != nil {
		return fmt.Errorf("hsm: vault generate key %q: %w", name, err)
	}
	return nil
}

// GetPublicKey retrieves the public key for the named key.
// GET /v1/{mount}/keys/{name}
func (v *VaultKeyManager) GetPublicKey(ctx context.Context, name string) ([]byte, error) {
	resp, err := v.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/v1/%s/keys/%s", v.mountPath, name), nil)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault get public key %q: %w", name, err)
	}

	// Response shape: {"data":{"keys":{"1":{"public_key":"..."}}}}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault get public key %q: unexpected response shape", name)
	}
	keys, ok := data["keys"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault get public key %q: no keys in response", name)
	}
	// Find the latest version (highest numeric string key).
	var pubKeyStr string
	for _, v := range keys {
		vMap, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if pk, ok := vMap["public_key"].(string); ok && pk != "" {
			pubKeyStr = pk
		}
	}
	if pubKeyStr == "" {
		return nil, fmt.Errorf("hsm: vault get public key %q: public key not found in response", name)
	}
	return []byte(pubKeyStr), nil
}

// Sign signs data using the named Vault Transit key.
// POST /v1/{mount}/sign/{name}
func (v *VaultKeyManager) Sign(ctx context.Context, name string, data []byte) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(data)
	body := map[string]any{
		"input":           encoded,
		"prehashed":       false,
		"marshaling_algo": "asn1",
	}
	resp, err := v.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/%s/sign/%s", v.mountPath, name), body)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault sign %q: %w", name, err)
	}

	respData, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault sign %q: unexpected response shape", name)
	}
	sig, ok := respData["signature"].(string)
	if !ok || sig == "" {
		return nil, fmt.Errorf("hsm: vault sign %q: signature missing in response", name)
	}
	return []byte(sig), nil
}

// Verify verifies a signature produced by Sign.
// POST /v1/{mount}/verify/{name}
func (v *VaultKeyManager) Verify(ctx context.Context, name string, data, signature []byte) (bool, error) {
	encoded := base64.StdEncoding.EncodeToString(data)
	body := map[string]any{
		"input":           encoded,
		"signature":       string(signature),
		"marshaling_algo": "asn1",
	}
	resp, err := v.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/%s/verify/%s", v.mountPath, name), body)
	if err != nil {
		return false, fmt.Errorf("hsm: vault verify %q: %w", name, err)
	}

	respData, ok := resp["data"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("hsm: vault verify %q: unexpected response shape", name)
	}
	valid, _ := respData["valid"].(bool)
	return valid, nil
}

// RotateKey rotates the named key in the Vault Transit engine.
// POST /v1/{mount}/keys/{name}/rotate
func (v *VaultKeyManager) RotateKey(ctx context.Context, name string) error {
	_, err := v.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/%s/keys/%s/rotate", v.mountPath, name), nil)
	if err != nil {
		return fmt.Errorf("hsm: vault rotate key %q: %w", name, err)
	}
	return nil
}

// ListKeys returns the names of all keys in the Transit engine mount.
// LIST /v1/{mount}/keys
func (v *VaultKeyManager) ListKeys(ctx context.Context) ([]string, error) {
	resp, err := v.doRequest(ctx, "LIST",
		fmt.Sprintf("/v1/%s/keys", v.mountPath), nil)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault list keys: %w", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault list keys: unexpected response shape")
	}
	keysRaw, ok := data["keys"].([]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault list keys: keys field missing or wrong type")
	}
	names := make([]string, 0, len(keysRaw))
	for _, k := range keysRaw {
		if s, ok := k.(string); ok {
			names = append(names, s)
		}
	}
	return names, nil
}

// EncryptData encrypts plaintext using the named Vault Transit key.
// POST /v1/{mount}/encrypt/{name}
func (v *VaultKeyManager) EncryptData(ctx context.Context, name string, plaintext []byte) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(plaintext)
	body := map[string]any{"plaintext": encoded}
	resp, err := v.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/%s/encrypt/%s", v.mountPath, name), body)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault encrypt %q: %w", name, err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault encrypt %q: unexpected response shape", name)
	}
	ct, ok := data["ciphertext"].(string)
	if !ok || ct == "" {
		return nil, fmt.Errorf("hsm: vault encrypt %q: ciphertext missing in response", name)
	}
	return []byte(ct), nil
}

// DecryptData decrypts ciphertext produced by EncryptData.
// POST /v1/{mount}/decrypt/{name}
func (v *VaultKeyManager) DecryptData(ctx context.Context, name string, ciphertext []byte) ([]byte, error) {
	body := map[string]any{"ciphertext": string(ciphertext)}
	resp, err := v.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/%s/decrypt/%s", v.mountPath, name), body)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault decrypt %q: %w", name, err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hsm: vault decrypt %q: unexpected response shape", name)
	}
	ptEncoded, ok := data["plaintext"].(string)
	if !ok || ptEncoded == "" {
		return nil, fmt.Errorf("hsm: vault decrypt %q: plaintext missing in response", name)
	}
	pt, err := base64.StdEncoding.DecodeString(ptEncoded)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault decrypt %q: base64 decode: %w", name, err)
	}
	return pt, nil
}

// doRequest performs a JSON HTTP request against the Vault API and returns the
// decoded response body as a map. A nil body sends an empty JSON object for
// POST requests.
func (v *VaultKeyManager) doRequest(ctx context.Context, method, path string, body any) (map[string]any, error) {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("hsm: vault request marshal: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, v.addr+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault new request: %w", err)
	}
	req.Header.Set("X-Vault-Token", v.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hsm: vault read response: %w", err)
	}

	// Vault returns 200/204 on success; 204 has no body.
	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hsm: vault %s %s: HTTP %d: %s", method, path, resp.StatusCode, string(raw))
	}

	// Some successful Vault responses (e.g. rotate) may return an empty body.
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("hsm: vault decode response: %w", err)
	}
	if errs, ok := result["errors"]; ok {
		if errList, ok := errs.([]any); ok && len(errList) > 0 {
			return nil, fmt.Errorf("hsm: vault error: %v", errList[0])
		}
	}
	return result, nil
}
