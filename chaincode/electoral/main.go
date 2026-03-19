// Package main is the entry point for the Electoral Hyperledger Fabric chaincode.
//
// Deploy this chaincode to the "electoral-channel" Fabric channel.
// Endorsement policy: 4-of-5 NIA + Ministry of Interior peers required for
// AnchorElectionResult and RegisterElection. AnchorVoteProof requires 3-of-5 peers.
// Read-only evaluations require no endorsement policy.
package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&ElectoralContract{})
	if err != nil {
		log.Fatalf("failed to create electoral chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Fatalf("failed to start electoral chaincode: %v", err)
	}
}
