# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

INDIS (Iran National Digital Identity System) is a sovereign digital identity infrastructure built on W3C DID/VC standards, zero-knowledge proofs, and a privacy-by-architecture design. It handles 88M+ domestic citizens and 8-10M diaspora with ZK-proof-based verification so that verifiers learn only boolean claims, never raw identity data.

## Commands

### Development Lifecycle
```bash
make dev-up      # Start local infrastructure (Postgres, Redis, Kafka, CouchDB, Prometheus, Grafana)
make build       # Build all services (Go + Rust + Python)
make test        # Run all tests
make lint        # Run all linters
make dev-down    # Tear down infrastructure
make proto-gen   # Regenerate gRPC stubs from .proto files → api/gen/go/
make clean       # Remove build artifacts
```

### Per-Language Targets
```bash
make build-go / test-go / lint-go        # golangci-lint, go test ./...
make build-rust / test-rust / lint-rust  # cargo build, cargo test, cargo clippy -- -D warnings
make build-python / test-python / lint-python  # pytest tests/ -v, ruff check src/
```

### Single-Service Testing
```bash
cd services/<service-name> && go test ./...          # Go service
cd services/zkproof && cargo test                    # Rust ZK service
cd services/ai && python -m pytest tests/ -v         # Python AI service
```

### Prerequisites
Go 1.22+, Rust 1.75+, Python 3.11+, Docker, protoc, Make

## Architecture

### Service Topology
INDIS is a polyglot microservice system. All internal communication uses **gRPC**; external verifiers use **REST/OpenAPI**; the gov portal uses **GraphQL**.

| Layer | Components |
|-------|-----------|
| Client | Mobile (Android/iOS/HarmonyOS), PWA (React+TS), Verifier terminal, Gov portal |
| Gateway | `services/gateway` — rate limiting, mTLS, WAF, routing |
| Core Services (Go) | identity, credential, enrollment, biometric, audit, notification, electoral, justice |
| ZK Service (Rust) | `services/zkproof` — Groth16, PLONK, STARK, Bulletproofs via 3-crate workspace |
| AI Service (Python) | `services/ai` (FastAPI) — biometric deduplication, fraud detection |
| Shared Libraries (Go) | `pkg/` — blockchain, crypto, did, vc, i18n |

### Data Layer
- **PostgreSQL 16** — primary identity/credential/enrollment store
- **Redis 7** — sessions, revocation lists (72h offline cache)
- **Kafka** — async event streaming between services
- **CouchDB** — Hyperledger Fabric dev state DB
- **HSM** — FIPS 140-2 Level 3 key storage (production)

### API Definitions
Proto files live in `api/proto/`; generated Go stubs are written to `api/gen/go/` (never edit generated files directly). Three services have proto definitions: `identity/v1`, `credential/v1`, `enrollment/v1`.

### ZK Circuits
`circuits/` contains Circom 2.0 circuits (Groth16/PLONK) for `age_proof`, `citizenship_proof`, `voter_eligibility`, `credential_validity`, and a Cairo circuit for STARK-based `electoral_proof`. All circuits are placeholder scaffolds — actual constraint logic is TODO.

### Shared Go Packages (`pkg/`)
- `crypto` — Ed25519, ECDSA, AES-256-GCM, Dilithium (NIST PQC)
- `did` — W3C DID Core 1.0 operations
- `vc` — W3C Verifiable Credentials 2.0
- `blockchain` — Abstraction layer over Hyperledger Fabric (and future chains)
- `i18n` — Persian/RTL support, Solar Hijri calendar, Vazirmatn typography

### Key Design Patterns
- **Layered Go services:** Handler → Service → Repository
- **ZK-first verification:** Verifiers receive boolean proofs, never raw attributes
- **Offline capability:** Mobile clients generate ZK proofs locally; PWA caches revocation lists for 72h
- **Enrollment pathways:** Standard (documents + biometrics), Enhanced (civil registry), Social Attestation (3+ community co-attestors + biometrics)
- **Verification tiers:** Level 1 (QR/ZK, boolean) → Level 4 (emergency override with full audit trail)

## Code Standards

**Go:** Effective Go conventions; all exported symbols require doc comments.
**Rust:** Deny warnings (`-D warnings`); all `unsafe` blocks require justification comments.
**Python:** PEP 8 via `ruff`; type hints required on all function signatures.
**Cryptographic code:** Must reference the standard being implemented; requires review by 2+ maintainers.
**Commits:** Conventional Commits format (`feat:`, `fix:`, `docs:`, etc.)

## Persian/RTL Requirements

All UI components must be RTL-first. Dates default to Solar Hijri calendar. Numeric output should use Persian numerals by default. Use `pkg/i18n` for all localization.
