// Package hsm — software.go
// In-memory software KeyManager implementation for development and testing.
//
// SECURITY NOTE: Keys stored by SoftwareKeyManager are held in process memory
// and are permanently lost when the process exits. This backend must NEVER be
// used in production. Use VaultKeyManager in production deployments.
package hsm

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
)

// softwareKey holds the in-memory representation of a single managed key.
type softwareKey struct {
	keyType    KeyType
	publicKey  []byte // nil for symmetric keys
	privateKey []byte // nil for symmetric keys
	aesKey     []byte // non-nil only for KeyTypeAES256
	// For ECDSA keys we also store the full ecdsa.PrivateKey so we can sign.
	ecPriv *ecdsa.PrivateKey
	// versions tracks rotated AES key versions (newest first).
	aesVersions [][]byte
}

// ecdsaSig is used to marshal/unmarshal ECDSA signatures as JSON.
type ecdsaSig struct {
	R []byte `json:"r"`
	S []byte `json:"s"`
}

// SoftwareKeyManager implements KeyManager using in-memory key storage.
//
// FOR DEVELOPMENT AND TESTING ONLY — keys are lost on restart.
// Use VaultKeyManager in production.
type SoftwareKeyManager struct {
	mu   sync.RWMutex
	keys map[string]*softwareKey
}

// NewSoftwareKeyManager creates a new SoftwareKeyManager with no keys.
func NewSoftwareKeyManager() *SoftwareKeyManager {
	return &SoftwareKeyManager{
		keys: make(map[string]*softwareKey),
	}
}

// GenerateKey generates a new key under the given name.
// Returns an error if a key with that name already exists.
func (m *SoftwareKeyManager) GenerateKey(_ context.Context, name string, keyType KeyType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keys[name]; exists {
		return fmt.Errorf("hsm: software generate: key %q already exists", name)
	}

	k, err := generateSoftwareKey(keyType)
	if err != nil {
		return err
	}
	m.keys[name] = k
	return nil
}

// generateSoftwareKey creates a new softwareKey of the requested type.
func generateSoftwareKey(keyType KeyType) (*softwareKey, error) {
	switch keyType {
	case KeyTypeEd25519:
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("hsm: software generate ed25519: %w", err)
		}
		return &softwareKey{
			keyType:    KeyTypeEd25519,
			publicKey:  []byte(pub),
			privateKey: []byte(priv),
		}, nil

	case KeyTypeECDSAP256:
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("hsm: software generate ecdsa-p256: %w", err)
		}
		pubBytes := elliptic.MarshalCompressed(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)
		return &softwareKey{
			keyType:    KeyTypeECDSAP256,
			publicKey:  pubBytes,
			privateKey: priv.D.Bytes(),
			ecPriv:     priv,
		}, nil

	case KeyTypeDilithium3:
		// Placeholder: use Ed25519 padded to Dilithium3 sizes.
		// Replace with FIPS 204-compliant implementation before production.
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("hsm: software generate dilithium3 (placeholder): %w", err)
		}
		marker := [4]byte{0xD1, 0x1B, 0x10, 0x4D}
		pubPadded := make([]byte, 1952)
		copy(pubPadded[0:4], marker[:])
		copy(pubPadded[4:4+ed25519.PublicKeySize], pub)
		privPadded := make([]byte, 4000)
		copy(privPadded[0:4], marker[:])
		copy(privPadded[4:4+ed25519.PrivateKeySize], priv)
		return &softwareKey{
			keyType:    KeyTypeDilithium3,
			publicKey:  pubPadded,
			privateKey: privPadded,
		}, nil

	case KeyTypeAES256:
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("hsm: software generate aes256-gcm: %w", err)
		}
		return &softwareKey{
			keyType: KeyTypeAES256,
			aesKey:  key,
			aesVersions: [][]byte{
				append([]byte(nil), key...),
			},
		}, nil

	default:
		return nil, fmt.Errorf("hsm: software generate: unsupported key type %q", keyType)
	}
}

