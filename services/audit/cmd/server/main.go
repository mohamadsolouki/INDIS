// INDIS — Audit logging — append-only, cryptographically signed, 10-year retention
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Printf("Starting INDIS audit service...")

	// TODO: Load configuration
	// TODO: Initialize dependencies (DB, cache, blockchain adapter)
	// TODO: Register gRPC handlers
	// TODO: Start gRPC/HTTP server

	fmt.Printf("INDIS audit service is ready\n")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down INDIS audit service...")
}
