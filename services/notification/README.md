# INDIS — Notification Service

> Notification service — SMS, Push, Email for credential expiry and verification alerts

## Quick Start

```bash
cd services/notification
go run ./cmd/server
```

## Structure

```
services/notification/
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
