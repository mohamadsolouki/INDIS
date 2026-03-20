// Package service implements business logic for the audit service.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	"github.com/IranProsperityProject/INDIS/pkg/blockchain"
	"github.com/IranProsperityProject/INDIS/services/audit/internal/repository"
)

// AuditService implements tamper-evident append-only audit logging.
type AuditService struct {
	repo  *repository.Repository
	chain blockchain.BlockchainAdapter
}

// New creates an AuditService with an optional blockchain anchor backend.
// Pass a non-nil chain to enable on-chain tamper-evidence anchoring.
func New(repo *repository.Repository, chain blockchain.BlockchainAdapter) *AuditService {
	return &AuditService{repo: repo, chain: chain}
}

func generateEventID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "evt_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// entryHash computes SHA-256(prevHash + category + action + actorDID + subjectDID + resourceID + ts).
func entryHash(prevHash, action, actorDID, subjectDID, resourceID string, ts time.Time) string {
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write([]byte(action))
	h.Write([]byte(actorDID))
	h.Write([]byte(subjectDID))
	h.Write([]byte(resourceID))
	h.Write([]byte(ts.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))
}

// AppendEvent writes a new audit event and returns its ID, prev_hash, and timestamp.
func (s *AuditService) AppendEvent(ctx context.Context, req *auditv1.AppendEventRequest) (eventID, prevHash, timestamp string, err error) {
	eventID, err = generateEventID()
	if err != nil {
		return "", "", "", fmt.Errorf("service: generate event id: %w", err)
	}

	prevHash, err = s.repo.LatestHash(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("service: fetch latest hash: %w", err)
	}

	now := time.Now().UTC()
	hash := entryHash(prevHash, req.GetAction(), req.GetActorDid(), req.GetSubjectDid(), req.GetResourceId(), now)

	meta := req.GetMetadata()
	if len(meta) == 0 {
		meta, _ = json.Marshal(map[string]string{})
	}

	rec := repository.EventRecord{
		EventID:    eventID,
		Category:   int32(req.GetCategory()),
		Action:     req.GetAction(),
		ActorDID:   req.GetActorDid(),
		SubjectDID: req.GetSubjectDid(),
		ResourceID: req.GetResourceId(),
		ServiceID:  req.GetServiceId(),
		Metadata:   meta,
		PrevHash:   prevHash,
		EntryHash:  hash,
		Timestamp:  now,
	}
	if err = s.repo.Append(ctx, rec); err != nil {
		return "", "", "", err
	}

	// Best-effort on-chain anchor — mirrors the DB hash to the immutable audit-log chaincode.
	// A failure here does not roll back the DB write; a background reconciler can retry.
	if s.chain != nil {
		if _, anchorErr := s.chain.AnchorAuditEvent(ctx, eventID, hash); anchorErr != nil {
			// Log but do not surface to caller — consistent with other best-effort anchors.
			_ = fmt.Errorf("audit: blockchain anchor failed for event %s: %w", eventID, anchorErr)
		}
	}

	return eventID, prevHash, now.Format(time.RFC3339), nil
}

// QueryEvents returns audit events matching the given filter.
func (s *AuditService) QueryEvents(ctx context.Context, req *auditv1.QueryEventsRequest) ([]repository.EventRecord, string, error) {
	var from, to time.Time
	if v := req.GetFromTime(); v != "" {
		from, _ = time.Parse(time.RFC3339, v)
	}
	if v := req.GetToTime(); v != "" {
		to, _ = time.Parse(time.RFC3339, v)
	}
	recs, err := s.repo.Query(ctx,
		req.GetActorDid(), req.GetSubjectDid(), int32(req.GetCategory()),
		from, to, req.GetLimit(), req.GetPageToken(),
	)
	if err != nil {
		return nil, "", fmt.Errorf("service: query events: %w", err)
	}
	nextToken := ""
	if len(recs) > 0 {
		nextToken = recs[len(recs)-1].EventID
	}
	return recs, nextToken, nil
}

// GetEventByID retrieves a single event by ID.
func (s *AuditService) GetEventByID(ctx context.Context, eventID string) (*repository.EventRecord, error) {
	rec, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("service: get event: %w", err)
	}
	return rec, nil
}
