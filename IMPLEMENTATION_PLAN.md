# INDIS Implementation Plan
# نقشه راه پیاده‌سازی INDIS

> **Last updated:** 2026-03-18
> **Current build status:** All 9 Go services + Rust zkproof + Python AI compile cleanly
> **Estimated overall completion:** ~25% of production-ready system

---

## Current State Inventory / وضعیت کنونی

| Component | Status | Notes |
|-----------|--------|-------|
| **Shared libs** (`pkg/crypto`, `pkg/did`, `pkg/vc`, `pkg/i18n`) | ✅ Implemented | Unit tests present in each package |
| **Proto definitions** (9 services) | ✅ Generated | `api/gen/go/` |
| **Identity service** | 🟡 Scaffold | Handler→Service→Repo; service-level tests present; MockAdapter |
| **Credential service** | 🟡 Scaffold+ | 11 VC types; service tests present; consumes `enrollment.completed` |
| **Enrollment service** | 🟡 Scaffold+ | 3 pathways; DID generation; service tests present; publishes `enrollment.completed` |
| **Biometric service** | 🟡 Scaffold | AES-GCM template encrypt; dedup is a stub |
| **Audit service** | 🟡 Scaffold | Hash-chain logic; append-only; no tests |
| **Notification service** | 🟡 Scaffold | 3-tier expiry alerts; no actual delivery |
| **Electoral service** | 🟡 Scaffold | Nullifier double-vote guard; ZK is a stub |
| **Justice service** | 🟡 Scaffold | ZK citizenship proof is a stub |
| **Gateway service** | 🟡 Scaffold+ | HTTP→gRPC proxy; rate limiter; backend transport mode configurable (`plaintext`/`tls`) |
| **ZK service** (Rust) | 🔴 Stub | Traits defined; no proof generation/verification |
| **AI service** (Python) | 🔴 Stub | FastAPI skeleton; no ML models loaded |
| **Blockchain adapter** | 🔴 Mock | `MockAdapter` logs calls; no real chain |
| **DB migrations** | 🟡 Partial | SQL files + `pkg/migrate` runner exist; not wired into all services |
| **ZK circuits** (Circom) | 🔴 Placeholder | No constraint logic |
| **Tests** | 🟡 Partial | Core package tests + identity/enrollment/credential service tests |
| **Mobile apps** | 🔴 None | iOS / Android / HarmonyOS |
| **PWA frontend** | 🔴 None | React + TypeScript |
| **Government portal** | 🔴 None | GraphQL + admin dashboard |
| **Verifier terminal** | 🔴 None | QR scan + ZK display |
| **mTLS / service mesh** | 🟡 Partial | TLS helpers + cert script exist; rollout to all service listeners pending |
| **Kafka event streaming** | 🟡 Partial | `pkg/events` in place; enrollment→credential flow wired |
| **Redis caching** | 🟡 Partial | `pkg/cache` revocation cache exists; credential service wiring pending |
| **Kubernetes / Helm** | 🔴 None | Only docker-compose in Makefile |
| **CI/CD** | 🔴 None | No GitLab CI or ArgoCD config |
| **Observability** | 🔴 None | No Prometheus metrics or traces |
| **HSM integration** | 🔴 None | Ephemeral keys everywhere |
| **Physical card** | 🔴 None | ICAO 9303 / ISO 7816 |
| **USSD/SMS gateway** | 🔴 None | No feature-phone path |

---

## Priority Tiers / اولویت‌بندی

Tiers are ordered by the PRD hard deadlines:
- **Tier 1** → Day 40 (military vetting) — Phase 1
- **Tier 2** → Day 60 (justice) + Month 4 (referendum) — Phase 2
- **Tier 3** → Months 4–12 (national rollout) — Phase 3
- **Tier 4** → Months 12–24 (full coverage) — Phase 4

---

## Tier 1 — Day 40: Working End-to-End Enrollment + Vetting

*Goal: A real human can enroll, receive a DID + credentials, and be verified.*

### T1.1 — Integration Tests for Core Services

**Status (2026-03-18):** Partial complete.

