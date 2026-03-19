// Package service implements business logic for the card service.
//
// It generates ICAO 9303-compliant physical identity card data including
// Machine Readable Zone (MRZ) lines, chip data, QR payloads, and Ed25519
// issuer signatures over the card fields. No biometric raw data is stored
// per PRD FR-016.3.
package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IranProsperityProject/INDIS/services/card/internal/repository"
)

// CardData is the structured card record returned to callers.
type CardData struct {
	DID          string `json:"did"`
	MRZLine1     string `json:"mrz_line1"`      // 44 chars per ICAO 9303 Part 4
	MRZLine2     string `json:"mrz_line2"`      // 44 chars
	ChipDataHex  string `json:"chip_data_hex"`  // DID doc reference + public key
	QRPayloadB64 string `json:"qr_payload_b64"` // base64(JSON)
	IssuerSig    string `json:"issuer_sig"`     // hex Ed25519 sig over SHA-256(mrz1+mrz2+chip)
	IssuedAt     string `json:"issued_at"`
	ExpiresAt    string `json:"expires_at"`
	Status       string `json:"status"` // "active" | "invalidated"
}

// GenerateRequest carries the inputs required to generate a card record.
type GenerateRequest struct {
	DID         string // W3C DID, e.g. "did:indis:abc123"
	HolderName  string // Surname<<Givenname format or plain name
	DateOfBirth string // YYMMDD
	ExpiryDate  string // YYMMDD
}

// Service implements card data generation and lifecycle management.
type Service struct {
	repo       *repository.Repository
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// New creates a Service. seedHex is a 32-byte hex-encoded Ed25519 seed; if empty
// a random key pair is generated at startup.
func New(repo *repository.Repository, seedHex string) (*Service, error) {
	var priv ed25519.PrivateKey
	var pub ed25519.PublicKey

	if seedHex != "" {
		seed, err := hex.DecodeString(seedHex)
		if err != nil {
			return nil, fmt.Errorf("service: decode issuer seed: %w", err)
		}
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("service: issuer seed must be %d bytes, got %d", ed25519.SeedSize, len(seed))
		}
		priv = ed25519.NewKeyFromSeed(seed)
		pub = priv.Public().(ed25519.PublicKey)
	} else {
		var err error
		pub, priv, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("service: generate issuer key: %w", err)
		}
	}

	return &Service{repo: repo, privateKey: priv, publicKey: pub}, nil
}

// GenerateCard produces a full ICAO 9303-compliant card record for the given DID,
// persists it, and returns the structured CardData.
func (s *Service) GenerateCard(ctx context.Context, req GenerateRequest) (*CardData, error) {
	if req.DID == "" {
		return nil, errors.New("service: DID is required")
	}
	if req.HolderName == "" {
		return nil, errors.New("service: holder_name is required")
	}
	if len(req.DateOfBirth) != 6 {
		return nil, errors.New("service: date_of_birth must be YYMMDD")
	}
	if len(req.ExpiryDate) != 6 {
		return nil, errors.New("service: expiry_date must be YYMMDD")
	}

	// Derive a document number from the DID (first 9 alphanumeric chars, uppercase).
	docNum := deriveDocNumber(req.DID)

	mrz1 := buildMRZLine1(req.HolderName)
	mrz2 := buildMRZLine2(docNum, req.DateOfBirth, req.ExpiryDate)

	// Chip data: DID document reference + hex public key.
	chipData := fmt.Sprintf("%s:%s", req.DID, hex.EncodeToString(s.publicKey))
	chipDataHex := hex.EncodeToString([]byte(chipData))

	// QR payload: JSON encoded as base64.
	now := time.Now().UTC()
	expiresAt := parseYYMMDDToTime(req.ExpiryDate)
	qrJSON, err := json.Marshal(map[string]string{
		"did":     req.DID,
		"cert_id": docNum,
		"issued":  now.Format(time.RFC3339),
		"expires": expiresAt.Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("service: marshal qr payload: %w", err)
	}
	qrPayloadB64 := base64.StdEncoding.EncodeToString(qrJSON)

	// Sign SHA-256(mrz1 || mrz2 || chipDataHex).
	sigInput := sha256.Sum256([]byte(mrz1 + mrz2 + chipDataHex))
	sig := ed25519.Sign(s.privateKey, sigInput[:])
	issuerSig := hex.EncodeToString(sig)

	// Generate a unique record ID.
	id, err := generateID("card_")
	if err != nil {
		return nil, fmt.Errorf("service: generate card id: %w", err)
	}

	rec := repository.CardRecord{
		ID:           id,
		DID:          req.DID,
		MRZLine1:     mrz1,
		MRZLine2:     mrz2,
		ChipDataHex:  chipDataHex,
		QRPayloadB64: qrPayloadB64,
		IssuerSig:    issuerSig,
		Status:       "active",
		IssuedAt:     now,
		ExpiresAt:    expiresAt,
	}
	if err = s.repo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("service: persist card: %w", err)
	}

	return recordToCardData(rec), nil
}