// GetPublicKey returns the public key bytes for the named key.
// Returns an error for symmetric (AES-256-GCM) keys.
func (m *SoftwareKeyManager) GetPublicKey(_ context.Context, name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k, err := m.lookup(name)
	if err != nil {
		return nil, err
	}
	if k.keyType == KeyTypeAES256 {
		return nil, fmt.Errorf("hsm: software get public key: key %q is symmetric (no public key)", name)
	}
	out := make([]byte, len(k.publicKey))
	copy(out, k.publicKey)
	return out, nil
}

// Sign signs data using the named key and returns the raw signature bytes.
func (m *SoftwareKeyManager) Sign(_ context.Context, name string, data []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k, err := m.lookup(name)
	if err != nil {
		return nil, err
	}

	switch k.keyType {
	case KeyTypeEd25519:
		if len(k.privateKey) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("hsm: software sign: ed25519 private key corrupted for %q", name)
		}
		sig := ed25519.Sign(ed25519.PrivateKey(k.privateKey), data)
		return sig, nil

	case KeyTypeECDSAP256:
		if k.ecPriv == nil {
			return nil, fmt.Errorf("hsm: software sign: ecdsa private key not loaded for %q", name)
		}
		hash := sha256.Sum256(data)
		r, s, err := ecdsa.Sign(rand.Reader, k.ecPriv, hash[:])
		if err != nil {
			return nil, fmt.Errorf("hsm: software sign ecdsa: %w", err)
		}
		raw, err := json.Marshal(ecdsaSig{R: r.Bytes(), S: s.Bytes()})
		if err != nil {
			return nil, fmt.Errorf("hsm: software sign ecdsa marshal: %w", err)
		}
		return raw, nil

	case KeyTypeDilithium3:
		marker := [4]byte{0xD1, 0x1B, 0x10, 0x4D}
		if len(k.privateKey) < 4 ||
			k.privateKey[0] != marker[0] || k.privateKey[1] != marker[1] ||
			k.privateKey[2] != marker[2] || k.privateKey[3] != marker[3] {
			return nil, fmt.Errorf("hsm: software sign dilithium3: private key marker missing for %q", name)
		}
		edPriv := ed25519.PrivateKey(k.privateKey[4 : 4+ed25519.PrivateKeySize])
		edSig := ed25519.Sign(edPriv, data)
		sig := make([]byte, 3293)
		copy(sig[0:4], marker[:])
		copy(sig[4:4+len(edSig)], edSig)
		return sig, nil

	case KeyTypeAES256:
		return nil, fmt.Errorf("hsm: software sign: key %q is symmetric and cannot be used for signing", name)

	default:
		return nil, fmt.Errorf("hsm: software sign: unsupported key type %q", k.keyType)
	}
}

// Verify verifies signature over data using the named key's public key.
func (m *SoftwareKeyManager) Verify(_ context.Context, name string, data, signature []byte) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k, err := m.lookup(name)
	if err != nil {
		return false, err
	}

	switch k.keyType {
	case KeyTypeEd25519:
		if len(k.publicKey) != ed25519.PublicKeySize {
			return false, fmt.Errorf("hsm: software verify: ed25519 public key corrupted for %q", name)
		}
		return ed25519.Verify(ed25519.PublicKey(k.publicKey), data, signature), nil

	case KeyTypeECDSAP256:
		if k.ecPriv == nil {
			return false, fmt.Errorf("hsm: software verify: ecdsa key not loaded for %q", name)
		}
		var s ecdsaSig
		if err := json.Unmarshal(signature, &s); err != nil {
			return false, fmt.Errorf("hsm: software verify ecdsa: invalid signature format: %w", err)
		}
		hash := sha256.Sum256(data)
		r := new(big.Int).SetBytes(s.R)
		sv := new(big.Int).SetBytes(s.S)
		return ecdsa.Verify(&k.ecPriv.PublicKey, hash[:], r, sv), nil

	case KeyTypeDilithium3:
		marker := [4]byte{0xD1, 0x1B, 0x10, 0x4D}
		if len(k.publicKey) < 4 || k.publicKey[0] != marker[0] {
			return false, fmt.Errorf("hsm: software verify dilithium3: public key marker missing for %q", name)
		}
		if len(signature) != 3293 {
			return false, errors.New("hsm: software verify dilithium3: invalid signature length")
		}
		edPub := ed25519.PublicKey(k.publicKey[4 : 4+ed25519.PublicKeySize])
		edSig := signature[4 : 4+ed25519.SignatureSize]
		return ed25519.Verify(edPub, data, edSig), nil

	case KeyTypeAES256:
		return false, fmt.Errorf("hsm: software verify: key %q is symmetric and cannot be used for verification", name)

	default:
		return false, fmt.Errorf("hsm: software verify: unsupported key type %q", k.keyType)
	}
}

