package hsm

import (
	"bytes"
	"context"
	"sort"
	"testing"
	"time"
)

var ctx = context.Background()

// ---------------------------------------------------------------------------
// GenerateKey
// ---------------------------------------------------------------------------

func TestSoftwareKeyManager_GenerateKey_Ed25519(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "signing-key", KeyTypeEd25519); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	pub, err := m.GetPublicKey(ctx, "signing-key")
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}
	if len(pub) == 0 {
		t.Error("GetPublicKey returned empty bytes")
	}
}

func TestSoftwareKeyManager_GenerateKey_DuplicateName(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "k", KeyTypeEd25519); err != nil {
		t.Fatalf("first GenerateKey: %v", err)
	}
	if err := m.GenerateKey(ctx, "k", KeyTypeEd25519); err == nil {
		t.Error("second GenerateKey with the same name should return an error")
	}
}

func TestSoftwareKeyManager_GenerateKey_UnsupportedType(t *testing.T) {
	m := NewSoftwareKeyManager()
	err := m.GenerateKey(ctx, "k", KeyType("rsa-4096"))
	if err == nil {
		t.Error("GenerateKey with unsupported type should return an error")
	}
}

// ---------------------------------------------------------------------------
// GenerateAndSign (Ed25519)
// ---------------------------------------------------------------------------

func TestSoftwareKeyManager_GenerateAndSign_Ed25519(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "ed-key", KeyTypeEd25519); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	msg := []byte("INDIS identity claim")
	sig, err := m.Sign(ctx, "ed-key", msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) == 0 {
		t.Error("Sign returned empty signature")
	}
	valid, err := m.Verify(ctx, "ed-key", msg, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Error("Verify returned false for a valid Ed25519 signature")
	}
}

func TestSoftwareKeyManager_Sign_TamperedMessage(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "k", KeyTypeEd25519)
	sig, _ := m.Sign(ctx, "k", []byte("original"))
	valid, err := m.Verify(ctx, "k", []byte("tampered"), sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if valid {
		t.Error("Verify should return false for a tampered message")
	}
}

func TestSoftwareKeyManager_Sign_ECDSAKey(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "ec-key", KeyTypeECDSAP256); err != nil {
		t.Fatalf("GenerateKey ECDSA: %v", err)
	}
	msg := []byte("ecdsa message")
	sig, err := m.Sign(ctx, "ec-key", msg)
	if err != nil {
		t.Fatalf("Sign ECDSA: %v", err)
	}
	valid, err := m.Verify(ctx, "ec-key", msg, sig)
	if err != nil {
		t.Fatalf("Verify ECDSA: %v", err)
	}
	if !valid {
		t.Error("Verify ECDSA returned false for a valid signature")
	}
}

func TestSoftwareKeyManager_Sign_Dilithium3(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "dil-key", KeyTypeDilithium3); err != nil {
		t.Fatalf("GenerateKey Dilithium3: %v", err)
	}
	msg := []byte("post-quantum identity claim")
	sig, err := m.Sign(ctx, "dil-key", msg)
	if err != nil {
		t.Fatalf("Sign Dilithium3: %v", err)
	}
	valid, err := m.Verify(ctx, "dil-key", msg, sig)
	if err != nil {
		t.Fatalf("Verify Dilithium3: %v", err)
	}
	if !valid {
		t.Error("Verify Dilithium3 returned false for a valid signature")
	}
}

func TestSoftwareKeyManager_Sign_AESKeyReturnsError(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "sym", KeyTypeAES256)
	_, err := m.Sign(ctx, "sym", []byte("data"))
	if err == nil {
		t.Error("Sign on an AES key should return an error")
	}
}

func TestSoftwareKeyManager_Sign_UnknownKeyReturnsError(t *testing.T) {
	m := NewSoftwareKeyManager()
	_, err := m.Sign(ctx, "nonexistent", []byte("data"))
	if err == nil {
		t.Error("Sign on a nonexistent key should return an error")
	}
}