// GetCard fetches the card data for a given DID.
func (s *Service) GetCard(ctx context.Context, did string) (*CardData, error) {
	rec, err := s.repo.GetByDID(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("service: get card: %w", err)
	}
	return recordToCardData(*rec), nil
}

// InvalidateCard marks a card as lost or stolen.
func (s *Service) InvalidateCard(ctx context.Context, did, reason string) error {
	if err := s.repo.Invalidate(ctx, did, reason); err != nil {
		return fmt.Errorf("service: invalidate card: %w", err)
	}
	return nil
}

// VerifyCard verifies the issuer signature over a stored card record.
// Returns true when the signature is valid and the card is active.
func (s *Service) VerifyCard(ctx context.Context, did string) (bool, error) {
	rec, err := s.repo.GetByDID(ctx, did)
	if err != nil {
		return false, fmt.Errorf("service: verify card get: %w", err)
	}
	if rec.Status != "active" {
		return false, nil
	}

	sigInput := sha256.Sum256([]byte(rec.MRZLine1 + rec.MRZLine2 + rec.ChipDataHex))
	sigBytes, err := hex.DecodeString(rec.IssuerSig)
	if err != nil {
		return false, fmt.Errorf("service: decode issuer sig: %w", err)
	}

	return ed25519.Verify(s.publicKey, sigInput[:], sigBytes), nil
}

// ── MRZ construction ─────────────────────────────────────────────────────────

// buildMRZLine1 constructs the 44-character MRZ line 1 per ICAO 9303 Part 4.
// Format: IP<IRN<name_field...> padded to 44 chars with '<'.
func buildMRZLine1(holderName string) string {
	// Normalise name: uppercase, replace spaces with '<', strip unsupported chars.
	name := normalizeMRZField(holderName)
	// Fixed prefix: document type 'I' + filler 'P' + country code 'IRN'.
	prefix := "IP<IRN"
	// Remaining 38 chars for the name field.
	nameField := padRight(name, 38, '<')
	line := prefix + nameField
	return line[:44]
}

// buildMRZLine2 constructs the 44-character MRZ line 2 per ICAO 9303 Part 4.
// Format: doc_num(9) + check(1) + nationality(3) + dob(6) + check(1) + expiry(6) + check(1) + personal_num(14) + check(1) + composite_check(1)
func buildMRZLine2(docNum, dob, expiry string) string {
	docNum9 := padRight(docNum, 9, '<')
	docCheck := string([]byte{checkDigit(docNum9)})

	nationality := "IRN"
	dobCheck := string([]byte{checkDigit(dob)})

	expiryCheck := string([]byte{checkDigit(expiry)})

	personalNum := padRight("", 14, '<')
	personalCheck := string([]byte{checkDigit(personalNum)})

	// Composite check covers docNum+check + dob+check + expiry+check + personal+check.
	compositeField := docNum9 + docCheck + dob + dobCheck + expiry + expiryCheck + personalNum + personalCheck
	compositeCheck := string([]byte{checkDigit(compositeField)})

	line := docNum9 + docCheck + nationality + dob + dobCheck + expiry + expiryCheck + personalNum + personalCheck + compositeCheck
	return padRight(line, 44, '<')[:44]
}