Implemented now:
- `services/identity/internal/service/service_test.go`
- `services/credential/internal/service/service_test.go`
- `services/enrollment/internal/service/service_test.go`
- `pkg/crypto/crypto_test.go`
- `pkg/did/did_test.go`
- `pkg/vc/vc_test.go`
- `pkg/i18n/i18n_test.go`

**Why first:** Tests catch regressions and document expected behavior. Every subsequent Tier builds on these.

**Files to create:**
- `services/identity/internal/service/service_test.go` — register/resolve/deactivate
- `services/credential/internal/service/service_test.go` — issue/verify/revoke
- `services/enrollment/internal/service/service_test.go` — full state machine
- `pkg/crypto/crypto_test.go` — Ed25519, AES-256-GCM, ECDSA
- `pkg/did/did_test.go` — DID generation, parsing, validation
- `pkg/vc/vc_test.go` — issue + verify round-trip
- `pkg/i18n/i18n_test.go` — Solar Hijri dates, Persian numerals

**Integration test scaffold:** Use `testcontainers-go` to spin up Postgres in tests.

**Command when done:** `make test-go` should pass.

---

### T1.2 — Database Migration Runner

**Status (2026-03-18):** Partial complete — `pkg/migrate/migrate.go` exists; startup wiring still pending per service.

The 7 SQL migration files exist but are never applied. Services start against an empty database.

**Approach:** Add a `migrate.go` helper to each service that runs pending SQL files from `db/migrations/` on startup (or use `golang-migrate/migrate`).

**Files to create:**
- `pkg/migrate/migrate.go` — reads and applies `*.sql` files in order
- Wire into each service's `cmd/server/main.go` before the gRPC server starts

---

### T1.3 — Kafka Event Wiring

**Status (2026-03-18):** Partial complete.

Implemented now:
- `pkg/events/events.go`
- `pkg/events/producer.go`
- `pkg/events/consumer.go`
- Enrollment service publishes `indis.enrollment.completed` in `CompleteEnrollment`
- Credential service consumes `indis.enrollment.completed` and auto-issues citizenship/age-range/voter-eligibility credentials

Pending from this item:
- `credential.revoked` pipeline to audit + notification
- `identity.deactivated` pipeline to credential revocation

The enrollment service creates a DID but never tells the credential service to issue credentials. This gap means no credentials are ever issued after enrollment.

**Events needed:**
- `enrollment.completed` → credential service issues Citizenship + AgeRange + VoterEligibility
- `credential.revoked` → audit service logs + notification service alerts holder
- `identity.deactivated` → credential service revokes all active credentials

**Files to create:**
- `pkg/events/events.go` — event type definitions and Kafka topic names
- `pkg/events/producer.go` — thin wrapper around `segmentio/kafka-go` producer
- `pkg/events/consumer.go` — consumer loop with handler registration
- Wire producer into: enrollment service (emit `enrollment.completed`)
- Wire consumer into: credential service (consume `enrollment.completed`)

---

### T1.4 — Redis Revocation Cache

The PRD requires revocation propagation ≤ 60 seconds (FR-002.R1). Currently nothing caches revocation status.

**Files to create:**
- `pkg/cache/revocation.go` — `RevocationCache` interface with `Set(credentialID)`, `IsRevoked(credentialID) bool`, `TTL: 72h`
- `pkg/cache/redis.go` — Redis implementation using `go-redis/v9`
- Wire into credential service's `CheckRevocationStatus` and `RevokeCredential`

---

### T1.5 — mTLS Between Services

**Status (2026-03-18):** Partial complete — `scripts/gen-certs.sh` and `pkg/tls/tls.go` are present; gateway backend transport is now configurable and supports TLS modes.

Currently all gRPC connections use `insecure.NewCredentials()`. Production requires mTLS.

**Approach for now:** Generate self-signed CA + per-service certs via a script, load from environment. Real HSM-backed certs in Phase 2.

**Files to create:**
- `scripts/gen-certs.sh` — generates CA + per-service TLS certs (dev only)
- `pkg/tls/tls.go` — `LoadServerTLS(certFile, keyFile, caFile)` and `LoadClientTLS(caFile)` helpers
- Update `cmd/server/main.go` in each service to use `grpc.Creds(tls.LoadServerTLS(...))`
- Update `proxy/proxy.go` in gateway to use `grpc.Creds(tls.LoadClientTLS(...))`

