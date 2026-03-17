# INDIS — Electoral Service

> Electoral module — STARK-ZK voter verification, referendum support

## Quick Start

```bash
cd services/electoral
go run ./cmd/server
```

## Structure

```
services/electoral/
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
