// Package crypto provides shared cryptographic utilities for INDIS services.
//
// Supported standards (PRD §4.3):
//   - Ed25519 / ECDSA P-256 (digital signatures)
//   - AES-256-GCM (data at rest)
//   - CRYSTALS-Dilithium (post-quantum, long-term credentials)
//
// All cryptographic libraries used MUST be audited open-source.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
)

// KeyType identifies the algorithm of a key pair.
type KeyType string

const (
	// KeyTypeEd25519 is an Ed25519 signing key (RFC 8037).
	KeyTypeEd25519 KeyType = "Ed25519"
	// KeyTypeECDSAP256 is an ECDSA key on the P-256 curve (FIPS 186-4).
	KeyTypeECDSAP256 KeyType = "EcdsaSecp256r1"
)

// KeyPair holds a public/private key pair and its algorithm type.
type KeyPair struct {
	Type       KeyType
	PublicKey  []byte
	PrivateKey []byte
}

// Signature is a raw signature byte slice.
type Signature []byte

// ECDSASignature holds the two integers that make up an ECDSA signature.
type ECDSASignature struct {
	R, S *big.Int
}

// GenerateEd25519KeyPair generates a new Ed25519 key pair.
// The full 64-byte private key (seed + public key) is stored in PrivateKey.
// Ref: RFC 8037, https://www.rfc-editor.org/rfc/rfc8037
func GenerateEd25519KeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("crypto: ed25519 key generation: %w", err)
	}
	return &KeyPair{
		Type:       KeyTypeEd25519,
		PublicKey:  []byte(pub),
		PrivateKey: []byte(priv),
	}, nil
}

// SignEd25519 signs message with the given Ed25519 private key.
// message should be the raw payload — not pre-hashed (Ed25519 hashes internally).
// Ref: RFC 8032 §5.1
func SignEd25519(privateKey []byte, message []byte) (Signature, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("crypto: invalid ed25519 private key length")
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privateKey), message)
	return Signature(sig), nil
}

// VerifyEd25519 verifies an Ed25519 signature.
// Returns true only if the signature was produced by the private key matching publicKey.
func VerifyEd25519(publicKey []byte, message []byte, sig Signature) bool {
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(publicKey), message, sig)
}

// GenerateECDSAP256KeyPair generates a new ECDSA P-256 key pair.
// Ref: FIPS 186-4, SEC 2 §2.6
func GenerateECDSAP256KeyPair() (*KeyPair, *ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto: ecdsa p-256 key generation: %w", err)
	}
	pubBytes := elliptic.MarshalCompressed(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)
	privBytes := priv.D.Bytes()
	return &KeyPair{
		Type:       KeyTypeECDSAP256,
		PublicKey:  pubBytes,
		PrivateKey: privBytes,
	}, priv, nil
}

// SignECDSAP256 signs the SHA-256 hash of message with the given ECDSA private key.
// Ref: FIPS 186-4
func SignECDSAP256(priv *ecdsa.PrivateKey, message []byte) (*ECDSASignature, error) {
	hash := sha256.Sum256(message)
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: ecdsa sign: %w", err)
	}
	return &ECDSASignature{R: r, S: s}, nil
}

// VerifyECDSAP256 verifies an ECDSA P-256 signature over the SHA-256 hash of message.
func VerifyECDSAP256(pub *ecdsa.PublicKey, message []byte, sig *ECDSASignature) bool {
	hash := sha256.Sum256(message)
	return ecdsa.Verify(pub, hash[:], sig.R, sig.S)
}

// EncryptAES256GCM encrypts plaintext using AES-256-GCM with a random nonce.
// key must be exactly 32 bytes. The returned ciphertext is nonce||ciphertext+tag.
// Ref: NIST SP 800-38D
func EncryptAES256GCM(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("crypto: aes-256-gcm key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce generation: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAES256GCM decrypts AES-256-GCM ciphertext produced by EncryptAES256GCM.
// key must be exactly 32 bytes. ciphertext must be nonce||ciphertext+tag.
func DecryptAES256GCM(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("crypto: aes-256-gcm key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decryption failed: %w", err)
	}
	return plaintext, nil
}

// GenerateRandomKey generates n cryptographically random bytes.
func GenerateRandomKey(n int) ([]byte, error) {
	key := make([]byte, n)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("crypto: random key generation: %w", err)
	}
	return key, nil
}

// HashSHA256 returns the SHA-256 digest of data.
func HashSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
