package crypto

import (
	"bytes"
	"testing"
)

// ---------------------------------------------------------------------------
// GenerateDilithiumKeyPair
// ---------------------------------------------------------------------------

func TestGenerateDilithiumKeyPair(t *testing.T) {
	kp, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("GenerateDilithiumKeyPair() error = %v", err)
	}
	if kp == nil {
		t.Fatal("GenerateDilithiumKeyPair() returned nil key pair")
	}
	if got := len(kp.PublicKey); got != Dilithium3PublicKeySize {
		t.Errorf("PublicKey length = %d, want %d", got, Dilithium3PublicKeySize)
	}
	if got := len(kp.PrivateKey); got != Dilithium3PrivateKeySize {
		t.Errorf("PrivateKey length = %d, want %d", got, Dilithium3PrivateKeySize)
	}
}

func TestGenerateDilithiumKeyPair_TwoCallsProduceDifferentKeys(t *testing.T) {
	kp1, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("first GenerateDilithiumKeyPair: %v", err)
	}
	kp2, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("second GenerateDilithiumKeyPair: %v", err)
	}
	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		t.Error("two calls to GenerateDilithiumKeyPair produced identical public keys")
	}
	if bytes.Equal(kp1.PrivateKey, kp2.PrivateKey) {
		t.Error("two calls to GenerateDilithiumKeyPair produced identical private keys")
	}
}

// ---------------------------------------------------------------------------
// SignDilithium / VerifyDilithium
// ---------------------------------------------------------------------------

func TestSignVerifyDilithium_RoundTrip(t *testing.T) {
	kp, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("GenerateDilithiumKeyPair: %v", err)
	}
	msg := []byte("INDIS voter eligibility payload")
	sig, err := SignDilithium(kp.PrivateKey, msg)
	if err != nil {
		t.Fatalf("SignDilithium: %v", err)
	}
	if len(sig) != Dilithium3SignatureSize {
		t.Errorf("signature length = %d, want %d", len(sig), Dilithium3SignatureSize)
	}
	valid, err := VerifyDilithium(kp.PublicKey, msg, sig)
	if err != nil {
		t.Fatalf("VerifyDilithium: %v", err)
	}
	if !valid {
		t.Error("VerifyDilithium returned false for a valid signature")
	}
}

func TestSignVerifyDilithium_TamperedMessage(t *testing.T) {
	kp, _ := GenerateDilithiumKeyPair()
	msg := []byte("original message")
	sig, err := SignDilithium(kp.PrivateKey, msg)
	if err != nil {
		t.Fatalf("SignDilithium: %v", err)
	}
	tampered := []byte("tampered message")
	valid, err := VerifyDilithium(kp.PublicKey, tampered, sig)
	if err != nil {
		t.Fatalf("VerifyDilithium: %v", err)
	}
	if valid {
		t.Error("VerifyDilithium should return false for a tampered message")
	}
}

func TestSignVerifyDilithium_WrongPublicKey(t *testing.T) {
	kp1, _ := GenerateDilithiumKeyPair()
	kp2, _ := GenerateDilithiumKeyPair()
	msg := []byte("hello")
	sig, err := SignDilithium(kp1.PrivateKey, msg)
	if err != nil {
		t.Fatalf("SignDilithium: %v", err)
	}
	valid, err := VerifyDilithium(kp2.PublicKey, msg, sig)
	if err != nil {
		t.Fatalf("VerifyDilithium: %v", err)
	}
	if valid {
		t.Error("VerifyDilithium should return false when using a different public key")
	}
}

func TestSignDilithium_InvalidKeyLength(t *testing.T) {
	_, err := SignDilithium([]byte("tooshort"), []byte("msg"))
	if err == nil {
		t.Error("SignDilithium should return error for invalid private key length")
	}
}

func TestVerifyDilithium_InvalidPublicKeyLength(t *testing.T) {
	kp, _ := GenerateDilithiumKeyPair()
	sig, _ := SignDilithium(kp.PrivateKey, []byte("msg"))
	_, err := VerifyDilithium([]byte("tooshort"), []byte("msg"), sig)
	if err == nil {
		t.Error("VerifyDilithium should return error for invalid public key length")
	}
}

