// Package main implements the Audit Log chaincode for Hyperledger Fabric.
//
// This chaincode stores anonymized verification events for public auditability.
// Events are append-only: once written, they cannot be modified or deleted.
// Personal data is NEVER stored; events contain only credential type, verifier
// category, boolean result, and timestamp — never subject identity.
//
// This design satisfies INDIS PRD §9.2 (public audit trail) while maintaining
// privacy-by-architecture per §8.1.
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// forbiddenAuditFields is the deny-list of JSON field names that must not appear
// in audit event records. This enforces anonymization at the chaincode boundary.
var forbiddenAuditFields = []string{
	"name", "full_name", "first_name", "last_name",
	"national_id", "national_code", "ssn",
	"address", "phone", "phone_number", "mobile",
	"email", "email_address",
	"subject", "subject_id", "holder",
	"birth_date", "date_of_birth",
}

// AuditLogContract stores anonymized verification events for public auditability.
// All stored events are immutable once written (append-only semantics enforced
// by rejecting PutState on an existing key).
type AuditLogContract struct {
	contractapi.Contract
}

// auditEvent is the on-chain schema for a verification event.
// Fields mirror AnonymizedVerificationEvent in the adapter interface.
type auditEvent struct {
	EventID          string `json:"event_id"`
	CredentialType   string `json:"credential_type"`
	VerifierCategory string `json:"verifier_category"`
	Result           bool   `json:"result"`
	Timestamp        string `json:"timestamp"`
}

// eventCountKey is the world-state key for the monotonic event counter.
const eventCountKey = "META:event_count"

// eventKey returns the world-state key for an audit event record.
func eventKey(eventID string) string {
	return "EVENT:" + eventID
}

// containsPersonalData returns true if rawJSON contains any forbidden field name.
func containsPersonalData(rawJSON string) bool {
	lower := strings.ToLower(rawJSON)
	for _, field := range forbiddenAuditFields {
		if strings.Contains(lower, `"`+field+`"`) {
			return true
		}
	}
	return false
}

// LogVerificationEvent appends an anonymized verification event to the ledger.
// The eventJSON must conform to the auditEvent schema. Any event containing
// personal data field names is rejected.
// Returns an error if the event already exists (enforcing append-only semantics).
func (c *AuditLogContract) LogVerificationEvent(
	ctx contractapi.TransactionContextInterface,
	eventJSON string,
) error {
	if containsPersonalData(eventJSON) {
		return fmt.Errorf("rejected: audit event contains prohibited personal data fields")
	}

	var evt auditEvent
	if err := json.Unmarshal([]byte(eventJSON), &evt); err != nil {
		return fmt.Errorf("invalid event JSON: %w", err)
	}
	if evt.EventID == "" {
		return fmt.Errorf("event must contain a non-empty 'event_id' field")
	}
	if evt.CredentialType == "" {
		return fmt.Errorf("event must contain a non-empty 'credential_type' field")
	}
	if evt.Timestamp == "" {
		evt.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	key := eventKey(evt.EventID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read ledger state: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("event %s already exists; audit log is append-only", evt.EventID)
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	if err := ctx.GetStub().PutState(key, data); err != nil {
		return fmt.Errorf("failed to write event to ledger: %w", err)
	}

	// Increment the event counter.
	return c.incrementEventCount(ctx)
}

// GetEvent retrieves a single audit event by its event ID.
// Returns the raw JSON string of the event record.
func (c *AuditLogContract) GetEvent(
	ctx contractapi.TransactionContextInterface,
	eventID string,
) (string, error) {
	if eventID == "" {
		return "", fmt.Errorf("eventID must not be empty")
	}

	data, err := ctx.GetStub().GetState(eventKey(eventID))
	if err != nil {
		return "", fmt.Errorf("failed to read ledger state: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("event %s not found", eventID)
	}
	return string(data), nil
}

// GetEventCount returns the total number of verification events recorded on the
// ledger. This is maintained as a separate counter key for O(1) lookup.
func (c *AuditLogContract) GetEventCount(ctx contractapi.TransactionContextInterface) (int, error) {
	data, err := ctx.GetStub().GetState(eventCountKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read event count: %w", err)
	}
	if data == nil {
		return 0, nil
	}
	count, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("malformed event count value: %w", err)
	}
	return count, nil
}

// GetRecentEvents returns a JSON array of the most recent N audit events.
// Events are retrieved via a range query over the "EVENT:" key namespace and
// the last `limit` entries are returned. In a production deployment, a
// composite-key index by timestamp should be used for efficient time-range queries.
//
// limit must be > 0 and <= 1000.
func (c *AuditLogContract) GetRecentEvents(
	ctx contractapi.TransactionContextInterface,
	limit int,
) (string, error) {
	if limit <= 0 {
		return "", fmt.Errorf("limit must be a positive integer")
	}
	if limit > 1000 {
		return "", fmt.Errorf("limit must not exceed 1000 to prevent excessive read sets")
	}

	iter, err := ctx.GetStub().GetStateByRange("EVENT:", "EVENT:~")
	if err != nil {
		return "", fmt.Errorf("failed to query event range: %w", err)
	}
	defer iter.Close()

	// Collect all events then take the last `limit` entries.
	// For large ledgers a reverse iterator or composite key by timestamp should be used.
	var events []auditEvent
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return "", fmt.Errorf("failed to iterate events: %w", err)
		}
		var evt auditEvent
		if err := json.Unmarshal(kv.Value, &evt); err != nil {
			continue // skip malformed entries
		}
		events = append(events, evt)
	}

	// Return the last `limit` events.
	if len(events) > limit {
		events = events[len(events)-limit:]
	}
	if events == nil {
		events = []auditEvent{} // return [] not null
	}

	out, err := json.Marshal(events)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event list: %w", err)
	}
	return string(out), nil
}

// incrementEventCount atomically increments the event counter stored at eventCountKey.
func (c *AuditLogContract) incrementEventCount(ctx contractapi.TransactionContextInterface) error {
	data, err := ctx.GetStub().GetState(eventCountKey)
	if err != nil {
		return fmt.Errorf("failed to read event count: %w", err)
	}
	count := 0
	if data != nil {
		count, err = strconv.Atoi(string(data))
		if err != nil {
			return fmt.Errorf("malformed event count value: %w", err)
		}
	}
	count++
	return ctx.GetStub().PutState(eventCountKey, []byte(strconv.Itoa(count)))
}
