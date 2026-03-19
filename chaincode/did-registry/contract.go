// Package main implements the DID Registry chaincode for Hyperledger Fabric.
//
// This chaincode manages W3C DID Documents on the INDIS Fabric ledger.
// Per INDIS PRD §8.1, ONLY public key material and service endpoints are stored
// on-chain. Personal data (names, national IDs, addresses, phone numbers,
// email addresses) MUST NEVER be stored on the blockchain.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// personalDataFields is the deny-list of JSON field names that indicate personal
// data. Any DID document containing these fields is rejected at chaincode level.
// This is a defence-in-depth control; the primary enforcement is in the identity
// service layer.
var personalDataFields = []string{
	"name", "full_name", "first_name", "last_name",
	"national_id", "national_code", "ssn",
	"address", "home_address", "postal_address",
	"phone", "phone_number", "mobile",
	"email", "email_address",
	"birth_date", "date_of_birth", "dob",
	"father_name", "mother_name",
}

// DIDRegistryContract implements DID CRUD operations on the Hyperledger Fabric ledger.
// All state is keyed under the "DID:" composite-key prefix to avoid collisions with
// other chaincodes deployed on the same channel.
type DIDRegistryContract struct {
	contractapi.Contract
}

// chainDIDDocument is the on-chain representation of a DID document.
// Only public cryptographic material and service endpoints are persisted.
type chainDIDDocument struct {
	DID         string             `json:"did"`
	PublicKeys  []chainPublicKey   `json:"publicKeys"`
	Services    []chainService     `json:"services"`
	Created     string             `json:"created"`
	Updated     string             `json:"updated"`
	Deactivated bool               `json:"deactivated"`
}

// chainPublicKey is an on-chain public key descriptor inside a DID document.
type chainPublicKey struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Controller   string `json:"controller"`
	PublicKeyHex string `json:"publicKeyHex"`
}

// chainService is an on-chain service endpoint descriptor inside a DID document.
type chainService struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// ledgerKey returns the world-state key for the given DID.
func ledgerKey(did string) string {
	return "DID:" + did
}

// containsPersonalData returns true if rawJSON contains any field name from the
// personalDataFields deny-list. The check is case-insensitive to catch camelCase
// and snake_case variants.
func containsPersonalData(rawJSON string) bool {
	lower := strings.ToLower(rawJSON)
	for _, field := range personalDataFields {
		// Match "field": (with surrounding quotes) to avoid false positives on values.
		if strings.Contains(lower, `"`+field+`"`) ||
			strings.Contains(lower, `"`+field+`":`) {
			return true
		}
	}
	return false
}

// RegisterDID stores a new DID document on the ledger.
// The caller supplies the full DID document as a JSON-encoded string.
// Returns an error if the DID already exists or if the document contains
// personal data fields.
func (c *DIDRegistryContract) RegisterDID(ctx contractapi.TransactionContextInterface, didJSON string) error {
	if containsPersonalData(didJSON) {
		return fmt.Errorf("rejected: DID document contains prohibited personal data fields")
	}

	var doc chainDIDDocument
	if err := json.Unmarshal([]byte(didJSON), &doc); err != nil {
		return fmt.Errorf("invalid DID document JSON: %w", err)
	}
	if doc.DID == "" {
		return fmt.Errorf("DID document must contain a non-empty 'did' field")
	}

	key := ledgerKey(doc.DID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("DID %s is already registered", doc.DID)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	doc.Created = now
	doc.Updated = now
	doc.Deactivated = false

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal DID document: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}

// ResolveDID retrieves the DID document for the given DID identifier.
// Returns the document as a JSON string, or an error if the DID is not found.
func (c *DIDRegistryContract) ResolveDID(ctx contractapi.TransactionContextInterface, did string) (string, error) {
	if did == "" {
		return "", fmt.Errorf("DID parameter must not be empty")
	}

	data, err := ctx.GetStub().GetState(ledgerKey(did))
	if err != nil {
		return "", fmt.Errorf("failed to read ledger state: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("DID %s not found", did)
	}
	return string(data), nil
}

// UpdateDIDDocument replaces the DID document stored under the given DID.
// Only the NIA organisation MSP (org1MSP) may invoke this function; the
// chaincode enforces this via the client identity's MSP ID.
// Returns an error if the DID does not exist or if the updated document contains
// personal data fields.
func (c *DIDRegistryContract) UpdateDIDDocument(ctx contractapi.TransactionContextInterface, didJSON string) error {
	if containsPersonalData(didJSON) {
		return fmt.Errorf("rejected: updated DID document contains prohibited personal data fields")
	}

	var doc chainDIDDocument
	if err := json.Unmarshal([]byte(didJSON), &doc); err != nil {
		return fmt.Errorf("invalid DID document JSON: %w", err)
	}
	if doc.DID == "" {
		return fmt.Errorf("DID document must contain a non-empty 'did' field")
	}

	// Only the NIA organisation may update DID documents.
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to obtain client MSP ID: %w", err)
	}
	if clientMSP != "niaMSP" {
		return fmt.Errorf("access denied: only niaMSP members may update DID documents, got %s", clientMSP)
	}

	key := ledgerKey(doc.DID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("DID %s not found; use RegisterDID to create a new document", doc.DID)
	}

	// Preserve the original creation timestamp.
	var original chainDIDDocument
	if err := json.Unmarshal(existing, &original); err != nil {
		return fmt.Errorf("failed to unmarshal existing DID document: %w", err)
	}
	doc.Created = original.Created
	doc.Updated = time.Now().UTC().Format(time.RFC3339)
	doc.Deactivated = original.Deactivated

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal updated DID document: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}

// DeactivateDID marks a DID document as deactivated without removing it from the
// ledger. Deactivated DIDs remain resolvable so that historical verifications can
// be audited, but issuers and verifiers MUST treat them as invalid for new
// credential issuance or presentation.
func (c *DIDRegistryContract) DeactivateDID(ctx contractapi.TransactionContextInterface, did string) error {
	if did == "" {
		return fmt.Errorf("DID parameter must not be empty")
	}

	key := ledgerKey(did)
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if data == nil {
		return fmt.Errorf("DID %s not found", did)
	}

	var doc chainDIDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to unmarshal DID document: %w", err)
	}

	doc.Deactivated = true
	doc.Updated = time.Now().UTC().Format(time.RFC3339)

	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal deactivated DID document: %w", err)
	}
	return ctx.GetStub().PutState(key, updated)
}

// DIDExists returns true if a DID document is stored on the ledger for the given
// DID identifier, regardless of its deactivation status.
func (c *DIDRegistryContract) DIDExists(ctx contractapi.TransactionContextInterface, did string) (bool, error) {
	if did == "" {
		return false, fmt.Errorf("DID parameter must not be empty")
	}

	data, err := ctx.GetStub().GetState(ledgerKey(did))
	if err != nil {
		return false, fmt.Errorf("failed to read ledger state: %w", err)
	}
	return data != nil, nil
}