// ---------------------------------------------------------------------------
// GetPublicKey
// ---------------------------------------------------------------------------

func TestSoftwareKeyManager_GetPublicKey_AESKeyReturnsError(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "sym", KeyTypeAES256)
	_, err := m.GetPublicKey(ctx, "sym")
	if err == nil {
		t.Error("GetPublicKey on a symmetric key should return an error")
	}
}

// ---------------------------------------------------------------------------
// EncryptDecrypt
// ---------------------------------------------------------------------------

func TestSoftwareKeyManager_EncryptDecrypt(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "enc-key", KeyTypeAES256); err != nil {
		t.Fatalf("GenerateKey AES256: %v", err)
	}
	plaintext := []byte("sensitive biometric data")
	ct, err := m.EncryptData(ctx, "enc-key", plaintext)
	if err != nil {
		t.Fatalf("EncryptData: %v", err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Error("EncryptData returned plaintext unchanged")
	}
	pt, err := m.DecryptData(ctx, "enc-key", ct)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Errorf("DecryptData = %q, want %q", pt, plaintext)
	}
}

func TestSoftwareKeyManager_EncryptDecrypt_EmptyPlaintext(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "k", KeyTypeAES256)
	ct, err := m.EncryptData(ctx, "k", []byte{})
	if err != nil {
		t.Fatalf("EncryptData empty: %v", err)
	}
	pt, err := m.DecryptData(ctx, "k", ct)
	if err != nil {
		t.Fatalf("DecryptData empty: %v", err)
	}
	if len(pt) != 0 {
		t.Errorf("expected empty plaintext, got %v", pt)
	}
}

func TestSoftwareKeyManager_Encrypt_NonAESKeyReturnsError(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "k", KeyTypeEd25519)
	_, err := m.EncryptData(ctx, "k", []byte("data"))
	if err == nil {
		t.Error("EncryptData on a non-AES key should return an error")
	}
}

func TestSoftwareKeyManager_Decrypt_WrongKey(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "k1", KeyTypeAES256)
	_ = m.GenerateKey(ctx, "k2", KeyTypeAES256)
	ct, _ := m.EncryptData(ctx, "k1", []byte("data"))
	_, err := m.DecryptData(ctx, "k2", ct)
	if err == nil {
		t.Error("DecryptData with wrong key should return an error")
	}
}

// ---------------------------------------------------------------------------
// RotateKey
// ---------------------------------------------------------------------------

func TestSoftwareKeyManager_RotateKey_Ed25519(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.GenerateKey(ctx, "k", KeyTypeEd25519); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	pub1, _ := m.GetPublicKey(ctx, "k")

	if err := m.RotateKey(ctx, "k"); err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	pub2, _ := m.GetPublicKey(ctx, "k")

	if bytes.Equal(pub1, pub2) {
		t.Error("public key should change after rotation")
	}

	// New key should be functional for signing.
	msg := []byte("after rotation")
	sig, err := m.Sign(ctx, "k", msg)
	if err != nil {
		t.Fatalf("Sign after rotation: %v", err)
	}
	valid, err := m.Verify(ctx, "k", msg, sig)
	if err != nil {
		t.Fatalf("Verify after rotation: %v", err)
	}
	if !valid {
		t.Error("Verify should succeed with the new key after rotation")
	}
}

