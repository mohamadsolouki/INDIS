// Package main is the entry point for the Credential Anchor Hyperledger Fabric chaincode.
//
// Deploy this chaincode to the "credential-anchor-channel" Fabric channel.
// Endorsement policy: 3-of-5 NIA peers required for state-mutating transactions
// (AnchorCredential, RevokeCredential).
package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&CredentialAnchorContract{})
	if err != nil {
		log.Fatalf("failed to create credential anchor chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Fatalf("failed to start credential anchor chaincode: %v", err)
	}
}
