// Package main is the entry point for the Audit Log Hyperledger Fabric chaincode.
//
// Deploy this chaincode to the "audit-log-channel" Fabric channel.
// Endorsement policy: 2-of-3 NIA peers required for LogVerificationEvent.
// Read-only evaluations (GetEvent, GetEventCount, GetRecentEvents) require no
// endorsement policy.
package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&AuditLogContract{})
	if err != nil {
		log.Fatalf("failed to create audit log chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Fatalf("failed to start audit log chaincode: %v", err)
	}
}
