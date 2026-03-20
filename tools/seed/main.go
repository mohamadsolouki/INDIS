// Command seed populates a running INDIS gateway with test data for frontend development.
//
// It performs the following steps against http://localhost:8080 (or GATEWAY_URL env var):
//  1. Generates a dev admin JWT using the same HS256 logic as tools/devtoken.
//  2. Registers 3 test citizen DIDs via POST /v1/identity/register.
//  3. Initiates enrollment for each registered DID via POST /v1/enrollment/initiate.
//  4. Creates a test election (open status) via POST /v1/electoral/elections.
//  5. Registers a test verifier organisation via POST /v1/verifier/register.
//  6. Generates a test card via POST /v1/card/generate.
//
// All steps are best-effort: failures are logged and the tool continues.
// A summary of all created resource IDs is printed at the end.
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// ── JWT helpers (identical to tools/devtoken, no shared dep) ─────────────────

const jwtHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

func b64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// buildJWT returns a signed HS256 JWT with the given claims.
func buildJWT(subject, role, secret string, expSeconds int64) (string, error) {
	payload := map[string]interface{}{
		"sub":  subject,
		"role": role,
		"exp":  expSeconds,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	signingInput := jwtHeader + "." + b64url(payloadJSON)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64url(mac.Sum(nil)), nil
}

// ── HTTP client helper ────────────────────────────────────────────────────────

type client struct {
	base  string
	token string
	http  *http.Client
}

// post sends a JSON POST request and returns the decoded response body.
// On non-2xx status it logs the error and returns nil, err.
func (c *client) post(path string, body interface{}) (map[string]interface{}, error) {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST %s returned HTTP %d: %s", path, resp.StatusCode, string(respBytes))
	}

	var result map[string]interface{}
	if len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, &result); err != nil {
			// Response may not be JSON (e.g. plain-text ID); wrap it.
			result = map[string]interface{}{"raw": string(respBytes)}
		}
	}
	return result, nil
}

// extractID tries common ID field names from a response body.
func extractID(resp map[string]interface{}) string {
	for _, key := range []string{"id", "did", "election_id", "verifier_id", "card_id", "enrollment_id"} {
		if v, ok := resp[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	if v, ok := resp["raw"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return "(unknown)"
}

// ── Seed data ─────────────────────────────────────────────────────────────────

var testCitizens = []map[string]interface{}{
	{
		"did":          "did:indis:test-citizen-001",
		"display_name": "Ali Hosseini",
		"national_id":  "0012345678",
	},
	{
		"did":          "did:indis:test-citizen-002",
		"display_name": "Maryam Ahmadi",
		"national_id":  "0087654321",
	},
	{
		"did":          "did:indis:test-citizen-003",
		"display_name": "Reza Karimi",
		"national_id":  "0011223344",
	},
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	jwtSecret := os.Getenv("DEV_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "indis-dev-secret-change-in-prod"
	}

	// Generate an admin JWT valid for 1 hour.
	token, err := buildJWT("did:indis:seed-tool", "admin", jwtSecret, time.Now().Add(time.Hour).Unix())
	if err != nil {
		log.Fatalf("failed to generate dev JWT: %v", err)
	}

	c := &client{
		base:  gatewayURL,
		token: token,
		http:  &http.Client{Timeout: 15 * time.Second},
	}

	// Track created IDs for the summary.
	summary := make(map[string][]string)

	// ── Step 1: Register test citizen DIDs ───────────────────
	log.Println("Step 1: Registering test citizen DIDs...")
	for _, citizen := range testCitizens {
		resp, err := c.post("/v1/identity/register", citizen)
		if err != nil {
			log.Printf("  WARN: register %s: %v", citizen["did"], err)
			continue
		}
		id := extractID(resp)
		summary["identity"] = append(summary["identity"], id)
		log.Printf("  OK: registered identity %s", id)
	}

	// ── Step 2: Initiate enrollment for each DID ─────────────
	log.Println("Step 2: Initiating enrollment for each citizen...")
	for _, citizen := range testCitizens {
		payload := map[string]interface{}{
			"did":      citizen["did"],
			"pathway":  "standard",
			"metadata": map[string]string{"source": "seed-tool"},
		}
		resp, err := c.post("/v1/enrollment/initiate", payload)
		if err != nil {
			log.Printf("  WARN: enrollment for %s: %v", citizen["did"], err)
			continue
		}
		id := extractID(resp)
		summary["enrollment"] = append(summary["enrollment"], id)
		log.Printf("  OK: enrollment %s for %s", id, citizen["did"])
	}

	// ── Step 3: Create a test election ───────────────────────
	log.Println("Step 3: Creating a test election...")
	electionPayload := map[string]interface{}{
		"title":       "آزمون انتخابات توسعه",
		"description": "Test election for frontend development — do not use in production.",
		"status":      "open",
		"start_time":  time.Now().Format(time.RFC3339),
		"end_time":    time.Now().Add(72 * time.Hour).Format(time.RFC3339),
		"metadata": map[string]string{
			"region": "nationwide",
			"type":   "parliamentary",
			"env":    "development",
		},
	}
	resp, err := c.post("/v1/electoral/elections", electionPayload)
	if err != nil {
		log.Printf("  WARN: create election: %v", err)
	} else {
		id := extractID(resp)
		summary["election"] = append(summary["election"], id)
		log.Printf("  OK: election %s", id)
	}

	// ── Step 4: Register a test verifier organisation ─────────
	log.Println("Step 4: Registering a test verifier organisation...")
	verifierPayload := map[string]interface{}{
		"name":         "Test Verifier Org",
		"did":          "did:indis:verifier-test-001",
		"allowed_tiers": []int{1, 2},
		"contact_email": "dev-verifier@indis.test",
	}
	resp, err = c.post("/v1/verifier/register", verifierPayload)
	if err != nil {
		log.Printf("  WARN: register verifier: %v", err)
	} else {
		id := extractID(resp)
		summary["verifier"] = append(summary["verifier"], id)
		log.Printf("  OK: verifier %s", id)
	}

	// ── Step 5: Generate a test card ─────────────────────────
	log.Println("Step 5: Generating a test card...")
	cardPayload := map[string]interface{}{
		"did":       testCitizens[0]["did"],
		"card_type": "national_id",
		"metadata": map[string]string{
			"env": "development",
		},
	}
	resp, err = c.post("/v1/card/generate", cardPayload)
	if err != nil {
		log.Printf("  WARN: generate card: %v", err)
	} else {
		id := extractID(resp)
		summary["card"] = append(summary["card"], id)
		log.Printf("  OK: card %s", id)
	}

	// ── Summary ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("  INDIS Dev Seed — Summary")
	fmt.Println("═══════════════════════════════════════════════")
	for category, ids := range summary {
		fmt.Printf("  %-12s %d created\n", category+":", len(ids))
		for _, id := range ids {
			fmt.Printf("               %s\n", id)
		}
	}
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Printf("  Gateway: %s\n", gatewayURL)
	fmt.Println("═══════════════════════════════════════════════")
}
