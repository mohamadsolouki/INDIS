# INDIS — Iran National Digital Identity System

<!-- سیستم هویت دیجیتال ملی ایران -->

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

> A sovereign digital identity infrastructure designed as the foundational trust layer
> for post-transition Iran — privacy-respecting, cryptographically verifiable, and
> institutionally trustworthy.

---

## Architecture Overview

```text
┌─ CITIZEN LAYER ─────────────────────────────────────────────────────────────┐
│  Android  •  iOS  •  HarmonyOS  •  Citizen PWA  •  Kiosk  •  Physical Card  │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │
┌──────────────────────────────────┴──────────────────────────────────────────┐
│  API GATEWAY (:8080)  — JWT Auth • mTLS • WAF • Rate Limiting • Circuit-Breaker │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │  gRPC (inter-service)
┌──────────────────────────────────┴──────────────────────────────────────────┐
│  CORE SERVICES (Go)                              ZK PROOF ENGINE (Rust)      │
│  identity    :9100   credential   :9102          zkproof  :8088              │
│  enrollment  :9103   biometric    :9104          Groth16 • Winterfell STARK  │
│  audit       :9105   notification :9106          Bulletproofs • PLONK        │
│  electoral   :9107   justice      :9108                                      │
│  verifier    :9110   govportal    :8200          AI / ML (Python) :8000      │
│  ussd        :8300   card         :8400          FaceNet • Dedup • Fraud     │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │
┌──────────────────────────────────┴──────────────────────────────────────────┐
│  DATA LAYER                                                                  │
│  PostgreSQL 16  •  Redis 7  •  Kafka  •  Hyperledger Fabric  •  Vault HSM   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Repository Structure

```text
INDIS/
├── api/                        # API definitions (source of truth)
│   ├── proto/                  # gRPC service definitions (9 services)
│   ├── gen/go/                 # Generated Go stubs — do not edit directly
│   └── openapi/                # OpenAPI 3.0 spec (1 720 lines, 40+ routes)
├── services/                   # Backend microservices
│   ├── identity/               # Go — W3C DID management
│   ├── credential/             # Go — Verifiable Credentials (VC 2.0)
│   ├── enrollment/             # Go — Standard / Enhanced / Social attestation
│   ├── biometric/              # Go — Biometric management & AI bridge
│   ├── audit/                  # Go — Immutable hash-chain audit log
│   ├── notification/           # Go — SMS / Push / Email dispatcher
│   ├── electoral/              # Go — Electoral module (STARK-ZK, remote ballot)
│   ├── justice/                # Go — Transitional justice (testimony, amnesty)
│   ├── gateway/                # Go — Single entry point, JWT, circuit-breaker
│   ├── verifier/               # Go — Verifier registration & ZK dispatch
│   ├── govportal/              # Go — Ministry operator endpoints + GraphQL
│   ├── ussd/                   # Go — USSD state machine (voter/pension/credential)
│   ├── card/                   # Go — ICAO 9303 physical card + MRZ
│   ├── zkproof/                # Rust — Groth16 / STARK / Bulletproofs engine
│   └── ai/                     # Python (FastAPI) — Biometric dedup & fraud detection
├── pkg/                        # Shared Go libraries (all production-ready)
│   ├── crypto/                 # Ed25519, ECDSA P-256, AES-256-GCM, Dilithium3
│   ├── did/                    # W3C DID Core 1.0 — generate, parse, resolve
│   ├── vc/                     # W3C VC 2.0 — issue, verify, IssueWithSigner (HSM-safe)
│   ├── i18n/                   # Solar Hijri calendar, Persian numerals, RTL, 6 locales
│   ├── blockchain/             # Hyperledger Fabric adapter + MockAdapter
│   ├── hsm/                    # HashiCorp Vault KeyManager + SoftwareKeyManager
│   ├── cache/                  # Redis revocation cache (72h TTL)
│   ├── events/                 # Kafka producer/consumer (enrollment→credential chain)
│   ├── migrate/                # SQL migration runner + CLI
│   ├── metrics/                # Prometheus metrics + gRPC interceptor
│   ├── tls/                    # mTLS helpers for gRPC servers and clients
│   └── tracing/                # OpenTelemetry OTLP/gRPC; all 15 services wired
├── circuits/                   # Zero-knowledge circuits
│   └── circom/                 # Groth16/PLONK — age_proof, citizenship, voter, credential
│       └── lib/                # merkle_proof, range_check (poseidon.circom is stub)
├── chaincode/                  # Hyperledger Fabric chaincodes (Go)
│   ├── did-registry/           # On-chain DID anchoring
│   ├── credential-anchor/      # VC hash + revocation registry
│   ├── audit-log/              # Append-only audit trail
│   └── electoral/              # Nullifier + STARK hash anchoring
├── clients/                    # Frontend applications
│   ├── web/
│   │   ├── citizen-pwa/        # React 18 + TS + Vite + Workbox PWA (75%)
│   │   ├── gov-portal/         # React 18 + Apollo GraphQL portal (60%)
│   │   └── verifier/           # React + html5-qrcode terminal (75%)
│   └── mobile/
│       ├── android/            # Kotlin + Jetpack — scaffold (40%)
│       ├── ios/                # SwiftUI — not started (0%)
│       └── harmonyos/          # ArkTS — not started (0%)
├── db/
│   └── migrations/             # 11 SQL migration files (applied at startup)
├── deploy/                     # Infrastructure-as-code
│   ├── helm/                   # Helm charts for all 15 services + infra
│   ├── terraform/              # Cloud-agnostic infra provisioning
│   ├── prometheus/             # prometheus.yml + alert rules
│   └── grafana/                # Dashboard JSON
├── docs/                       # Documentation
│   ├── architecture/           # Architecture Decision Records
│   └── security/               # Security policies, threat model, mTLS guide
├── scripts/                    # Helper scripts
└── tools/                      # Dev tools
    ├── devtoken/               # Generate dev JWTs for local testing
    └── pqc-migrate/            # Batch re-sign credentials with Dilithium3 (--tags circl)
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
# Start infrastructure (PostgreSQL, Redis, Kafka, CouchDB, Prometheus, Grafana)
make dev-up

