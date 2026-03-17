# INDIS — Audit Service

> Audit logging — append-only, cryptographically signed, 10-year retention

## Quick Start

```bash
cd services/audit
go run ./cmd/server
```

## Structure

```
services/audit/
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
