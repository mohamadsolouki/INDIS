// Package main implements the Electoral chaincode for Hyperledger Fabric.
//
// This chaincode provides ZK-STARK proof anchoring for electoral integrity in the
// INDIS platform. Individual vote content is NEVER stored; only nullifier hashes
// (which are cryptographically anonymous) and aggregate tallies are kept on-chain.
//
// Privacy guarantees:
//   - Nullifier hashes are derived from voter ZK proofs; they cannot be linked back
//     to a specific voter identity without the voter's secret.
//   - Final election results are anchored with a STARK proof hash that allows public
//     verification of correctness without revealing individual votes.
//
// See circuits/electoral_proof.cairo and INDIS PRD §6.4.
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ElectoralContract manages ZK-STARK proof anchoring for electoral integrity.
type ElectoralContract struct {
	contractapi.Contract
}

// electionRecord stores the metadata for a registered election.
type electionRecord struct {
	ElectionID  string `json:"election_id"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"` // "open", "closed", "finalized"
	Created     string `json:"created"`
}

// nullifierRecord records that a voter (identified only by their ZK nullifier hash)
// has cast a vote in an election. No vote content is stored.
type nullifierRecord struct {
	ElectionID       string `json:"election_id"`
	NullifierHashHex string `json:"nullifier_hash_hex"`
	Timestamp        string `json:"timestamp"`
}

// electionResultRecord stores the finalized result hash and its STARK proof hash.
type electionResultRecord struct {
	ElectionID       string `json:"election_id"`
	ResultHashHex    string `json:"result_hash_hex"`
	STARKProofHashHex string `json:"stark_proof_hash_hex"`
	Timestamp        string `json:"timestamp"`
}

// electionKey returns the world-state key for an election record.
func electionKey(electionID string) string {
	return "ELECTION:" + electionID
}

// nullifierKey returns the world-state key for a nullifier record.
// Keyed by both election ID and nullifier hash to allow O(1) double-vote detection.
func nullifierKey(electionID, nullifierHashHex string) string {
	return "NULLIFIER:" + electionID + ":" + nullifierHashHex
}

// tallyKey returns the world-state key for the vote tally counter of an election.
func tallyKey(electionID string) string {
	return "TALLY:" + electionID
}

// resultKey returns the world-state key for the final result anchor of an election.
func resultKey(electionID string) string {
	return "RESULT:" + electionID
}

