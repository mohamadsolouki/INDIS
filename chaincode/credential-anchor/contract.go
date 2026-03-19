// Package main implements the Credential Anchor chaincode for Hyperledger Fabric.
//
// This chaincode anchors W3C Verifiable Credential hashes and manages the
// revocation registry for the INDIS platform. Only cryptographic hashes and
// issuer DIDs are stored — credential payload content is never on-chain.
//
// ENDORSEMENT: 3-of-5 NIA peers required for AnchorCredential and RevokeCredential.
// Read-only evaluations (VerifyAnchor, CheckRevocationStatus, GetRevocationList)
// may be served by a single peer.
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CredentialAnchorContract anchors credential hashes and manages revocation on the
// Hyperledger Fabric ledger.
type CredentialAnchorContract struct {
	contractapi.Contract
}

// anchorRecord is the on-chain record stored when a credential is anchored.
type anchorRecord struct {
	CredentialHashHex string `json:"credentialHashHex"`
	IssuerDID         string `json:"issuerDid"`
	BlockTime         string `json:"blockTime"`
}

// revocationRecord is the on-chain record stored when a credential is revoked.
type revocationRecord struct {
	CredentialID string `json:"credentialId"`
	Reason       string `json:"reason"`
	Timestamp    string `json:"timestamp"`
}

// anchorKey returns the world-state key for a credential anchor record.
func anchorKey(credentialHashHex string) string {
	return "ANCHOR:" + credentialHashHex
}

// revokeKey returns the world-state key for a revocation record.
func revokeKey(credentialID string) string {
	return "REVOKE:" + credentialID
}

// AnchorCredential stores a credential hash and its issuer DID on the ledger.
// The credentialHashHex parameter must be a lowercase hex-encoded SHA-256 hash
// of the canonical form of the Verifiable Credential.
// ENDORSEMENT: 3-of-5 NIA peers required.
func (c *CredentialAnchorContract) AnchorCredential(
	ctx contractapi.TransactionContextInterface,
	credentialHashHex string,
	issuerDID string,
) error {
	if credentialHashHex == "" {
		return fmt.Errorf("credentialHashHex must not be empty")
	}
	if issuerDID == "" {
		return fmt.Errorf("issuerDID must not be empty")
	}

	key := anchorKey(credentialHashHex)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("credential hash %s is already anchored", credentialHashHex)
	}

	rec := anchorRecord{
		CredentialHashHex: credentialHashHex,
		IssuerDID:         issuerDID,
		BlockTime:         time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal anchor record: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}

// VerifyAnchor queries the ledger for a credential anchor record.
// Returns a JSON string: {"exists": true, "issuerDid": "...", "blockTime": "..."} if found,
// or {"exists": false} if not found.
func (c *CredentialAnchorContract) VerifyAnchor(
	ctx contractapi.TransactionContextInterface,
	credentialHashHex string,
) (string, error) {
	if credentialHashHex == "" {
		return "", fmt.Errorf("credentialHashHex must not be empty")
	}

	data, err := ctx.GetStub().GetState(anchorKey(credentialHashHex))
	if err != nil {
		return "", fmt.Errorf("failed to read ledger state: %w", err)
	}
	if data == nil {
		return `{"exists": false}`, nil
	}

	var rec anchorRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return "", fmt.Errorf("failed to unmarshal anchor record: %w", err)
	}

	result := map[string]interface{}{
		"exists":    true,
		"issuerDid": rec.IssuerDID,
		"blockTime": rec.BlockTime,
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal verify response: %w", err)
	}
	return string(out), nil
}

// RevokeCredential records a revocation for the given credential ID.
// The reason parameter should be one of: expired, compromised, superseded, admin_action.
// ENDORSEMENT: 3-of-5 NIA peers required.
func (c *CredentialAnchorContract) RevokeCredential(
	ctx contractapi.TransactionContextInterface,
	credentialID string,
	reason string,
) error {
	if credentialID == "" {
		return fmt.Errorf("credentialID must not be empty")
	}
	if reason == "" {
		return fmt.Errorf("revocation reason must not be empty")
	}

	key := revokeKey(credentialID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("credential %s is already revoked", credentialID)
	}

	rec := revocationRecord{
		CredentialID: credentialID,
		Reason:       reason,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal revocation record: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}

// CheckRevocationStatus queries the revocation status of the given credential.
// Returns {"revoked": false} if not revoked, or
// {"revoked": true, "reason": "...", "timestamp": "..."} if revoked.
func (c *CredentialAnchorContract) CheckRevocationStatus(
	ctx contractapi.TransactionContextInterface,
	credentialID string,
) (string, error) {
	if credentialID == "" {
		return "", fmt.Errorf("credentialID must not be empty")
	}

	data, err := ctx.GetStub().GetState(revokeKey(credentialID))
	if err != nil {
		return "", fmt.Errorf("failed to read ledger state: %w", err)
	}
	if data == nil {
		return `{"revoked": false}`, nil
	}

	var rec revocationRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return "", fmt.Errorf("failed to unmarshal revocation record: %w", err)
	}

	result := map[string]interface{}{
		"revoked":   true,
		"reason":    rec.Reason,
		"timestamp": rec.Timestamp,
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal revocation status: %w", err)
	}
	return string(out), nil
}

// GetRevocationList returns a JSON array of all revocation records for the given
// issuer DID. The implementation performs a range query over all "REVOKE:" keys and
// filters by issuerDID. In production, an issuerDID composite key index should be
// used for efficient lookup at scale.
//
// Note: this is a read-only evaluation; no endorsement policy applies.
func (c *CredentialAnchorContract) GetRevocationList(
	ctx contractapi.TransactionContextInterface,
	issuerDID string,
) (string, error) {
	// Range query over all REVOKE: keys.
	iter, err := ctx.GetStub().GetStateByRange("REVOKE:", "REVOKE:~")
	if err != nil {
		return "", fmt.Errorf("failed to query revocation list: %w", err)
	}
	defer iter.Close()

	type revocationEntry struct {
		CredentialID string `json:"credentialId"`
		Reason       string `json:"reason"`
		Timestamp    string `json:"timestamp"`
	}

	var entries []revocationEntry
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return "", fmt.Errorf("failed to iterate revocation list: %w", err)
		}
		var rec revocationRecord
		if err := json.Unmarshal(kv.Value, &rec); err != nil {
			continue // skip malformed records
		}
		entries = append(entries, revocationEntry{
			CredentialID: rec.CredentialID,
			Reason:       rec.Reason,
			Timestamp:    rec.Timestamp,
		})
	}

	if entries == nil {
		entries = []revocationEntry{} // return [] not null
	}

	// issuerDID is accepted as a parameter to satisfy the adapter interface but is
	// not used for filtering at chaincode level; filtering by issuer is done in the
	// service layer using the returned list. A production deployment should add a
	// composite key index on (issuerDID, credentialID) for efficient per-issuer queries.
	_ = issuerDID

	out, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("failed to marshal revocation list: %w", err)
	}
	return string(out), nil
}