// RotateKey generates a new key version for the named key, retaining the
// previous version so existing ciphertext can still be decrypted.
//
// For asymmetric keys, the previous private key is discarded and a fresh key
// pair is generated (callers must re-issue signatures with the new key).
// For AES keys, the old key is moved to aesVersions[1] and the new key
// becomes aesVersions[0]; DecryptData tries versions in order.
func (m *SoftwareKeyManager) RotateKey(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	k, err := m.lookup(name)
	if err != nil {
		return err
	}

	if k.keyType == KeyTypeAES256 {
		newAES := make([]byte, 32)
		if _, err := rand.Read(newAES); err != nil {
			return fmt.Errorf("hsm: software rotate: aes key generation: %w", err)
		}
		// Prepend new version, keep all older versions for decryption.
		k.aesVersions = append([][]byte{newAES}, k.aesVersions...)
		k.aesKey = newAES
		return nil
	}

	// Asymmetric key: generate a fresh key pair.
	newKey, err := generateSoftwareKey(k.keyType)
	if err != nil {
		return fmt.Errorf("hsm: software rotate: %w", err)
	}
	m.keys[name] = newKey
	return nil
}

// ListKeys returns the names of all keys held by this manager.
func (m *SoftwareKeyManager) ListKeys(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.keys))
	for name := range m.keys {
		names = append(names, name)
	}
	return names, nil
}

// EncryptData encrypts plaintext using AES-256-GCM with the named key.
// The key must be of type KeyTypeAES256.
// The returned ciphertext format is: version_index(1) || nonce(12) || ciphertext+tag.
func (m *SoftwareKeyManager) EncryptData(_ context.Context, name string, plaintext []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k, err := m.lookup(name)
	if err != nil {
		return nil, err
	}
	if k.keyType != KeyTypeAES256 {
		return nil, fmt.Errorf("hsm: software encrypt: key %q is not an AES-256-GCM key", name)
	}

	ct, err := aesGCMEncrypt(k.aesKey, plaintext)
	if err != nil {
		return nil, fmt.Errorf("hsm: software encrypt: %w", err)
	}
	// Prepend version byte 0 (current version).
	return append([]byte{0}, ct...), nil
}

// DecryptData decrypts ciphertext produced by EncryptData.
// Tries all retained key versions to support data encrypted before rotation.
func (m *SoftwareKeyManager) DecryptData(_ context.Context, name string, ciphertext []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k, err := m.lookup(name)
	if err != nil {
		return nil, err
	}
	if k.keyType != KeyTypeAES256 {
		return nil, fmt.Errorf("hsm: software decrypt: key %q is not an AES-256-GCM key", name)
	}
	if len(ciphertext) < 1 {
		return nil, errors.New("hsm: software decrypt: ciphertext too short")
	}

	// Try all versions: current first, then older.
	blob := ciphertext[1:] // strip version byte
	for i, ver := range k.aesVersions {
		pt, err := aesGCMDecrypt(ver, blob)
		if err == nil {
			return pt, nil
		}
		_ = i
	}
	return nil, fmt.Errorf("hsm: software decrypt: decryption failed for key %q (tried %d version(s))", name, len(k.aesVersions))
}

// lookup returns the named key or an error. Must be called with the lock held.
func (m *SoftwareKeyManager) lookup(name string) (*softwareKey, error) {
	k, ok := m.keys[name]
	if !ok {
		return nil, fmt.Errorf("hsm: software: key %q not found", name)
	}
	return k, nil
}

// aesGCMEncrypt encrypts plaintext with AES-256-GCM using a random nonce.
// Output format: nonce(12) || ciphertext+tag.
func aesGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// aesGCMDecrypt decrypts AES-256-GCM ciphertext in nonce(12)||ct+tag format.
func aesGCMDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, data, nil)
}