---

### T1.6 — Minimal AI Biometric Deduplication

The biometric service stubs the `CheckDuplicate` call. The enrollment service cannot complete without dedup.

**Approach:** Implement a minimal deduplication endpoint in the Python AI service using cosine similarity on perceptual hash embeddings. This is not production-quality but unblocks the enrollment flow.

**Files to create:**
- `services/ai/src/biometric/dedup.py` — `DeduplicationService` class with `check_duplicate(template_bytes) -> (is_duplicate, confidence)` using a simple feature vector store
- `services/ai/src/biometric/router.py` — FastAPI router at `/v1/biometric/deduplicate`
- `services/ai/src/biometric/models.py` — pydantic request/response models
- Wire into `services/ai/src/main.py`
- Update `services/biometric/internal/service/service.go` to call the AI service HTTP endpoint

---

### T1.7 — Android Mobile App Skeleton (Persian RTL)

Without a mobile app there is no self-enrollment path. The Android app is the highest-priority client.

**Structure to create:** `clients/android/`

```
clients/android/
  app/
    src/main/
      java/org/indis/app/
        ui/
          enrollment/   EnrollmentActivity, BiometricFragment, DocumentFragment
          wallet/       WalletActivity, CredentialCard, PrivacyCenter
          verify/       VerifyActivity, ZKProofFragment
          settings/     SettingsActivity
        data/
          repository/   IdentityRepository, CredentialRepository
          network/       GatewayApiClient (Retrofit2)
          local/         EncryptedWalletDatabase (Room)
        domain/
          did/           DIDManager (generate on-device)
          zk/            ZKProofManager (Groth16 via JNI → Rust)
          i18n/          PersianCalendar, PersianNumerals
      res/
        values/          strings.xml (English)
        values-fa/       strings.xml (Persian)
        values-ckb/      strings.xml (Kurdish Sorani)
        font/            vazirmatn*.ttf
        layout/          RTL-first layouts
  build.gradle
  settings.gradle
```

**Critical requirements:**
- Persian RTL as default locale
- Vazirmatn font for all text
- Solar Hijri dates using `pkg/i18n` algorithm ported to Kotlin
- Private key generated on-device using Android Keystore (NOT sent to server)
- DID = `did:indis:<hex(sha256(pubkey)[:20])>` — same algorithm as `pkg/did`
- Credential wallet stored in encrypted Room database (EncryptedSharedPreferences for keys)

---

### T1.8 — Prometheus Metrics

Every service needs `/metrics` for observability. Required before Phase 1 launch.

**Files to create:**
- `pkg/metrics/metrics.go` — standard metric definitions: `identity_operations_total`, `credential_operations_total`, `enrollment_operations_total`, `zk_proof_duration_seconds`, `grpc_requests_total`, `grpc_errors_total`
- Wire into each service's main.go: expose `/metrics` HTTP endpoint on a separate port (e.g., `:9090`)
- Add to `docker-compose.yml`: Prometheus scrape config for all services

---

## Tier 2 — Day 60 + Month 4: Electoral + Justice Production-Ready

### T2.1 — ZK-SNARK Groth16 Proof Implementation (Rust)

The Rust zkproof service has trait definitions but no implementation. This is the most technically complex component.

**Crates to add to `Cargo.toml`:**
```toml
arkworks-circuits = { git = "..." }
ark-groth16 = "0.4"
ark-bn254 = "0.4"
ark-ff = "0.4"
ark-relations = "0.4"
ark-serialize = "0.4"
```

**Files to create in `services/zkproof/crates/`:**

```
zkproof-circuits/src/
  age_proof.rs          — AgeAboveCircuit: prove age ≥ threshold
  citizenship_proof.rs  — CitizenshipCircuit: prove valid citizenship VC
  credential_valid.rs   — CredentialValidCircuit: issued + not revoked + not expired
  voter_eligibility.rs  — VoterEligibilityCircuit: citizenship + age≥18 + not excluded

zkproof-core/src/
  groth16.rs            — Groth16ProofGenerator / Groth16ProofVerifier impls
  proving_key.rs        — ProvingKey load/store (from trusted setup ceremony output)

zkproof-server/src/
  handlers.rs           — HTTP handlers: POST /prove, POST /verify
  server.rs             — axum HTTP server wiring
```

