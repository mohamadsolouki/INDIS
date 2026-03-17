# INDIS — API Definitions

## gRPC / Protobuf

Service definitions for internal communication between INDIS microservices.

| Service | Proto File | Description |
|---------|-----------|-------------|
| Identity | `proto/identity/v1/identity.proto` | DID management (W3C DID Core 1.0) |
| Credential | `proto/credential/v1/credential.proto` | Verifiable Credential issuance & verification |
| Enrollment | `proto/enrollment/v1/enrollment.proto` | Enrollment processing (3 pathways) |
| Biometric | `proto/biometric/v1/` | Biometric capture & deduplication |
| Audit | `proto/audit/v1/` | Audit logging |
| Notification | `proto/notification/v1/` | SMS/Push/Email alerts |
| Electoral | `proto/electoral/v1/` | Electoral verification (STARK-ZK) |
| Justice | `proto/justice/v1/` | Anonymous testimony & amnesty workflows |

## REST / OpenAPI

External-facing REST APIs for third-party verifiers and government portals.
Specifications are in `openapi/v1/`.

## Code Generation

```bash
make proto-gen
```