func TestVerifyDilithium_InvalidSignatureLength(t *testing.T) {
	kp, _ := GenerateDilithiumKeyPair()
	_, err := VerifyDilithium(kp.PublicKey, []byte("msg"), []byte("tooshort"))
	if err == nil {
		t.Error("VerifyDilithium should return error for invalid signature length")
	}
}

func TestSignVerifyDilithium_EmptyMessage(t *testing.T) {
	kp, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("GenerateDilithiumKeyPair: %v", err)
	}
	sig, err := SignDilithium(kp.PrivateKey, []byte{})
	if err != nil {
		t.Fatalf("SignDilithium empty message: %v", err)
	}
	valid, err := VerifyDilithium(kp.PublicKey, []byte{}, sig)
	if err != nil {
		t.Fatalf("VerifyDilithium empty message: %v", err)
	}
	if !valid {
		t.Error("VerifyDilithium should return true for empty message round-trip")
	}
}

// ---------------------------------------------------------------------------
// MigrationNeeded
// ---------------------------------------------------------------------------

func TestMigrationNeeded(t *testing.T) {
	cases := []struct {
		keyType  KeyType
		expected bool
	}{
		{KeyTypeEd25519, true},
		{KeyTypeECDSAP256, true},
		{KeyTypeDilithium3, false},
		{KeyType("unknown"), false},
	}
	for _, tc := range cases {
		got := MigrationNeeded(tc.keyType)
		if got != tc.expected {
			t.Errorf("MigrationNeeded(%q) = %v, want %v", tc.keyType, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// RecommendedKeyType
// ---------------------------------------------------------------------------

func TestRecommendedKeyType(t *testing.T) {
	cases := []struct {
		useCase  string
		expected KeyType
	}{
		{"long-term", KeyTypeDilithium3},
		{"credential", KeyTypeDilithium3},
		{"operational", KeyTypeEd25519},
		{"session", KeyTypeEd25519},
		{"", KeyTypeEd25519},
	}
	for _, tc := range cases {
		got := RecommendedKeyType(tc.useCase)
		if got != tc.expected {
			t.Errorf("RecommendedKeyType(%q) = %q, want %q", tc.useCase, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// MigrateKeyPair
// ---------------------------------------------------------------------------

func TestMigrateKeyPair(t *testing.T) {
	existing, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateEd25519KeyPair: %v", err)
	}
	newPair, err := MigrateKeyPair(existing)
	if err != nil {
		t.Fatalf("MigrateKeyPair: %v", err)
	}
	if newPair == nil {
		t.Fatal("MigrateKeyPair returned nil")
	}
	if len(newPair.PublicKey) != Dilithium3PublicKeySize {
		t.Errorf("migrated public key length = %d, want %d", len(newPair.PublicKey), Dilithium3PublicKeySize)
	}
	if len(newPair.PrivateKey) != Dilithium3PrivateKeySize {
		t.Errorf("migrated private key length = %d, want %d", len(newPair.PrivateKey), Dilithium3PrivateKeySize)
	}
}

func TestMigrateKeyPair_NilInput(t *testing.T) {
	_, err := MigrateKeyPair(nil)
	if err == nil {
		t.Error("MigrateKeyPair(nil) should return an error")
	}
}

func TestMigrateKeyPair_AlreadyDilithium(t *testing.T) {
	// Simulate a key pair that is already post-quantum.
	pseudo := &KeyPair{Type: KeyTypeDilithium3}
	_, err := MigrateKeyPair(pseudo)
	if err == nil {
		t.Error("MigrateKeyPair should return error when key type does not need migration")
	}
}

func TestMigrateKeyPair_ECDSAInput(t *testing.T) {
	kp, _, err := GenerateECDSAP256KeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAP256KeyPair: %v", err)
	}
	newPair, err := MigrateKeyPair(kp)
	if err != nil {
		t.Fatalf("MigrateKeyPair from ECDSA: %v", err)
	}
	if newPair == nil {
		t.Fatal("MigrateKeyPair returned nil for ECDSA input")
	}
}
