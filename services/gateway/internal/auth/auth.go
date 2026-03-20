// Package auth implements JWT and API-key authentication middleware for the INDIS API gateway.
//
// JWT format: HS256 signed with claims:
//
//	{"sub": "did:indis:...", "ministry": "interior", "role": "operator", "exp": 1234567890}
//
// API keys: SHA-256 hash of the key stored in an in-memory map loaded from env var
// API_KEYS (comma-separated list of "keyID:sha256hex" pairs). The raw key is passed
// via the X-API-Key header; the gateway hashes it and compares against the stored digest.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Context key type to avoid collisions with other packages.
type ctxKey string

const (
	// CtxKeyDID is the context key for the authenticated subject DID.
	CtxKeyDID ctxKey = "auth.did"
	// CtxKeyRole is the context key for the operator role.
	CtxKeyRole ctxKey = "auth.role"
	// CtxKeyMinistry is the context key for the originating ministry.
	CtxKeyMinistry ctxKey = "auth.ministry"
)

// publicRoute describes a URL prefix + optional method pair that skips auth.
type publicRoute struct {
	prefix string
	method string // empty means all methods
}

// publicRoutes lists routes that do not require authentication.
var publicRoutes = []publicRoute{
	{prefix: "/health", method: ""},
	{prefix: "/v1/identity/", method: http.MethodGet},
	{prefix: "/v1/credential/", method: http.MethodGet},
	{prefix: "/v1/electoral/verify", method: ""},
	{prefix: "/v1/ussd", method: ""},
}

// isPublic reports whether the request is for a public (unauthenticated) route.
func isPublic(r *http.Request) bool {
	for _, route := range publicRoutes {
		if strings.HasPrefix(r.URL.Path, route.prefix) {
			if route.method == "" || route.method == r.Method {
				return true
			}
		}
	}
	return false
}

// jwtClaims holds the subset of JWT claims used by the gateway.
type jwtClaims struct {
	Sub      string `json:"sub"`
	Ministry string `json:"ministry"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
	JTI      string `json:"jti"`
}

// verifyJWT parses and verifies an HS256 JWT using only standard library crypto.
// It returns the validated claims or a non-nil error.
func verifyJWT(tokenStr, secret string) (*jwtClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errInvalid("malformed token: expected 3 parts")
	}

	// Verify signature: HMAC-SHA256(base64url(header) + "." + base64url(payload), secret).
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errInvalid("malformed token: bad signature encoding")
	}
	if !hmac.Equal(expectedSig, gotSig) {
		return nil, errInvalid("invalid token signature")
	}

	// Decode payload.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errInvalid("malformed token: bad payload encoding")
	}
	var claims jwtClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, errInvalid("malformed token: cannot parse claims")
	}

	// Verify expiry.
	if claims.Exp == 0 {
		return nil, errInvalid("token missing exp claim")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errInvalid("token expired")
	}
	if claims.Sub == "" {
		return nil, errInvalid("token missing sub claim")
	}

	return &claims, nil
}

// verifyAPIKey checks the provided raw API key against the stored SHA-256 hashes.
// apiKeys maps keyID → sha256hex of the raw key. The X-API-Key header value is
// compared against every stored hash (constant-time). Returns the keyID or error.
func verifyAPIKey(rawKey string, apiKeys map[string]string) (string, error) {
	if rawKey == "" {
		return "", errInvalid("empty API key")
	}
	sum := sha256.Sum256([]byte(rawKey))
	gotHex := hex.EncodeToString(sum[:])

	for keyID, storedHex := range apiKeys {
		if hmac.Equal([]byte(gotHex), []byte(storedHex)) {
			return keyID, nil
		}
	}
	return "", errInvalid("unknown API key")
}

// authError is a sentinel type for authentication failures.
type authError string

func (e authError) Error() string { return string(e) }

func errInvalid(msg string) error { return authError(msg) }

// Middleware returns an HTTP middleware that authenticates requests via JWT Bearer
// token or X-API-Key header. On success the DID, role, and ministry are stored in
// the request context. Public routes bypass authentication entirely.
//
// nc may be nil, in which case jti replay protection is disabled (useful in tests
// that construct the middleware without a full NonceCache).
func Middleware(jwtSecret string, apiKeys map[string]string, nc *NonceCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublic(r) {
				next.ServeHTTP(w, r)
				return
			}

			var (
				did      string
				role     string
				ministry string
			)

			// Attempt Bearer JWT first.
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := verifyJWT(token, jwtSecret)
				if err != nil {
					writeUnauthorized(w, err.Error())
					return
				}
				// jti replay protection — only enforced when the token carries a jti
				// claim AND a NonceCache is wired in. Tokens without jti are allowed
				// for backward compatibility.
				if nc != nil && claims.JTI != "" {
					exp := time.Unix(claims.Exp, 0)
					if !nc.Check(claims.JTI, exp) {
						writeUnauthorized(w, "token replay detected")
						return
					}
				}
				did = claims.Sub
				role = claims.Role
				ministry = claims.Ministry
			} else if rawKey := r.Header.Get("X-API-Key"); rawKey != "" {
				// Fall back to API key.
				keyID, err := verifyAPIKey(rawKey, apiKeys)
				if err != nil {
					writeUnauthorized(w, err.Error())
					return
				}
				// API keys are service-to-service; use keyID as the DID, role=service.
				did = "apikey:" + keyID
				role = "service"
				ministry = ""
			} else {
				writeUnauthorized(w, "authentication required: provide Authorization Bearer or X-API-Key header")
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyDID, did)
			ctx = context.WithValue(ctx, CtxKeyRole, role)
			ctx = context.WithValue(ctx, CtxKeyMinistry, ministry)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DIDFromContext returns the authenticated DID stored in ctx, or empty string.
func DIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyDID).(string)
	return v
}

// RoleFromContext returns the role stored in ctx, or empty string.
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyRole).(string)
	return v
}

// MinistryFromContext returns the ministry stored in ctx, or empty string.
func MinistryFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyMinistry).(string)
	return v
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="INDIS"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + jsonEscape(msg) + `"}`))
}

// jsonEscape escapes a string for safe embedding in a JSON string literal.
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	// json.Marshal wraps in quotes; strip them.
	return string(b[1 : len(b)-1])
}

// ParseAPIKeysEnv parses the API_KEYS environment variable value into a map.
// The format is a comma-separated list of "keyID:sha256hex" pairs.
func ParseAPIKeysEnv(raw string) map[string]string {
	out := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, ":")
		if idx <= 0 || idx == len(pair)-1 {
			continue // malformed entry; skip
		}
		keyID := pair[:idx]
		hash := pair[idx+1:]
		out[keyID] = hash
	}
	return out
}