func TestSoftwareKeyManager_RotateKey_AES_DecryptOldCiphertext(t *testing.T) {
	m := NewSoftwareKeyManager()
	_ = m.GenerateKey(ctx, "k", KeyTypeAES256)

	original := []byte("old plaintext")
	ct, err := m.EncryptData(ctx, "k", original)
	if err != nil {
		t.Fatalf("EncryptData before rotation: %v", err)
	}

	if err := m.RotateKey(ctx, "k"); err != nil {
		t.Fatalf("RotateKey: %v", err)
	}

	// Old ciphertext must still be decryptable after rotation.
	pt, err := m.DecryptData(ctx, "k", ct)
	if err != nil {
		t.Fatalf("DecryptData old ciphertext after rotation: %v", err)
	}
	if !bytes.Equal(pt, original) {
		t.Errorf("decrypted = %q, want %q", pt, original)
	}

	// New ciphertext encrypted after rotation must also work.
	newPlain := []byte("new plaintext")
	ct2, err := m.EncryptData(ctx, "k", newPlain)
	if err != nil {
		t.Fatalf("EncryptData after rotation: %v", err)
	}
	pt2, err := m.DecryptData(ctx, "k", ct2)
	if err != nil {
		t.Fatalf("DecryptData new ciphertext after rotation: %v", err)
	}
	if !bytes.Equal(pt2, newPlain) {
		t.Errorf("decrypted new = %q, want %q", pt2, newPlain)
	}
}

func TestSoftwareKeyManager_RotateKey_NonexistentKeyReturnsError(t *testing.T) {
	m := NewSoftwareKeyManager()
	if err := m.RotateKey(ctx, "ghost"); err == nil {
		t.Error("RotateKey on nonexistent key should return an error")
	}
}

// ---------------------------------------------------------------------------
// ListKeys
// ---------------------------------------------------------------------------

func TestSoftwareKeyManager_ListKeys(t *testing.T) {
	m := NewSoftwareKeyManager()
	names := []string{"alpha", "beta", "gamma"}
	for _, n := range names {
		if err := m.GenerateKey(ctx, n, KeyTypeEd25519); err != nil {
			t.Fatalf("GenerateKey %q: %v", n, err)
		}
	}
	got, err := m.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(got) != len(names) {
		t.Errorf("ListKeys returned %d keys, want %d", len(got), len(names))
	}
	sort.Strings(got)
	sort.Strings(names)
	for i, want := range names {
		if got[i] != want {
			t.Errorf("ListKeys[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestSoftwareKeyManager_ListKeys_Empty(t *testing.T) {
	m := NewSoftwareKeyManager()
	keys, err := m.ListKeys(ctx)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected empty list, got %v", keys)
	}
}

// ---------------------------------------------------------------------------
// RotationPolicy
// ---------------------------------------------------------------------------

func TestDefaultRotationPolicy_Ed25519(t *testing.T) {
	p := DefaultRotationPolicy(KeyTypeEd25519)
	if p.MaxAge != 90*24*time.Hour {
		t.Errorf("Ed25519 MaxAge = %v, want 90 days", p.MaxAge)
	}
	if p.MaxSignatures != 1_000_000 {
		t.Errorf("Ed25519 MaxSignatures = %d, want 1000000", p.MaxSignatures)
	}
}

func TestDefaultRotationPolicy_AES256(t *testing.T) {
	p := DefaultRotationPolicy(KeyTypeAES256)
	if p.MaxAge != 365*24*time.Hour {
		t.Errorf("AES256 MaxAge = %v, want 365 days", p.MaxAge)
	}
	if p.MaxSignatures != 0 {
		t.Errorf("AES256 MaxSignatures = %d, want 0", p.MaxSignatures)
	}
}

func TestNeedsRotation_Expired(t *testing.T) {
	policy := RotationPolicy{MaxAge: 90 * 24 * time.Hour}
	createdAt := time.Now().Add(-91 * 24 * time.Hour)
	if !NeedsRotation(createdAt, policy) {
		t.Error("NeedsRotation should return true for key older than MaxAge")
	}
}

func TestNeedsRotation_NotExpired(t *testing.T) {
	policy := RotationPolicy{MaxAge: 90 * 24 * time.Hour}
	createdAt := time.Now().Add(-10 * 24 * time.Hour)
	if NeedsRotation(createdAt, policy) {
		t.Error("NeedsRotation should return false for a fresh key")
	}
}

func TestNeedsRotation_ZeroMaxAge(t *testing.T) {
	policy := RotationPolicy{MaxAge: 0}
	if NeedsRotation(time.Time{}, policy) {
		t.Error("NeedsRotation should return false when MaxAge is zero")
	}
}
