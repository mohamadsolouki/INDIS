// Package hsm provides hardware security module key management for INDIS.
//
// Production deployments are backed by HashiCorp Vault with Transit secrets
// engine and HSM unsealing (FIPS 140-2 Level 3). Development and CI
// deployments use the SoftwareKeyManager ephemeral in-memory backend.
//
// Ref: INDIS PRD §4.3 — FIPS 140-2 Level 3 HSM
package hsm

import "context"

// KeyType identifies the algorithm of a managed key.
type KeyType string

const (
	// KeyTypeEd25519 is an Ed25519 signing key (RFC 8037).
	KeyTypeEd25519 KeyType = "ed25519"
	// KeyTypeECDSAP256 is an ECDSA P-256 signing key (FIPS 186-4).
	KeyTypeECDSAP256 KeyType = "ecdsa-p256"
	// KeyTypeDilithium3 is a CRYSTALS-Dilithium3 post-quantum signing key (NIST FIPS 204).
	KeyTypeDilithium3 KeyType = "dilithium3"
	// KeyTypeAES256 is an AES-256-GCM symmetric encryption key (NIST SP 800-38D).
	KeyTypeAES256 KeyType = "aes256-gcm"
)

// KeyManager abstracts key storage and signing operations across HSM backends.
//
// All methods accept a context.Context so callers can apply deadlines and
// cancellation to remote HSM calls.
//
// Error strings are prefixed with "hsm: " followed by the backend and
// operation name, e.g. "hsm: vault sign: ...".
type KeyManager interface {
	// GenerateKey generates a new key pair (or symmetric key) under the given
	// name. If a key with that name already exists the behaviour is
	// backend-specific (Vault: no-op if the key exists; Software: returns an
	// error).
	GenerateKey(ctx context.Context, name string, keyType KeyType) error

	// GetPublicKey retrieves the public key bytes for the named key. For
	// symmetric keys (KeyTypeAES256) this returns an error.
	GetPublicKey(ctx context.Context, name string) ([]byte, error)

	// Sign signs data using the named key and returns the raw signature bytes.
	Sign(ctx context.Context, name string, data []byte) ([]byte, error)

	// Verify verifies a signature over data using the named key's public key.
	// Returns true if and only if the signature is valid.
	Verify(ctx context.Context, name string, data, signature []byte) (bool, error)

	// RotateKey rotates the named key: generates a new key version and retains
	// the previous version for signature verification and decryption of existing
	// ciphertext.
	RotateKey(ctx context.Context, name string) error

	// ListKeys lists the names of all keys managed by this backend.
	ListKeys(ctx context.Context) ([]string, error)

	// EncryptData encrypts plaintext using AES-256-GCM with the named key.
	// The returned ciphertext is opaque to the caller; it must be passed
	// unchanged to DecryptData.
	EncryptData(ctx context.Context, name string, plaintext []byte) ([]byte, error)

	// DecryptData decrypts ciphertext produced by EncryptData using the named
	// key.
	DecryptData(ctx context.Context, name string, ciphertext []byte) ([]byte, error)
}
