package crypto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

// ---------------------------------------------------------------------------
// GenerateEd25519KeyPair
// ---------------------------------------------------------------------------

func TestGenerateEd25519KeyPair_KeyLengths(t *testing.T) {
	kp, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateEd25519KeyPair() error = %v", err)
	}
	if kp.Type != KeyTypeEd25519 {
		t.Errorf("Type = %q, want %q", kp.Type, KeyTypeEd25519)
	}
	if got := len(kp.PublicKey); got != ed25519.PublicKeySize {
		t.Errorf("PublicKey length = %d, want %d", got, ed25519.PublicKeySize)
	}
	if got := len(kp.PrivateKey); got != ed25519.PrivateKeySize {
		t.Errorf("PrivateKey length = %d, want %d", got, ed25519.PrivateKeySize)
	}
}

// ---------------------------------------------------------------------------
// SignEd25519 / VerifyEd25519
// ---------------------------------------------------------------------------

func TestSignVerifyEd25519_RoundTrip(t *testing.T) {
	kp, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateEd25519KeyPair: %v", err)
	}
	msg := []byte("hello INDIS")
	sig, err := SignEd25519(kp.PrivateKey, msg)
	if err != nil {
		t.Fatalf("SignEd25519: %v", err)
	}
	if !VerifyEd25519(kp.PublicKey, msg, sig) {
		t.Error("VerifyEd25519 returned false for a valid signature")
	}
}

func TestVerifyEd25519_WrongKey(t *testing.T) {
	kp1, _ := GenerateEd25519KeyPair()
	kp2, _ := GenerateEd25519KeyPair()
	msg := []byte("hello INDIS")
	sig, err := SignEd25519(kp1.PrivateKey, msg)
	if err != nil {
		t.Fatalf("SignEd25519: %v", err)
	}
	if VerifyEd25519(kp2.PublicKey, msg, sig) {
		t.Error("VerifyEd25519 should return false when using a different public key")
	}
}

func TestVerifyEd25519_TamperedMessage(t *testing.T) {
	kp, _ := GenerateEd25519KeyPair()
	msg := []byte("original message")
	sig, err := SignEd25519(kp.PrivateKey, msg)
	if err != nil {
		t.Fatalf("SignEd25519: %v", err)
	}
	tampered := []byte("tampered message")
	if VerifyEd25519(kp.PublicKey, tampered, sig) {
		t.Error("VerifyEd25519 should return false for a tampered message")
	}
}

func TestSignEd25519_InvalidKeyLength(t *testing.T) {
	_, err := SignEd25519([]byte("tooshort"), []byte("msg"))
	if err == nil {
		t.Error("SignEd25519 should return error for invalid private key length")
	}
}

func TestVerifyEd25519_InvalidPublicKeyLength(t *testing.T) {
	result := VerifyEd25519([]byte("tooshort"), []byte("msg"), Signature("fakesig"))
	if result {
		t.Error("VerifyEd25519 should return false for invalid public key length")
	}
}

// ---------------------------------------------------------------------------
// GenerateECDSAP256KeyPair
// ---------------------------------------------------------------------------

func TestGenerateECDSAP256KeyPair_PublicKeyNotNil(t *testing.T) {
	kp, priv, err := GenerateECDSAP256KeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAP256KeyPair: %v", err)
	}
	if priv == nil {
		t.Fatal("returned *ecdsa.PrivateKey is nil")
	}
	if priv.PublicKey.X == nil || priv.PublicKey.Y == nil {
		t.Error("ECDSA public key X or Y is nil")
	}
	if kp.Type != KeyTypeECDSAP256 {
		t.Errorf("Type = %q, want %q", kp.Type, KeyTypeECDSAP256)
	}
	if len(kp.PublicKey) == 0 {
		t.Error("compressed public key bytes must not be empty")
	}
	if len(kp.PrivateKey) == 0 {
		t.Error("private key bytes must not be empty")
	}
}

// ---------------------------------------------------------------------------
// SignECDSAP256 / VerifyECDSAP256
// ---------------------------------------------------------------------------

func TestSignVerifyECDSAP256_RoundTrip(t *testing.T) {
	_, priv, err := GenerateECDSAP256KeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAP256KeyPair: %v", err)
	}
	msg := []byte("hello INDIS ECDSA")
	sig, err := SignECDSAP256(priv, msg)
	if err != nil {
		t.Fatalf("SignECDSAP256: %v", err)
	}
	if !VerifyECDSAP256(&priv.PublicKey, msg, sig) {
		t.Error("VerifyECDSAP256 returned false for a valid signature")
	}
}

