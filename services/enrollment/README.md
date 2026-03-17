# INDIS — Enrollment Service

> Enrollment processing — standard, enhanced, and social attestation pathways

## Quick Start

```bash
cd services/enrollment
go run ./cmd/server
```

## Structure

```
services/enrollment/
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
