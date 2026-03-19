// Package main is the entry point for the DID Registry Hyperledger Fabric chaincode.
//
// Deploy this chaincode to the "did-registry-channel" Fabric channel.
// Endorsement policy: 2-of-3 NIA peers required for RegisterDID, UpdateDIDDocument,
// and DeactivateDID. ResolveDID and DIDExists are read-only evaluations.
package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&DIDRegistryContract{})
	if err != nil {
		log.Fatalf("failed to create DID registry chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Fatalf("failed to start DID registry chaincode: %v", err)
	}
}
