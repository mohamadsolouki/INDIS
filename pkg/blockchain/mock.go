// Package blockchain — mock adapter for testing and development.
package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockAdapter is an in-memory implementation of BlockchainAdapter for development and testing.
type MockAdapter struct {
	mu           sync.RWMutex
	dids         map[string]*DIDDocument
	anchors      map[Hash]*AnchorStatus
	revocations  map[string]*RevocationStatus
	blockHeight  uint64
}

// NewMockAdapter creates a new in-memory blockchain adapter.
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		dids:        make(map[string]*DIDDocument),
		anchors:     make(map[Hash]*AnchorStatus),
		revocations: make(map[string]*RevocationStatus),
		blockHeight: 0,
	}
}

func (m *MockAdapter) nextBlock() *TxReceipt {
	m.blockHeight++
	return &TxReceipt{
		TxID:        fmt.Sprintf("tx-%d", m.blockHeight),
		BlockHeight: m.blockHeight,
		Timestamp:   time.Now(),
	}
}

func (m *MockAdapter) RegisterDID(_ context.Context, did string, document DIDDocument) (*TxReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.dids[did]; exists {
		return nil, fmt.Errorf("DID %s already registered", did)
	}
	document.ID = did
	document.Created = time.Now()
	document.Updated = time.Now()
	m.dids[did] = &document
	return m.nextBlock(), nil
}

func (m *MockAdapter) ResolveDID(_ context.Context, did string) (*DIDDocument, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	doc, exists := m.dids[did]
	if !exists {
		return nil, fmt.Errorf("DID %s not found", did)
	}
	return doc, nil
}

func (m *MockAdapter) UpdateDIDDocument(_ context.Context, did string, update DIDDocument) (*TxReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.dids[did]; !exists {
		return nil, fmt.Errorf("DID %s not found", did)
	}
	update.ID = did
	update.Updated = time.Now()
	m.dids[did] = &update
	return m.nextBlock(), nil
}

func (m *MockAdapter) DeactivateDID(_ context.Context, did string) (*TxReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.dids[did]; !exists {
		return nil, fmt.Errorf("DID %s not found", did)
	}
	delete(m.dids, did)
	return m.nextBlock(), nil
}

func (m *MockAdapter) AnchorCredential(_ context.Context, credentialHash Hash, issuerDID string) (*TxReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.anchors[credentialHash] = &AnchorStatus{
		Exists:      true,
		IssuerDID:   issuerDID,
		BlockHeight: m.blockHeight + 1,
		Timestamp:   time.Now(),
	}
	return m.nextBlock(), nil
}

func (m *MockAdapter) VerifyAnchor(_ context.Context, credentialHash Hash) (*AnchorStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, exists := m.anchors[credentialHash]
	if !exists {
		return &AnchorStatus{Exists: false}, nil
	}
	return status, nil
}

func (m *MockAdapter) RevokeCredential(_ context.Context, credentialID string, reason RevocationReason) (*TxReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revocations[credentialID] = &RevocationStatus{
		Revoked:   true,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	return m.nextBlock(), nil
}

func (m *MockAdapter) CheckRevocationStatus(_ context.Context, credentialID string) (*RevocationStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, exists := m.revocations[credentialID]
	if !exists {
		return &RevocationStatus{Revoked: false}, nil
	}
	return status, nil
}

func (m *MockAdapter) GetRevocationList(_ context.Context, issuerDID string) (*RevocationList, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ids []string
	for id, status := range m.revocations {
		if status.Revoked {
			ids = append(ids, id)
		}
	}
	return &RevocationList{
		IssuerDID:   issuerDID,
		RevokedIDs:  ids,
		LastUpdated: time.Now(),
	}, nil
}

func (m *MockAdapter) LogVerificationEvent(_ context.Context, _ AnonymizedVerificationEvent) (*TxReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextBlock(), nil
}

func (m *MockAdapter) GetBlockHeight(_ context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blockHeight, nil
}

func (m *MockAdapter) GetValidatorStatus(_ context.Context) ([]ValidatorStatus, error) {
	return []ValidatorStatus{
		{NodeID: "mock-node-1", Address: "localhost:7051", IsActive: true, LastSeen: time.Now()},
	}, nil
}

func (m *MockAdapter) EstimateTxTime(_ context.Context) (time.Duration, error) {
	return 100 * time.Millisecond, nil
}

// Compile-time interface check.
var _ BlockchainAdapter = (*MockAdapter)(nil)