**Trusted setup:** For development, use `arkworks` `generate_random_parameters` with a seeded RNG. For production, requires the multi-party ceremony (Phase 0).

---

### T2.2 — ZK-STARK Electoral Proof Implementation (Rust)

For the referendum (hard deadline Month 4), voter eligibility must use ZK-STARK (post-quantum).

**Crates to add:**
```toml
winterfell = "0.8"   # STARK prover/verifier
```

**Files to create:**
```
zkproof-circuits/src/
  electoral_stark.rs   — VoterEligibilityStarkAir: define AIR constraints
                         Inputs: voter DID commitment, election_id, nullifier
                         Output: eligibility boolean + nullifier (public)

zkproof-core/src/
  stark.rs             — StarkProofGenerator / StarkProofVerifier impls
```

---

### T2.3 — Wire Electoral Service to ZK Service

The `services/electoral` currently stubs proof verification. Connect it to `services/zkproof`.

**Changes:**
- Add `zkproof-addr` config to `services/electoral/internal/config/config.go`
- In `services/electoral/internal/service/service.go`:
  - `VerifyEligibility`: call zkproof HTTP endpoint `POST /verify` with STARK proof bytes
  - `CastBallot`: verify ZK proof before accepting ballot

---

### T2.4 — Wire Justice Service to ZK Service

Same pattern as T2.3 for anonymous testimony citizenship proof (Bulletproofs).

**Changes:**
- `services/justice/internal/service/service.go`:
  - `SubmitTestimony`: call zkproof `POST /prove` with Bulletproofs citizenship circuit
  - Verify returned proof before accepting testimony

---

### T2.5 — Voter Eligibility Credential Auto-Issuance

Currently the credential service can issue any credential type but nobody calls it automatically after enrollment. For the referendum, every enrolled citizen needs a VoterEligibility credential.

**Approach:**
- Add `enrollment.completed` consumer in credential service (T1.3 must be done first)
- On `enrollment.completed` event: issue Citizenship + AgeRange + VoterEligibility credentials automatically
- VoterEligibility requires district code from enrollment data — pass in event payload

---

### T2.6 — Remote Voting Infrastructure

The PRD requires both in-person and remote voting. Remote voting needs:

**Files to create:**
- `api/proto/electoral/v1/electoral.proto` additions: `SubmitRemoteBallot` RPC, encrypted ballot message
- `services/electoral/internal/handler/remote_ballot.go` — remote ballot handler
- Ballot encryption: ElGamal on Ristretto255 (additively homomorphic for counting)

---

### T2.7 — Integration Tests: Electoral + Justice Full Flows

- `services/electoral/internal/service/service_test.go` — register election → verify eligibility → cast ballot → double-vote rejection
- `services/justice/internal/service/service_test.go` — submit testimony → link testimony → amnesty workflow

---

## Tier 3 — Months 4–12: National Rollout

### T3.1 — Government Portal (GraphQL + React)

**Structure:** `clients/web/gov-portal/`

```
clients/web/gov-portal/
  src/
    graphql/
      schema.graphql          — ministry-facing queries
      resolvers/              — Go GraphQL resolvers (gqlgen)
    pages/
      enrollment/             — bulk enrollment management
      credentials/            — credential issuance workflows
      audit/                  — audit log viewer
      elections/              — electoral management
    components/
      PersianDatePicker/      — Solar Hijri date picker (RTL)
      CredentialTable/        — data-minimized credential viewer
      AuditTimeline/          — hash-verified audit log display
```

**Backend:** Add GraphQL service (`services/portal`) using `99designs/gqlgen` — talks gRPC to existing backend services.

---

### T3.2 — Verifier Terminal Application

**Structure:** `clients/web/verifier/`

