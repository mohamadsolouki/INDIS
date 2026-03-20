// Package handler implements gRPC and HTTP handlers for the identity service.
package handler

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	"github.com/IranProsperityProject/INDIS/pkg/did"
	"github.com/IranProsperityProject/INDIS/services/identity/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IdentityHandler implements identityv1.IdentityServiceServer.
type IdentityHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	svc *service.IdentityService
}

// New creates an IdentityHandler wrapping the given service.
func New(svc *service.IdentityService) *IdentityHandler {
	return &IdentityHandler{svc: svc}
}

// RegisterIdentity creates a new DID for an enrolled citizen.
func (h *IdentityHandler) RegisterIdentity(ctx context.Context, req *identityv1.RegisterIdentityRequest) (*identityv1.RegisterIdentityResponse, error) {
	if req.GetDid() == "" {
		return nil, status.Error(codes.InvalidArgument, "did is required")
	}
	if req.GetDocument() == nil {
		return nil, status.Error(codes.InvalidArgument, "document is required")
	}

	doc := protoToDocument(req.GetDid(), req.GetDocument())
	result, err := h.svc.RegisterIdentity(ctx, req.GetDid(), doc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "register identity: %v", err)
	}
	return &identityv1.RegisterIdentityResponse{
		TxId:        result.TxID,
		BlockHeight: result.BlockHeight,
	}, nil
}

// ResolveIdentity retrieves a DID document.
func (h *IdentityHandler) ResolveIdentity(ctx context.Context, req *identityv1.ResolveIdentityRequest) (*identityv1.ResolveIdentityResponse, error) {
	if req.GetDid() == "" {
		return nil, status.Error(codes.InvalidArgument, "did is required")
	}
	result, err := h.svc.ResolveIdentity(ctx, req.GetDid())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "resolve identity: %v", err)
	}

	// Unmarshal the stored JSON document and map it to the proto representation.
	var doc did.Document
	protoDoc := &identityv1.DIDDocument{
		Id:      result.Record.DID,
		Created: result.Record.CreatedAt.UTC().Format(time.RFC3339),
		Updated: result.Record.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if len(result.Record.Document) > 0 {
		if jsonErr := json.Unmarshal(result.Record.Document, &doc); jsonErr == nil {
			for _, vm := range doc.VerificationMethods {
				// Strip leading 'z' multibase prefix if present; send raw hex.
				keyHex := vm.PublicKeyMultibase
				if len(keyHex) > 0 && keyHex[0] == 'z' {
					keyHex = keyHex[1:]
				}
				keyBytes, _ := hex.DecodeString(keyHex)
				protoDoc.PublicKeys = append(protoDoc.PublicKeys, &identityv1.PublicKey{
					Id:         vm.ID,
					Type:       vm.Type,
					Controller: vm.Controller,
					PublicKey:  keyBytes,
				})
			}
			for _, svc := range doc.Services {
				protoDoc.ServiceEndpoints = append(protoDoc.ServiceEndpoints, &identityv1.ServiceEndpoint{
					Id:       svc.ID,
					Type:     svc.Type,
					Endpoint: svc.ServiceEndpoint,
				})
			}
		}
	}
	return &identityv1.ResolveIdentityResponse{Document: protoDoc}, nil
}

// DeactivateIdentity deactivates a DID.
func (h *IdentityHandler) DeactivateIdentity(ctx context.Context, req *identityv1.DeactivateIdentityRequest) (*identityv1.DeactivateIdentityResponse, error) {
	if req.GetDid() == "" {
		return nil, status.Error(codes.InvalidArgument, "did is required")
	}
	result, err := h.svc.DeactivateIdentity(ctx, req.GetDid(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "deactivate identity: %v", err)
	}
	return &identityv1.DeactivateIdentityResponse{TxId: result.TxID}, nil
}

// protoToDocument converts a proto DIDDocument to the pkg/did Document type.
func protoToDocument(didStr string, p *identityv1.DIDDocument) *did.Document {
	now := time.Now().UTC()
	doc := &did.Document{
		Context:     []string{"https://www.w3.org/ns/did/v1"},
		ID:          did.DID(didStr),
		Created:     now,
		Updated:     now,
		Deactivated: false,
	}
	for _, pk := range p.GetPublicKeys() {
		doc.VerificationMethods = append(doc.VerificationMethods, did.VerificationMethod{
			ID:                 pk.GetId(),
			Type:               pk.GetType(),
			Controller:         pk.GetController(),
			PublicKeyMultibase: "z" + hex.EncodeToString(pk.GetPublicKey()),
		})
	}
	for _, se := range p.GetServiceEndpoints() {
		doc.Services = append(doc.Services, did.Service{
			ID:              se.GetId(),
			Type:            se.GetType(),
			ServiceEndpoint: se.GetEndpoint(),
		})
	}
	return doc
}
