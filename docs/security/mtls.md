# mTLS and gRPC Transport Configuration

This document centralizes transport security configuration for INDIS Tier 1 services.

## Scope

- gRPC servers: identity, credential, enrollment, biometric, audit, notification, electoral, justice
- gRPC clients: gateway backend proxy to all service backends

## Server-side Environment Variables

All Go gRPC services use the same environment contract via pkg/tls and startup wiring:

- GRPC_TLS_MODE: plaintext or tls
- TLS_CERT_FILE: server certificate path (required when GRPC_TLS_MODE=tls)
- TLS_KEY_FILE: server private key path (required when GRPC_TLS_MODE=tls)
- TLS_CA_FILE: CA path (optional). When set in tls mode, client certificates are required and verified.

Behavior summary:

- GRPC_TLS_MODE=plaintext: development/local compatibility mode
- GRPC_TLS_MODE=tls with cert/key only: one-way TLS
- GRPC_TLS_MODE=tls with cert/key + TLS_CA_FILE: mTLS (client cert verification enabled)

## Gateway Backend Client Transport Variables

Gateway controls backend transport mode via:

- BACKEND_TLS_MODE: plaintext, tls, or tls_insecure_skip_verify (dev only)
- BACKEND_CA_FILE: required when BACKEND_TLS_MODE=tls
- BACKEND_CLIENT_CERT_FILE: optional, but must be paired with BACKEND_CLIENT_KEY_FILE
- BACKEND_CLIENT_KEY_FILE: optional, paired with BACKEND_CLIENT_CERT_FILE

Behavior summary:

- plaintext: insecure.NewCredentials (local development)
- tls: certificate verification with BACKEND_CA_FILE
- tls + client cert/key: mutual TLS to backend services
- tls_insecure_skip_verify: encrypted but no cert verification (development only)

## Local Development Certificate Workflow

Generate test CA and service certs:

- scripts/gen-certs.sh

Production key material and certificate lifecycle are expected to migrate to HSM-backed processes in later phases.

## Validation Checklist

- All gRPC servers load transport options via pkg/tls ServerOptionsFromEnv.
- Gateway proxy dials all backend clients through one transport configuration path.
- Gateway config validates incompatible cert mode combinations.
- Service startup test packages compile with transport settings and migration startup code.