func TestVerifyECDSAP256_TamperedMessage(t *testing.T) {
	_, priv, _ := GenerateECDSAP256KeyPair()
	msg := []byte("original")
	sig, err := SignECDSAP256(priv, msg)
	if err != nil {
		t.Fatalf("SignECDSAP256: %v", err)
	}
	tampered := []byte("tampered")
	if VerifyECDSAP256(&priv.PublicKey, tampered, sig) {
		t.Error("VerifyECDSAP256 should return false for a tampered message")
	}
}

// ---------------------------------------------------------------------------
// EncryptAES256GCM / DecryptAES256GCM
// ---------------------------------------------------------------------------

func TestEncryptDecryptAES256GCM_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("sensitive identity data")
	ciphertext, err := EncryptAES256GCM(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAES256GCM: %v", err)
	}
	decrypted, err := DecryptAES256GCM(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAES256GCM: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptAES256GCM_WrongKeyLength(t *testing.T) {
	_, err := EncryptAES256GCM([]byte("tooshort"), []byte("data"))
	if err == nil {
		t.Error("EncryptAES256GCM should return error for key that is not 32 bytes")
	}
}

func TestDecryptAES256GCM_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = 0xFF
	}
	ct, err := EncryptAES256GCM(key1, []byte("data"))
	if err != nil {
		t.Fatalf("EncryptAES256GCM: %v", err)
	}
	_, err = DecryptAES256GCM(key2, ct)
	if err == nil {
		t.Error("DecryptAES256GCM should return error when wrong key is used")
	}
}

func TestDecryptAES256GCM_WrongKeyLength(t *testing.T) {
	_, err := DecryptAES256GCM([]byte("tooshort"), []byte("data"))
	if err == nil {
		t.Error("DecryptAES256GCM should return error for key that is not 32 bytes")
	}
}

func TestDecryptAES256GCM_TruncatedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	// Provide only 4 bytes — shorter than the 12-byte GCM nonce.
	_, err := DecryptAES256GCM(key, []byte{0x01, 0x02, 0x03, 0x04})
	if err == nil {
		t.Error("DecryptAES256GCM should return error for truncated ciphertext")
	}
}

func TestEncryptDecryptAES256GCM_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	ct, err := EncryptAES256GCM(key, []byte{})
	if err != nil {
		t.Fatalf("EncryptAES256GCM with empty plaintext: %v", err)
	}
	pt, err := DecryptAES256GCM(key, ct)
	if err != nil {
		t.Fatalf("DecryptAES256GCM with empty plaintext: %v", err)
	}
	if len(pt) != 0 {
		t.Errorf("expected empty plaintext, got %v", pt)
	}
}

// ---------------------------------------------------------------------------
// HashSHA256
// ---------------------------------------------------------------------------

func TestHashSHA256_Deterministic(t *testing.T) {
	data := []byte("INDIS citizen data")
	h1 := HashSHA256(data)
	h2 := HashSHA256(data)
	if !bytes.Equal(h1, h2) {
		t.Error("HashSHA256 is not deterministic")
	}
}

func TestHashSHA256_KnownVector_EmptyInput(t *testing.T) {
	// SHA-256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	want, _ := hex.DecodeString("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	got := HashSHA256([]byte{})
	if !bytes.Equal(got, want) {
		t.Errorf("HashSHA256(\"\") = %x, want %x", got, want)
	}
}

func TestHashSHA256_Length(t *testing.T) {
	got := HashSHA256([]byte("some data"))
	if len(got) != 32 {
		t.Errorf("HashSHA256 output length = %d, want 32", len(got))
	}
}

func TestHashSHA256_DifferentInputsDifferentOutputs(t *testing.T) {
	h1 := HashSHA256([]byte("a"))
	h2 := HashSHA256([]byte("b"))
	if bytes.Equal(h1, h2) {
		t.Error("HashSHA256 of different inputs should produce different digests")
	}
}

// ---------------------------------------------------------------------------
// GenerateRandomKey
// ---------------------------------------------------------------------------

func TestGenerateRandomKey_CorrectLength(t *testing.T) {
	tests := []struct{ n int }{{16}, {32}, {64}, {1}}
	for _, tc := range tests {
		key, err := GenerateRandomKey(tc.n)
		if err != nil {
			t.Errorf("GenerateRandomKey(%d): %v", tc.n, err)
			continue
		}
		if len(key) != tc.n {
			t.Errorf("GenerateRandomKey(%d) length = %d, want %d", tc.n, len(key), tc.n)
		}
	}
}

func TestGenerateRandomKey_TwoCallsDiffer(t *testing.T) {
	k1, err1 := GenerateRandomKey(32)
	k2, err2 := GenerateRandomKey(32)
	if err1 != nil || err2 != nil {
		t.Fatalf("GenerateRandomKey errors: %v, %v", err1, err2)
	}
	if bytes.Equal(k1, k2) {
		t.Error("two successive GenerateRandomKey(32) calls returned identical keys")
	}
}