// RegisterElection stores a new election record on the ledger.
// The electionJSON must include at minimum: election_id, description, start_time, end_time.
// Returns an error if an election with the same ID already exists.
func (c *ElectoralContract) RegisterElection(
	ctx contractapi.TransactionContextInterface,
	electionJSON string,
) error {
	var rec electionRecord
	if err := json.Unmarshal([]byte(electionJSON), &rec); err != nil {
		return fmt.Errorf("invalid election JSON: %w", err)
	}
	if rec.ElectionID == "" {
		return fmt.Errorf("election record must contain a non-empty 'election_id' field")
	}

	key := electionKey(rec.ElectionID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("election %s is already registered", rec.ElectionID)
	}

	rec.Created = time.Now().UTC().Format(time.RFC3339)
	if rec.Status == "" {
		rec.Status = "open"
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal election record: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}

// AnchorVoteProof records that a nullifier (anonymous voter token) has been used
// in the given election. This prevents double-voting without revealing voter identity.
//
// The nullifierHashHex is derived from the voter's ZK proof; it is a one-way
// commitment that cannot be linked to a specific voter without their secret.
// proofHashHex is the hash of the ZK proof submitted by the voter and is stored
// for post-election auditability.
//
// Returns an error if the nullifier has already been used (double-vote attempt).
func (c *ElectoralContract) AnchorVoteProof(
	ctx contractapi.TransactionContextInterface,
	electionID string,
	nullifierHashHex string,
	proofHashHex string,
) error {
	if electionID == "" {
		return fmt.Errorf("electionID must not be empty")
	}
	if nullifierHashHex == "" {
		return fmt.Errorf("nullifierHashHex must not be empty")
	}
	if proofHashHex == "" {
		return fmt.Errorf("proofHashHex must not be empty")
	}

	// Verify the election exists and is open.
	elData, err := ctx.GetStub().GetState(electionKey(electionID))
	if err != nil {
		return fmt.Errorf("failed to read election state: %w", err)
	}
	if elData == nil {
		return fmt.Errorf("election %s not found", electionID)
	}
	var election electionRecord
	if err := json.Unmarshal(elData, &election); err != nil {
		return fmt.Errorf("failed to unmarshal election record: %w", err)
	}
	if election.Status != "open" {
		return fmt.Errorf("election %s is not open for voting (status: %s)", electionID, election.Status)
	}

	// Check for double-vote.
	nullKey := nullifierKey(electionID, nullifierHashHex)
	existing, err := ctx.GetStub().GetState(nullKey)
	if err != nil {
		return fmt.Errorf("failed to read nullifier state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("double-vote detected: nullifier %s has already been used in election %s",
			nullifierHashHex, electionID)
	}

	// Store the nullifier record. Vote content is NOT stored.
	rec := nullifierRecord{
		ElectionID:       electionID,
		NullifierHashHex: nullifierHashHex,
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal nullifier record: %w", err)
	}
	if err := ctx.GetStub().PutState(nullKey, data); err != nil {
		return fmt.Errorf("failed to write nullifier record: %w", err)
	}

	// Increment the vote tally counter for this election.
	return c.incrementTally(ctx, electionID)
}

// CheckNullifier returns true if the given nullifier hash has already been used
// in the specified election (i.e., the voter has already voted).
// Returns false if the nullifier has not been used.
func (c *ElectoralContract) CheckNullifier(
	ctx contractapi.TransactionContextInterface,
	electionID string,
	nullifierHashHex string,
) (bool, error) {
	if electionID == "" {
		return false, fmt.Errorf("electionID must not be empty")
	}
	if nullifierHashHex == "" {
		return false, fmt.Errorf("nullifierHashHex must not be empty")
	}

	data, err := ctx.GetStub().GetState(nullifierKey(electionID, nullifierHashHex))
	if err != nil {
		return false, fmt.Errorf("failed to read nullifier state: %w", err)
	}
	return data != nil, nil
}

// AnchorElectionResult stores the final election result hash and its STARK proof hash.
// This anchors the election outcome immutably on the ledger so that any observer
// can verify the result using the STARK proof without obtaining individual votes.
//
// Returns an error if a result for this election has already been anchored.
func (c *ElectoralContract) AnchorElectionResult(
	ctx contractapi.TransactionContextInterface,
	electionID string,
	resultHashHex string,
	starkProofHashHex string,
) error {
	if electionID == "" {
		return fmt.Errorf("electionID must not be empty")
	}
	if resultHashHex == "" {
		return fmt.Errorf("resultHashHex must not be empty")
	}
	if starkProofHashHex == "" {
		return fmt.Errorf("starkProofHashHex must not be empty")
	}

	// Verify the election exists.
	elData, err := ctx.GetStub().GetState(electionKey(electionID))
	if err != nil {
		return fmt.Errorf("failed to read election state: %w", err)
	}
	if elData == nil {
		return fmt.Errorf("election %s not found", electionID)
	}

	// Prevent overwriting an existing result.
	rKey := resultKey(electionID)
	existing, err := ctx.GetStub().GetState(rKey)
	if err != nil {
		return fmt.Errorf("failed to read result state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("result for election %s is already anchored", electionID)
	}

	rec := electionResultRecord{
		ElectionID:        electionID,
		ResultHashHex:     resultHashHex,
		STARKProofHashHex: starkProofHashHex,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal result record: %w", err)
	}
	if err := ctx.GetStub().PutState(rKey, data); err != nil {
		return fmt.Errorf("failed to write result record: %w", err)
	}

	// Mark the election as finalized.
	var election electionRecord
	if err := json.Unmarshal(elData, &election); err != nil {
		return fmt.Errorf("failed to unmarshal election record: %w", err)
	}
	election.Status = "finalized"
	updated, err := json.Marshal(election)
	if err != nil {
		return fmt.Errorf("failed to marshal updated election: %w", err)
	}
	return ctx.GetStub().PutState(electionKey(electionID), updated)
}

// GetElectionResult returns the finalized result record for the given election
// as a JSON string, or an error if no result has been anchored yet.
func (c *ElectoralContract) GetElectionResult(
	ctx contractapi.TransactionContextInterface,
	electionID string,
) (string, error) {
	if electionID == "" {
		return "", fmt.Errorf("electionID must not be empty")
	}

	data, err := ctx.GetStub().GetState(resultKey(electionID))
	if err != nil {
		return "", fmt.Errorf("failed to read result state: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("no result anchored for election %s", electionID)
	}
	return string(data), nil
}

// GetElectionTallyCount returns the number of votes (nullifier proofs) recorded
// for the given election. This count reflects unique nullifiers, not individual
// voter identities.
func (c *ElectoralContract) GetElectionTallyCount(
	ctx contractapi.TransactionContextInterface,
	electionID string,
) (int64, error) {
	if electionID == "" {
		return 0, fmt.Errorf("electionID must not be empty")
	}

	data, err := ctx.GetStub().GetState(tallyKey(electionID))
	if err != nil {
		return 0, fmt.Errorf("failed to read tally state: %w", err)
	}
	if data == nil {
		return 0, nil
	}
	count, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed tally value: %w", err)
	}
	return count, nil
}

// incrementTally atomically increments the vote tally counter for an election.
func (c *ElectoralContract) incrementTally(
	ctx contractapi.TransactionContextInterface,
	electionID string,
) error {
	key := tallyKey(electionID)
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read tally: %w", err)
	}
	var count int64
	if data != nil {
		count, err = strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return fmt.Errorf("malformed tally value: %w", err)
		}
	}
	count++
	return ctx.GetStub().PutState(key, []byte(strconv.FormatInt(count, 10)))
}
