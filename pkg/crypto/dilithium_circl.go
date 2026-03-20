//go:build circl

// Package crypto — dilithium_circl.go
// Real CRYSTALS-Dilithium3 implementation via filippo.io/circl.
//
// This file is compiled only when building with: -tags circl
// It replaces the Ed25519 placeholder in dilithium.go with genuine
// post-quantum signatures backed by filippo.io/circl/sign/dilithium/mode3.
//
// Setup:
//   go get filippo.io/circl
//   go build -tags circl ./...
//
// Cryptographic standard reference: NIST FIPS 204 (August 2024),
// CRYSTALS-Dilithium3 parameter set (security level 3).
// Review required by 2+ maintainers before production key ceremony.
package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	dilithium3 "filippo.io/circl/sign/dilithium/mode3"
)

// GenerateDilithiumKeyPair generates a CRYSTALS-Dilithium3 key pair using
// filippo.io/circl/sign/dilithium/mode3.
//
// Keys must be stored in an HSM (FIPS 140-2 Level 3) in production.
func GenerateDilithiumKeyPair() (*DilithiumKeyPair, error) {
	pub, priv, err := dilithium3.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("crypto: dilithium3 keygen: %w", err)
	}

	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("crypto: dilithium3 marshal public key: %w", err)
	}

	privBytes, err := priv.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("crypto: dilithium3 marshal private key: %w", err)
	}

	return &DilithiumKeyPair{
		PublicKey:  pubBytes,
		PrivateKey: privBytes,
	}, nil
}

// SignDilithium signs a message using a Dilithium3 private key.
//
// privateKey must be Dilithium3PrivateKeySize (4000) bytes in the canonical
// MarshalBinary encoding produced by GenerateDilithiumKeyPair.
func SignDilithium(privateKey, message []byte) ([]byte, error) {
	if len(privateKey) != Dilithium3PrivateKeySize {
		return nil, fmt.Errorf("crypto: dilithium sign: invalid private key length %d (want %d)",
			len(privateKey), Dilithium3PrivateKeySize)
	}

	var priv dilithium3.PrivateKey
	if err := priv.UnmarshalBinary(privateKey); err != nil {
		return nil, fmt.Errorf("crypto: dilithium sign: unmarshal private key: %w", err)
	}

	sig := dilithium3.Sign(&priv, message)
	if len(sig) != Dilithium3SignatureSize {
		return nil, errors.New("crypto: dilithium sign: unexpected signature length from circl")
	}

	return sig, nil
}

// VerifyDilithium verifies a Dilithium3 signature over message using the given
// public key. Returns true if and only if the signature is valid.
func VerifyDilithium(publicKey, message, signature []byte) (bool, error) {
	if len(publicKey) != Dilithium3PublicKeySize {
		return false, fmt.Errorf("crypto: dilithium verify: invalid public key length %d (want %d)",
			len(publicKey), Dilithium3PublicKeySize)
	}
	if len(signature) != Dilithium3SignatureSize {
		return false, fmt.Errorf("crypto: dilithium verify: invalid signature length %d (want %d)",
			len(signature), Dilithium3SignatureSize)
	}

	var pub dilithium3.PublicKey
	if err := pub.UnmarshalBinary(publicKey); err != nil {
		return false, fmt.Errorf("crypto: dilithium verify: unmarshal public key: %w", err)
	}

	return dilithium3.Verify(&pub, message, signature), nil
}