// checkDigit computes the ICAO 9303 check digit for s.
// Weights cycle: 7, 3, 1.
// Character values: '0'-'9' → 0-9, 'A'-'Z' → 10-35, '<' → 0.
func checkDigit(s string) byte {
	weights := [3]int{7, 3, 1}
	sum := 0
	for i, c := range strings.ToUpper(s) {
		var v int
		switch {
		case c >= '0' && c <= '9':
			v = int(c - '0')
		case c >= 'A' && c <= 'Z':
			v = int(c-'A') + 10
		default: // '<' and anything else
			v = 0
		}
		sum += v * weights[i%3]
	}
	return byte('0' + sum%10)
}

// normalizeMRZField converts a name to MRZ-safe uppercase ASCII, replacing
// spaces and hyphens with '<' and stripping unsupported characters.
func normalizeMRZField(s string) string {
	s = strings.ToUpper(s)
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
			b.WriteRune(c)
		case c == ' ' || c == '-':
			b.WriteByte('<')
		case c >= '0' && c <= '9':
			b.WriteRune(c)
		// Skip all other characters (e.g. Persian letters — transliteration is
		// handled by the enrollment service before calling card generation).
		}
	}
	return b.String()
}

// padRight appends fill bytes to s until it reaches length n, then truncates.
func padRight(s string, n int, fill byte) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(string(fill), n-len(s))
}

// deriveDocNumber extracts up to 9 alphanumeric characters from a DID to use
// as the MRZ document number.
func deriveDocNumber(did string) string {
	var b strings.Builder
	for _, c := range strings.ToUpper(did) {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			if b.Len() == 9 {
				break
			}
		}
	}
	return padRight(b.String(), 9, '<')
}

// parseYYMMDDToTime converts a YYMMDD string to a time.Time (UTC, midnight).
// Years 00-49 are interpreted as 2000-2049; 50-99 as 1950-1999.
func parseYYMMDDToTime(yymmdd string) time.Time {
	if len(yymmdd) != 6 {
		return time.Now().UTC().AddDate(5, 0, 0) // fallback
	}
	yy := atoi2(yymmdd[0:2])
	mm := atoi2(yymmdd[2:4])
	dd := atoi2(yymmdd[4:6])

	year := 2000 + yy
	if yy >= 50 {
		year = 1900 + yy
	}
	return time.Date(year, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)
}

// atoi2 converts a 2-character ASCII decimal string to int. Returns 0 on error.
func atoi2(s string) int {
	if len(s) != 2 {
		return 0
	}
	hi := int(s[0] - '0')
	lo := int(s[1] - '0')
	if hi < 0 || hi > 9 || lo < 0 || lo > 9 {
		return 0
	}
	return hi*10 + lo
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// generateID produces a random URL-safe identifier with the given prefix.
func generateID(prefix string) (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// recordToCardData converts a repository record to the public CardData struct.
func recordToCardData(rec repository.CardRecord) *CardData {
	return &CardData{
		DID:          rec.DID,
		MRZLine1:     rec.MRZLine1,
		MRZLine2:     rec.MRZLine2,
		ChipDataHex:  rec.ChipDataHex,
		QRPayloadB64: rec.QRPayloadB64,
		IssuerSig:    rec.IssuerSig,
		IssuedAt:     rec.IssuedAt.Format(time.RFC3339),
		ExpiresAt:    rec.ExpiresAt.Format(time.RFC3339),
		Status:       rec.Status,
	}
}
