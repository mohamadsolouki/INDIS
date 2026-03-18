# INDIS — Gateway Service

> API gateway — rate limiting, mTLS, routing, load balancing, WAF

## Quick Start

```bash
cd services/gateway
go run ./cmd/server
```

## Configuration

Gateway reads configuration from environment variables.

- `HTTP_PORT` default: `8080`
- `IDENTITY_ADDR` ... `JUSTICE_ADDR` default: `localhost:50051` ... `localhost:50058`
- `RATE_LIMIT_RPS` default: `100`
- `BACKEND_TLS_MODE` default: `plaintext`
	- `plaintext`: no TLS to backends (local compatibility)
	- `tls`: verify backend certificates using `BACKEND_CA_FILE`
	- `tls_insecure_skip_verify`: TLS without cert verification (development only)
- `BACKEND_CA_FILE`: required only when `BACKEND_TLS_MODE=tls`

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
