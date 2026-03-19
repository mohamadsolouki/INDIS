// Package cors implements CORS (Cross-Origin Resource Sharing) middleware for the
// INDIS API gateway.
//
// In development the gateway is configured with CORS_ALLOWED_ORIGINS="*" so that
// the React PWA and mobile dev proxies can reach it without a separate CORS proxy.
// In production the env var should list explicit origins, e.g.:
//
//	CORS_ALLOWED_ORIGINS=https://id.iran.gov,https://mobile.iran.gov
package cors

import (
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that adds CORS headers.
// allowedOrigins is a comma-separated list of permitted origins; use "*" for all.
// Preflight OPTIONS requests are responded to immediately (no backend call).
func Middleware(allowedOrigins string) func(http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)
	allowAll := len(origins) == 1 && origins[0] == "*"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Non-browser request: skip CORS processing.
				next.ServeHTTP(w, r)
				return
			}

			// Determine allowed origin header value.
			var allowOrigin string
			if allowAll {
				allowOrigin = "*"
			} else {
				for _, o := range origins {
					if strings.EqualFold(o, origin) {
						allowOrigin = origin
						break
					}
				}
			}

			if allowOrigin == "" {
				// Origin not permitted; continue without CORS headers so the browser
				// will block the request.
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// When the origin is a specific domain (not "*") the response varies by
			// origin, so it must not be served from a shared cache.
			if allowOrigin != "*" {
				w.Header().Set("Vary", "Origin")
			}

			// Respond to preflight immediately.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseOrigins splits a comma-separated origins string into a trimmed slice.
func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