Simple React PWA with:
- QR code scanner (camera API)
- ZK proof verification display (GREEN ✅ / RED ❌ only — PRD FR-013)
- Offline capability: 72h cached revocation list
- Persian RTL UI

---

### T3.3 — Hyperledger Fabric Chaincode

Replace `pkg/blockchain/mock_adapter.go` with a real Fabric adapter.

**Files to create:**
```
blockchain/chaincode/
  did-registry-cc/
    main.go              — DID CRUD chaincode
  credential-anchor-cc/
    main.go              — hash anchor + revocation chaincode
  audit-log-cc/
    main.go              — anonymized verification events chaincode
  electoral-cc/
    main.go              — STARK proof anchoring chaincode

pkg/blockchain/
  fabric_adapter.go      — implements BlockchainAdapter using Fabric SDK
  config.go              — channel names, endorsement policies
```

**Policy:** No personal data on-chain enforced in chaincode (reject any tx with PII patterns).

---

### T3.4 — Physical Card Integration

**Files to create:**
- `services/card/` — new Go service: generates ICAO 9303-compliant card personalization data
- `api/proto/card/v1/card.proto` — `PersonalizeCard`, `ReadCard`, `VerifyCard` RPCs
- NFC interface abstraction: ISO 14443-4 APDU command set

---

### T3.5 — USSD / SMS Gateway

**Files to create:**
- `services/ussd/` — new Go service integrating with Africa's Talking or Infobip API
- Flows: voter eligibility check (`*123*ID#`) → pension check → enrollment status
- No session data retained after end

---

### T3.6 — PWA (Progressive Web App) for Citizens

**Structure:** `clients/web/citizen-pwa/`

React + TypeScript:
- Persian RTL first (i18n with `react-i18next`, Vazirmatn font)
- Solar Hijri date display
- Offline capability with Service Worker + IndexedDB credential wallet
- WebAuthn for on-device key generation (replaces Android Keystore for PWA)
- QR code display for offline verification

---

### T3.7 — Full Test Suite

```
make test    # target: >80% coverage on all Go services
```

- Add `_test.go` files for all remaining services (biometric, audit, notification, gateway, electoral, justice)
- End-to-end test with `testcontainers-go`: Postgres + Redis + (mock Kafka) + (mock ZK service)
- Load test scripts: `k6` scenarios in `tests/load/`
  - `enrollment_load.js` — 500K enrollments/day simulation
  - `electoral_load.js` — 2M verifications/hour simulation

---

### T3.8 — Kubernetes Deployment

**Files to create:**
```
deploy/
  helm/
    indis/
      Chart.yaml
      values.yaml           — default values (image tags, replicas, resource limits)
      values-prod.yaml      — production overrides
      templates/
        identity/           — Deployment, Service, ConfigMap, HPA
        credential/
        enrollment/
        biometric/
        audit/
        notification/
        electoral/
        justice/
        gateway/
        zkproof/
        ai/
  terraform/
    main.tf                 — infrastructure (VMs, load balancer, storage)
    variables.tf
    outputs.tf
```

---

### T3.9 — CI/CD Pipeline (Self-Hosted GitLab)

**Files to create:**
```
.gitlab-ci.yml              — main pipeline definition
.gitlab/
  ci/
    go.yml                  — lint + test + build Go services
    rust.yml                — cargo clippy + cargo test + cargo build
    python.yml              — ruff + pytest
    security.yml            — trivy container scan + gosec
    deploy-staging.yml      — deploy to staging on merge to main
    deploy-prod.yml         — deploy to prod on tag (manual gate)
```

---

## Tier 4 — Months 12–24: Full Coverage + Long-Term

### T4.1 — Post-Quantum Migration (CRYSTALS-Dilithium)

When long-term credentials approach the 10-year horizon where quantum computers become a threat, all Ed25519 signatures on credentials must be replaced.

**Files to create/modify:**
- Add `pkg/crypto/dilithium.go` using `CRYSTALS-Dilithium` from `open-quantum-safe/liboqs-go`
- `services/credential` service: add `--pqc-mode` flag to issue Dilithium-signed credentials
- Migration tool: `tools/pqc-migrate/` — re-signs existing credentials in batches

---

