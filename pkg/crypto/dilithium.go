// Package crypto — dilithium.go
// CRYSTALS-Dilithium post-quantum signatures per NIST FIPS 204.
// Ref: https://pq-crystals.org/dilithium/
//
// SECURITY NOTE: This implementation uses Ed25519 as a Dilithium3 placeholder for development.
// Before production deployment, replace with a FIPS 204-compliant implementation.
// Tracked in: T4.1 Post-quantum migration
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
)

// Dilithium3 key sizes as defined in NIST FIPS 204.
// These constants reflect the real Dilithium3 parameter set; the placeholder
// implementation pads Ed25519 keys to these sizes so that callers depending
// on fixed key lengths behave correctly once the real library is integrated.
const (
	// Dilithium3PublicKeySize is the public key size for CRYSTALS-Dilithium3 (1952 bytes).
	Dilithium3PublicKeySize = 1952
	// Dilithium3PrivateKeySize is the private key size for CRYSTALS-Dilithium3 (4000 bytes).
	Dilithium3PrivateKeySize = 4000
	// Dilithium3SignatureSize is the signature size for CRYSTALS-Dilithium3 (3293 bytes).
	Dilithium3SignatureSize = 3293
)

// KeyTypeDilithium3 is CRYSTALS-Dilithium at security level 3 (NIST PQC standard, FIPS 204).
const KeyTypeDilithium3 KeyType = "Dilithium3"

// DilithiumKeyPair holds a CRYSTALS-Dilithium3 key pair.
//
// PublicKey is 1952 bytes and PrivateKey is 4000 bytes, matching the Dilithium3
// parameter set from NIST FIPS 204. In this placeholder implementation the
// actual Ed25519 key material is embedded at the start of each padded buffer.
type DilithiumKeyPair struct {
	// PublicKey is the 1952-byte Dilithium3 public key.
	PublicKey []byte
	// PrivateKey is the 4000-byte Dilithium3 private key.
	PrivateKey []byte
}

// dilithiumMarker is a fixed 4-byte tag written into placeholder key buffers so
// that VerifyDilithium can detect keys produced by this implementation and
// dispatch to the Ed25519 fallback path.
var dilithiumMarker = [4]byte{0xD1, 0x1B, 0x10, 0x4D} // "DILITH" sentinel

// GenerateDilithiumKeyPair generates a new CRYSTALS-Dilithium3 key pair.
//
// In this placeholder implementation Ed25519 is used as the underlying
// cryptographic primitive. The raw Ed25519 key bytes are embedded at the start
// of Dilithium3-sized buffers. Production deployments MUST replace this with a
// FIPS 204-compliant library such as filippo.io/circl/sign/dilithium.
func GenerateDilithiumKeyPair() (*DilithiumKeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("crypto: dilithium keygen (ed25519 placeholder): %w", err)
	}

	// Build padded public key: marker(4) || ed25519_pub(32) || zeros(1916)
	pubPadded := make([]byte, Dilithium3PublicKeySize)
	copy(pubPadded[0:4], dilithiumMarker[:])
	copy(pubPadded[4:4+ed25519.PublicKeySize], pub)

	// Build padded private key: marker(4) || ed25519_priv(64) || zeros(3932)
	privPadded := make([]byte, Dilithium3PrivateKeySize)
	copy(privPadded[0:4], dilithiumMarker[:])
	copy(privPadded[4:4+ed25519.PrivateKeySize], priv)

	return &DilithiumKeyPair{
		PublicKey:  pubPadded,
		PrivateKey: privPadded,
	}, nil
}

// SignDilithium signs a message using a Dilithium3 private key and returns a
// Dilithium3-sized signature.
//
// In this placeholder implementation the signature is an Ed25519 signature
// (64 bytes) padded to Dilithium3SignatureSize (3293 bytes). The marker bytes
// allow VerifyDilithium to detect and handle the placeholder signature format.
func SignDilithium(privateKey, message []byte) ([]byte, error) {
	if len(privateKey) != Dilithium3PrivateKeySize {
		return nil, errors.New("crypto: dilithium sign: invalid private key length")
	}
	if !hasDilithiumMarker(privateKey) {
		return nil, errors.New("crypto: dilithium sign: private key marker missing (not a placeholder key?)")
	}

	// Extract the embedded Ed25519 private key.
	ed25519Priv := ed25519.PrivateKey(privateKey[4 : 4+ed25519.PrivateKeySize])
	edSig := ed25519.Sign(ed25519Priv, message)

	// Build padded signature: marker(4) || ed25519_sig(64) || zeros(3225)
	sig := make([]byte, Dilithium3SignatureSize)
	copy(sig[0:4], dilithiumMarker[:])
	copy(sig[4:4+len(edSig)], edSig)

	return sig, nil
}

// VerifyDilithium verifies a Dilithium3 signature over message using the given
// public key. Returns true if and only if the signature is valid.
//
// In this placeholder implementation verification is delegated to Ed25519.
func VerifyDilithium(publicKey, message, signature []byte) (bool, error) {
	if len(publicKey) != Dilithium3PublicKeySize {
		return false, errors.New("crypto: dilithium verify: invalid public key length")
	}
	if len(signature) != Dilithium3SignatureSize {
		return false, errors.New("crypto: dilithium verify: invalid signature length")
	}
	if !hasDilithiumMarker(publicKey) || !hasDilithiumMarker(signature) {
		return false, errors.New("crypto: dilithium verify: marker mismatch (key/signature not from placeholder implementation)")
	}

	ed25519Pub := ed25519.PublicKey(publicKey[4 : 4+ed25519.PublicKeySize])
	edSig := signature[4 : 4+ed25519.SignatureSize]

	return ed25519.Verify(ed25519Pub, message, edSig), nil
}

// hasDilithiumMarker returns true if b begins with the placeholder marker bytes.
func hasDilithiumMarker(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	return b[0] == dilithiumMarker[0] &&
		b[1] == dilithiumMarker[1] &&
		b[2] == dilithiumMarker[2] &&
		b[3] == dilithiumMarker[3]
}