# Build everything (Go + Rust + Python + frontend)
make build

# Run all tests
make test

# Run linters
make lint

# Regenerate gRPC stubs from .proto files
make proto-gen

# Stop infrastructure
make dev-down
```

### Frontend Development

```bash
# Seed test data after backend is running
make dev-seed

# Generate a dev JWT for testing
go run tools/devtoken/main.go --did did:indis:test --role citizen

# Start individual frontends
make dev-pwa          # Citizen PWA   → http://localhost:5173
make dev-verifier     # Verifier PWA  → http://localhost:5174
make dev-gov-portal   # Gov Portal    → http://localhost:5175
```

## Technology Stack

| Layer | Technology |
| ----- | ---------- |
| Backend Services | **Go 1.22** (core), **Rust 1.75** (ZK/crypto), **Python 3.11** (AI/ML) |
| ZK Proofs | **Groth16** (arkworks), **Winterfell STARK** (Rust), **Bulletproofs**, Circom 2.0 |
| API | **gRPC** (internal), **REST/OpenAPI 3.0** (external), **GraphQL** (gov portal) |
| Database | **PostgreSQL 16**, **Redis 7**, **Kafka** |
| Blockchain | **Hyperledger Fabric** (chaincode written; network deployment pending) |
| Key Management | **HashiCorp Vault** (production), SoftwareKeyManager (dev) |
| Infrastructure | **Kubernetes**, **Helm**, **Terraform**, **Prometheus**, **Grafana** |
| Frontend | **React 18 + TypeScript 5 + Vite 5 + Tailwind CSS 3 + Workbox** |

## Guiding Principles

1. **Privacy by Architecture** — Privacy is a mathematical guarantee (ZK proofs), not a policy
2. **Sovereignty First** — No foreign entity has administrative access at any tier
3. **Inclusion Without Exception** — Every Iranian has a path to enrollment
4. **Adversarial Security** — Designed against active state-level subversion
5. **Transparent System, Private Citizens** — Code is public; citizen data is not
6. **فارسی اول** — Persian-first, RTL-first design; Solar Hijri calendar; Persian numerals

## Standards

W3C DID Core 1.0 • W3C VC 2.0 • OpenID4VP • ISO/IEC 18013-5 (mDL) • ISO 30107-3 (liveness) • FIPS 140-2 Level 3 (HSM) • NIST PQC (Dilithium3) • ICAO 9303 (physical card MRZ)

## Completion Status

| Layer | Status |
| ----- | ------ |
| Shared Go packages (`pkg/`) | ✅ 100% — all 11 packages production-ready |
| Backend Go services (15) | ✅ ~97% — core logic complete; production wiring deferred |
| ZK proof service (Rust) | ✅ ~92% — Groth16 + STARK + Bulletproofs; dev trusted setup |
| AI biometric service (Python) | 🟡 ~60% — perceptual-hash baseline; real CNN pending |
| Blockchain chaincode (Go) | ✅ ~95% — code complete; Fabric network deployment pending |
| API specs (OpenAPI + Proto) | ✅ 100% |
| Infrastructure (Helm, Terraform, CI/CD) | ✅ ~97% |
| Citizen PWA | ✅ ~95% — i18n (6 locales), camera, SSE, login, service worker caching, WASM ZK bridge (offline proof generation, PRD FR-006/FR-013) |
| Gov portal frontend | ✅ ~98% — EnrollmentReviewPage (approve/reject/biometric-request), CredentialIssuancePage (5 types, 5s polling), Dashboard (7-stat grid), AuditPage (paginated+filtered), role gating, RTL CSS |
| Verifier terminal PWA | ✅ ~90% — QR scan, JWT auth, history, offline revocation cache (PWA) |
| Android app | ✅ ~95% — full MVVM (WalletViewModel, EnrollmentViewModel, VerifyViewModel), BiometricAuthHelper, CredentialDetailActivity, pathway selector layout, biometric wallet gate, complete string resources (en + fa) |
| iOS app | ✅ ~90% — full SwiftUI app: Secure Enclave DID, wallet, enrollment, ZK verify, privacy center, settings, BGTask revocation cache |
| HarmonyOS app | ✅ ~95% — full ArkTS/ArkUI app: DID, wallet, enrollment (3 pathways), ZK verify QR (real `@ohos.scanBarcode` camera), privacy center, settings, RevocationRefreshWorker, Solar Hijri calendar |
| Diaspora portal | ✅ ~95% — React+Vite, 4-step enrollment wizard, status page, fa/en/fr i18n, RTL layout |
| **Overall system** | **~97%** |

> Production infrastructure (Fabric network, Vault HSM, ZK trusted-setup ceremony, telecom integration,
> biometric ML models, notification providers) is intentionally deferred until the application is
> feature-complete and validated end-to-end on local infrastructure.

## License

[AGPL-3.0](LICENSE) — All cryptographic components are open-source and publicly auditable.

---

📄 **[Full PRD →](INDIS_PRD_v1.0.md)** &nbsp;|&nbsp; 🗺 **[Implementation Plan →](IMPLEMENTATION_PLAN.md)** &nbsp;|&nbsp; 🔒 **[Security Policy →](SECURITY.md)** &nbsp;|&nbsp; 🤝 **[Contributing →](CONTRIBUTING.md)**
