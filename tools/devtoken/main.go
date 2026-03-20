// Command devtoken generates a signed HS256 JWT for local INDIS development testing.
//
// Usage:
//
//	go run tools/devtoken/main.go [flags]
//
// Flags:
//
//	--did      Subject DID (default: did:indis:dev-test-001)
//	--role     Token role: citizen | ministry | admin | verifier (default: citizen)
//	--ministry Ministry identifier, empty unless role=ministry (default: "")
//	--secret   HMAC-SHA256 secret (default: indis-dev-secret-change-in-prod)
//	--expiry   Token lifetime as Go duration string (default: 24h)
//
// The tool uses only Go standard library packages (crypto/hmac, crypto/sha256,
// encoding/base64, encoding/json) — no external JWT libraries are required.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// jwtHeader is the fixed base64url-encoded JWT header for HS256.
const jwtHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

// b64url returns the base64url-encoded (no padding) form of data.
func b64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// buildJWT constructs and signs a minimal HS256 JWT.
// Payload fields: sub, role, ministry, exp.
func buildJWT(did, role, ministry, secret string, exp int64) (string, error) {
	// Build payload map; omit ministry if empty.
	payload := map[string]interface{}{
		"sub":  did,
		"role": role,
		"exp":  exp,
	}
	if ministry != "" {
		payload["ministry"] = ministry
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Signing input: base64url(header) + "." + base64url(payload)
	signingInput := jwtHeader + "." + b64url(payloadJSON)

	// Compute HMAC-SHA256 signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)

	return signingInput + "." + b64url(sig), nil
}

func main() {
	did := flag.String("did", "did:indis:dev-test-001", "Subject DID")
	role := flag.String("role", "citizen", "Token role: citizen | ministry | admin | verifier")
	ministry := flag.String("ministry", "", "Ministry identifier (used with role=ministry)")
	secret := flag.String("secret", "indis-dev-secret-change-in-prod", "HMAC-SHA256 signing secret")
	expiry := flag.String("expiry", "24h", "Token lifetime as a Go duration string (e.g. 24h, 1h, 30m)")
	flag.Parse()

	// Validate role.
	validRoles := map[string]bool{
		"citizen":  true,
		"ministry": true,
		"admin":    true,
		"verifier": true,
	}
	if !validRoles[strings.ToLower(*role)] {
		fmt.Fprintf(os.Stderr, "error: --role must be one of citizen|ministry|admin|verifier, got %q\n", *role)
		os.Exit(1)
	}

	// Parse expiry duration.
	dur, err := time.ParseDuration(*expiry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid --expiry %q: %v\n", *expiry, err)
		os.Exit(1)
	}
	exp := time.Now().Add(dur).Unix()

	token, err := buildJWT(*did, strings.ToLower(*role), *ministry, *secret, exp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
}