### T4.2 — HSM Integration (HashiCorp Vault + PKCS#11)

Replace all ephemeral key generation with Vault-backed HSM operations.

**Files to create:**
- `pkg/vault/vault.go` — `VaultKeyManager`: `GenerateKey`, `Sign`, `GetPublicKey`
- `pkg/vault/pkcs11.go` — PKCS#11 interface for direct HSM access
- Update all services to use `VaultKeyManager` instead of `crypto.GenerateEd25519KeyPair()`

---

### T4.3 — Diaspora Portal

**Structure:** `clients/web/diaspora/`

Special considerations:
- Multi-language: Persian, English, French
- Embassy agent interface for supervised enrollment
- Postal address verification for physical card delivery
- International timezone handling

---

### T4.4 — International Interoperability

**Scope:**
- W3C DID Resolution for `did:indis:` method — publish DID method spec
- OpenID4VP (Verifiable Presentations) for cross-border credential presentation
- ISO/IEC 18013-5 mobile driving licence interoperability layer
- Embassy integration API for foreign credential acceptance

---

### T4.5 — Circom ZK Circuit Formal Verification

The PRD requires formal verification of all ZK circuits before production deployment.

**Files:**
- `circuits/age_proof/age_proof.circom` — full constraint logic (currently placeholder)
- `circuits/citizenship_proof/citizenship_proof.circom`
- `circuits/voter_eligibility/voter_eligibility.circom`
- `circuits/credential_validity/credential_validity.circom`
- Formal verification using `Ecne` or `Picus` circuit verification tools
- Public audit reports in `docs/audits/`

---

## Development Sequence Recommendation / توالی پیشنهادی

For the current team size and the Day 40 / Day 60 deadlines, the recommended weekly sequence:

```
Week 1–2:   T1.1 (tests) + T1.2 (migration runner) — foundation health
Week 3–4:   T1.3 (Kafka) + T1.4 (Redis) — service wiring
Week 5–6:   T1.5 (mTLS) + T1.8 (metrics) — production-readiness basics
Week 7–8:   T1.6 (AI dedup minimal) + T1.7 (Android skeleton) — enrollment path
Week 9–10:  T2.1 (Groth16 ZK) + T2.2 (STARK ZK) — ZK proof engine
Week 11–12: T2.3 + T2.4 (wire electoral/justice to ZK) + T2.5 (auto-credential) — Day 60 target
Week 13–16: T2.6 (remote voting) + T2.7 (integration tests) + T3.8 skeleton (K8s) — Month 4 referendum target
Month 5+:   T3.1–T3.9 (government portal, verifier, chaincode, physical card, PWA, USSD)
Month 12+:  T4.1–T4.5 (PQC, HSM, diaspora, interop, formal verification)
```

---

## Key Decision Gates / نقاط تصمیم کلیدی

These decisions block specific work streams and must be resolved before the work can begin:

| Decision | Blocks | Deadline |
|----------|--------|----------|
| Blockchain platform selection | T3.3 (Fabric chaincode) | End of Month 1 (Phase 0) |
| ZK trusted setup ceremony | T2.1 production keys | Before Phase 2 launch |
| Biometric SDK selection (open vs commercial) | T1.6 production-quality dedup | End of Month 2 |
| Diaspora voting eligibility | T2.6 remote voting scope | Before Phase 2 starts |
| Minority language launch scope | T3.6 PWA i18n | Before Phase 3 starts |

---

## What's NOT Changing / آنچه تغییر نمی‌کند

The following architectural decisions are settled and should not be revisited:

- **Go** for all backend services — no NodeJS, no Java
- **Rust** for ZK proof service — memory safety in crypto is non-negotiable
- **gRPC** for all inter-service communication — no REST between services
- **PostgreSQL 16** as primary data store — no MongoDB, no DynamoDB
- **ZK proofs as the privacy mechanism** — no alternative "privacy policy" approach
- **Citizen private keys never leave the device** — no server-side key escrow
- **No foreign cloud** — no AWS/Azure/GCP at any tier

---

*نسخه: ۱.۰ | تاریخ: ۲۵۸۵/۱۲ | IranProsperityProject.org*
