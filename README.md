# INDIS — Iran National Digital Identity System
# سیستم هویت دیجیتال ملی ایران

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

> A sovereign digital identity infrastructure designed as the foundational trust layer
> for post-transition Iran — privacy-respecting, cryptographically verifiable, and
> institutionally trustworthy.

---

## Architecture Overview

```
┌─ CITIZEN LAYER ─────────────────────────────────────────────────┐
│  Mobile (Android/iOS/HarmonyOS)  •  PWA  •  Kiosk  •  Card     │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────────┐
│  API GATEWAY — Rate Limiting • mTLS • WAF                       │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────────┐
│  CORE SERVICES (Go)                          ZK PROOFS (Rust)   │
│  Identity • Credential • Enrollment         Groth16 • STARK     │
│  Biometric • Audit • Notification           PLONK • Bulletproof │
│  Electoral • Justice                         AI/ML (Python)     │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────────┐
│  DATA LAYER                                                      │
│  PostgreSQL • Redis • Blockchain [TBD] • HSM • Biometric DB     │
└─────────────────────────────────────────────────────────────────┘
```

## Repository Structure

```
INDIS/
├── api/                    # Protobuf & OpenAPI definitions
│   ├── proto/              # gRPC service definitions
│   └── openapi/            # REST API specs (external verifiers)
├── services/               # Backend microservices
│   ├── identity/           # Go — DID management
│   ├── credential/         # Go — Verifiable Credentials
│   ├── enrollment/         # Go — Enrollment processing
│   ├── biometric/          # Go — Biometric management
│   ├── audit/              # Go — Audit logging
│   ├── notification/       # Go — SMS/Push/Email
│   ├── electoral/          # Go — Electoral module (STARK-ZK)
│   ├── justice/            # Go — Transitional justice
│   ├── gateway/            # Go — API gateway
│   ├── zkproof/            # Rust — ZK proof engine
│   └── ai/                 # Python — Biometric dedup & fraud detection
├── pkg/                    # Shared Go libraries
│   ├── blockchain/         # Blockchain abstraction layer
│   ├── crypto/             # Cryptographic utilities
│   ├── did/                # W3C DID operations
│   ├── vc/                 # Verifiable Credentials
│   └── i18n/               # Internationalisation / RTL
├── circuits/               # Zero-knowledge circuits
│   ├── circom/             # Groth16/PLONK (age, citizenship, voter, credential)
│   └── cairo/              # STARK (electoral)
├── clients/                # Frontend applications
│   ├── mobile/             # Android / iOS / HarmonyOS
│   ├── web/                # Progressive Web App
│   ├── verifier/           # Verifier terminal
│   └── gov-portal/         # Government portal
├── deploy/                 # Infrastructure-as-code
│   ├── kubernetes/         # K8s manifests
│   ├── terraform/          # Infrastructure provisioning
│   └── helm/               # Helm chart
├── docs/                   # Documentation
│   ├── architecture/       # Architecture Decision Records
│   ├── security/           # Security policies & threat model
│   ├── api/                # Generated API docs
│   └── guides/             # Developer & operator guides
└── scripts/                # Helper scripts
```

## Quick Start

### Prerequisites

- **Go** 1.22+
- **Rust** 1.75+
- **Python** 3.11+
- **Docker** & Docker Compose
- **Make**

### Development

```bash
# Start infrastructure (PostgreSQL, Redis, Kafka, etc.)
make dev-up

# Build all services
make build

# Run tests
make test

# Stop infrastructure
make dev-down
```

## Technology Stack

| Layer | Technology |
|-------|-----------|
| Backend Services | **Go** (performance-critical), **Rust** (ZK/crypto), **Python** (AI/ML) |
| ZK Circuits | **Circom 2.0** + SnarkJS (Groth16/PLONK), **Cairo** (STARK) |
| API | **gRPC** (internal), **REST/OpenAPI** (external), **GraphQL** (gov portal) |
| Database | **PostgreSQL** + TimescaleDB, **Redis**, **pgvector** |
| Blockchain | **TBD** — Hyperledger Fabric primary candidate |
| Infrastructure | **Kubernetes**, **Helm**, **Terraform**, **Istio** |

## Guiding Principles

1. **Privacy by Architecture** — Privacy is a mathematical guarantee (ZK proofs)
2. **Sovereignty First** — No foreign entity has administrative access
3. **Inclusion Without Exception** — Every Iranian has a path to enrollment
4. **Adversarial Security** — Designed against active subversion
5. **Transparent System, Private Citizens** — Code is public; citizen data is not
6. **فارسی اول** — Persian-first, RTL-first design

## Standards

W3C DID Core 1.0 • W3C VC 2.0 • OpenID4VP • ISO 18013-5 • ISO 30107-3 • FIPS 140-2 Level 3 • NIST PQC

## License

[AGPL-3.0](LICENSE) — Source code for all cryptographic components is open-source and publicly auditable.

---

📄 **[Full PRD →](INDIS_PRD_v1.0.md)** &nbsp;|&nbsp; 🔒 **[Security Policy →](SECURITY.md)** &nbsp;|&nbsp; 🤝 **[Contributing →](CONTRIBUTING.md)**
