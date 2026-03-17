# INDIS — Identity Service

> Core identity management — DID generation, resolution, lifecycle

## Quick Start

```bash
cd services/identity
go run ./cmd/server
```

## Structure

```
services/identity/
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
