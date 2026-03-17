# INDIS — Gateway Service

> API gateway — rate limiting, mTLS, routing, load balancing, WAF

## Quick Start

```bash
cd services/gateway
go run ./cmd/server
```

## Structure

```
services/gateway/
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
