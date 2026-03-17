# INDIS — Biometric Service

> Biometric management — capture, deduplication, template storage

## Quick Start

```bash
cd services/biometric
go run ./cmd/server
```

## Structure

```
services/biometric/
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
