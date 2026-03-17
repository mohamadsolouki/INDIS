# INDIS — Justice Service

> Transitional justice — anonymous testimony, conditional amnesty workflows

## Quick Start

```bash
cd services/justice
go run ./cmd/server
```

## Structure

```
services/justice/
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
