# INDIS — Credential Service

> Credential issuance, verification, revocation, and selective disclosure

## Quick Start

```bash
cd services/credential
go run ./cmd/server
```

## Structure

```
services/credential/
├── cmd/server/main.go       # Entrypoint
├── internal/
│   ├── handler/              # gRPC/HTTP handlers
│   ├── service/              # Business logic
│   ├── repository/           # Data access
│   └── config/               # Configuration
├── go.mod
├── Dockerfile
└── README.md
```
