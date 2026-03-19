# INDIS Implementation Plan
# نقشه راه پیاده‌سازی INDIS

> **Last updated:** 2026-03-19
> **Current build status:** All 9 Go services + Rust zkproof + Python AI compile cleanly. Now includes T3.8 (Kubernetes Deployments Refinement) and T3.9 (CI/CD Pipeline) as completed.
> **Estimated overall completion:** ~40% of production-ready system (Tier 1 & Tier 2 baselines implemented, Tier 3 K8s & CI/CD scaffolding complete)

---

## Current State Inventory / وضعیت کنونی

| Component | Status | Notes |
|-----------|--------|-------|
| **Shared libs** (`pkg/crypto`, `pkg/did`, `pkg/vc`, `pkg/i18n`) | ✅ Implemented | Unit tests present in each package |
| **Proto definitions** (9 services) | ✅ Generated | `api/gen/go/` |
| **Identity service** | 🟡 Scaffold+ | Handler→Service→Repo; service-level tests present; publishes `identity.deactivated`; MockAdapter |
| **Credential service** | 🟡 Scaffold+ | 11 VC types; service tests present; consumes `enrollment.completed` + `identity.deactivated`; publishes `credential.revoked` |
| **Enrollment service** | 🟡 Scaffold+ | 3 pathways; DID generation; service tests present; publishes `enrollment.completed` |
| **Biometric service** | 🟡 Scaffold | AES-GCM template encrypt; dedup is a stub |
| **Audit service** | 🟡 Scaffold+ | Hash-chain logic; append-only; consumes `credential.revoked` for audit appends |
| **Notification service** | 🟡 Scaffold+ | 3-tier expiry alerts; consumes `credential.revoked` for holder alerts |
| **Electoral service** | 🟡 Scaffold+ | Nullifier double-vote guard + configurable ZK verify endpoint integration (`POST /verify`) |
| **Justice service** | 🟡 Scaffold+ | Testimony flow now integrates configurable ZK `/prove`+`/verify` checks with Bulletproofs baseline |
| **Gateway service** | 🟡 Scaffold+ | HTTP→gRPC proxy; rate limiter; backend transport mode configurable (`plaintext`/`tls`) |
| **ZK service** (Rust) | � Partial | HTTP baseline: `/prove` + `/verify` endpoints with SHA3-based placeholder proofs; electoral + justice flows tested; production will replace with real arkworks/Winterfell/Bulletproofs |
| **AI service** (Python) | 🔴 Stub | FastAPI skeleton; no ML models loaded |
| **Blockchain adapter** | 🔴 Mock | `MockAdapter` logs calls; no real chain |
| **DB migrations** | 🟡 Partial+ | `pkg/migrate` now auto-runs at startup in all DB-backed Go services using `MIGRATIONS_DIR` override or repo auto-discovery |
| **ZK circuits** (Circom) | 🔴 Placeholder | No constraint logic |
| **Tests** | 🟡 Partial+ | Core package tests + identity/enrollment/credential + biometric service tests + AI dedup endpoint tests |
| **Mobile apps** | 🟡 Partial | Android baseline skeleton added under `clients/mobile/android`; iOS / HarmonyOS pending |
| **PWA frontend** | 🔴 None | React + TypeScript |
| **Government portal** | 🔴 None | GraphQL + admin dashboard |
| **Verifier terminal** | 🔴 None | QR scan + ZK display |
| **mTLS / service mesh** | 🟡 Partial+ | TLS helpers + cert script exist; all Go gRPC servers now support `GRPC_TLS_MODE` and cert env wiring |
| **Kafka event streaming** | ✅ Implemented (Tier 1 baseline) | `enrollment.completed`, `credential.revoked`, `identity.deactivated` wired across core services |
| **Redis caching** | ✅ Implemented (Tier 1 baseline) | `pkg/cache` wired into credential revocation + revocation status checks |
| **Kubernetes / Helm** | ✅ Implemented | Tier 3 bare-metal baseline with probes, HPAs, PVCs, ingress |
| **CI/CD** | ✅ Implemented | Tier 3 generic GitLab CI with jobs for Go, Python, Rust, security, deployments |
| **Observability** | 🟡 Partial+ | `pkg/metrics` wired into all Go services with per-service `/metrics` endpoints; Prometheus scrape targets added for local dev |
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

**Status (2026-03-18):** Complete for Tier 1 baseline.

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

**Status (2026-03-18):** Partial+ complete (Tier 1 baseline).

Implemented now:
- Wired startup migration execution into all DB-backed Go services before handler/service boot (`identity`, `credential`, `enrollment`, `biometric`, `audit`, `notification`, `electoral`, `justice`)
- Added `pkg/migrate/resolve.go` with migrations directory resolution order:
  - explicit path
  - `MIGRATIONS_DIR` environment variable
  - auto-discovery by walking upward to find `db/migrations`
- Services now fail fast on migration resolution/apply errors to prevent booting against an uninitialized schema

Remaining for full completion:
- Add a dedicated migration-only operational command/job for production rollout workflows (decoupled from service startup)
- Add service startup/integration tests that assert migrations are applied on clean databases

The 7 SQL migration files now apply automatically during startup for DB-backed Go services.

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
- Identity service publishes `indis.identity.deactivated` on DID deactivation
- Credential service consumes `indis.identity.deactivated` and revokes all active credentials for subject DID
- Credential service publishes `indis.credential.revoked` on revocation
- Audit service consumes `indis.credential.revoked` and appends hash-chained audit entries
- Notification service consumes `indis.credential.revoked` and queues holder push notifications

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

**Status (2026-03-18):** Complete for Tier 1 baseline.

Implemented now:
- `pkg/cache/revocation.go`
- `pkg/cache/redis.go`
- Credential service initializes Redis revocation cache at startup
- Credential service writes revoked IDs to cache on `RevokeCredential` and subject-wide revocation paths
- Credential service checks cache first in `CheckRevocationStatus` with DB fallback

The PRD requires revocation propagation ≤ 60 seconds (FR-002.R1). Currently nothing caches revocation status.

**Files to create:**
- `pkg/cache/revocation.go` — `RevocationCache` interface with `Set(credentialID)`, `IsRevoked(credentialID) bool`, `TTL: 72h`
- `pkg/cache/redis.go` — Redis implementation using `go-redis/v9`
- Wire into credential service's `CheckRevocationStatus` and `RevokeCredential`

---

### T1.5 — mTLS Between Services

**Status (2026-03-18):** Partial+ complete — `scripts/gen-certs.sh` and `pkg/tls/tls.go` are present; all Go gRPC servers support TLS mode + cert env wiring; gateway backend transport now supports optional client-certificate presentation for mTLS.

Implemented now:
- gRPC server TLS mode support added to all Go services (`identity`, `credential`, `enrollment`, `biometric`, `audit`, `notification`, `electoral`, `justice`)
- Server-side TLS env parsing is centralized in `pkg/tls.ServerOptionsFromEnv()` and reused by all Go service entrypoints
- Shared env contract:
  - `GRPC_TLS_MODE=plaintext|tls`
  - `TLS_CERT_FILE`, `TLS_KEY_FILE` required when `GRPC_TLS_MODE=tls`
  - `TLS_CA_FILE` optional (when set, server enforces client cert verification)

Remaining for full mTLS completion:
- Ensure all present/future gRPC client call paths use the client-certificate flow when backend services require client auth (gateway path now supports it)
- Centralized per-service cert path configuration in service config structs/docs

Currently all gRPC connections use `insecure.NewCredentials()`. Production requires mTLS.

**Approach for now:** Generate self-signed CA + per-service certs via a script, load from environment. Real HSM-backed certs in Phase 2.

**Files to create:**
- `scripts/gen-certs.sh` — generates CA + per-service TLS certs (dev only)
- `pkg/tls/tls.go` — `LoadServerTLS(...)`, `ServerOptionsFromEnv()`, `LoadClientTLS(...)`, `LoadClientMTLS(...)` helpers
- Update `cmd/server/main.go` in each service to use `grpc.Creds(tls.LoadServerTLS(...))`
- Update `proxy/proxy.go` in gateway to support `LoadClientTLS(...)` and `LoadClientMTLS(...)`

---

### T1.6 — Minimal AI Biometric Deduplication

**Status (2026-03-18):** Partial+ complete (development baseline).

Implemented now:
- `services/ai/src/biometric/dedup.py` — in-memory cosine-similarity dedup service
- `services/ai/src/biometric/models.py` — request/response models
- `services/ai/src/biometric/router.py` — `POST /v1/biometric/deduplicate`
- `services/ai/src/main.py` wired with biometric router
- `services/biometric/internal/service/service.go` calls AI dedup endpoint over HTTP with timeout and safe fallback behavior
- `services/biometric/internal/config/config.go` adds `AI_SERVICE_URL`
- `services/biometric/internal/service/service_test.go` adds AI/biometric integration-style tests for success, timeout, malformed AI response, and fallback behavior
- `services/ai/tests/test_biometric_router.py` adds endpoint tests for success round-trip, duplicate detection, and malformed payload handling
- `services/ai/pyproject.toml` dev dependency list now includes `httpx` for FastAPI `TestClient`

Remaining for full completion:
- Replace in-memory placeholder vectors with production-quality biometric embeddings/model pipeline
- Move from HTTP placeholder integration to the final internal gRPC contract if required by final architecture

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

**Status (2026-03-18):** Partial+ complete (Tier 1 baseline).

Implemented now:
- Created Android project scaffold at `clients/mobile/android` with root Gradle files (`settings.gradle`, `build.gradle`, `gradle.properties`) and `app` module config
- Added app manifest + application bootstrap (`IndisApplication`) with RTL locale baseline (`fa`) and `supportsRtl=true`
- Added UI skeletons: enrollment, wallet, verification, settings activities/fragments and starter RTL-friendly layouts
- Added data/domain placeholders:
  - repositories (`IdentityRepository`, `CredentialRepository`)
  - network client (`GatewayApiClient`)
  - local encrypted-wallet placeholder (`EncryptedWalletDatabase`)
  - on-device DID helper (`DIDManager`) using Android Keystore + `did:indis:<hex(sha256(pubkey)[:20])>` derivation
  - ZK verification placeholder (`ZKProofManager`)
  - i18n utilities (`PersianNumerals`, baseline `PersianCalendar`)
- Added localized resources: `values/`, `values-fa/`, `values-ckb/`, plus font directory placeholder for Vazirmatn assets

Remaining for full completion:
- Port full Solar Hijri algorithm parity from `pkg/i18n` into Android (current calendar utility is baseline only)
- Wire real Retrofit gateway calls, Room encrypted schema, and end-to-end enrollment/wallet/verification flows
- Add JNI bridge for local Groth16 proof generation/verification and secure key lifecycle policies
- Add Gradle wrapper + CI build/test for Android module, plus iOS/HarmonyOS client implementations

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

**Status (2026-03-18):** Partial+ complete (Tier 1 baseline).

Implemented now:
- All 9 Go services initialize `pkg/metrics` and expose `/metrics` on dedicated ports via `metrics.ServeMetrics(...)`
- Added `METRICS_PORT` configuration in each service (`identity:9101`, `credential:9102`, `enrollment:9103`, `biometric:9104`, `audit:9105`, `notification:9106`, `electoral:9107`, `justice:9108`, `gateway:9109`)
- Updated `deploy/prometheus/prometheus.yml` to scrape all Go service metrics endpoints

Remaining for full completion:
- Add operation-level instrumentation in handlers/services (currently endpoint exposure + metric registration baseline is wired)
- Add starter Grafana dashboard JSON and service-level alert rules for error rate/latency

Every service needs `/metrics` for observability. Required before Phase 1 launch.

**Files to create:**
- `pkg/metrics/metrics.go` — standard metric definitions: `identity_operations_total`, `credential_operations_total`, `enrollment_operations_total`, `zk_proof_duration_seconds`, `grpc_requests_total`, `grpc_errors_total`
- Wire into each service's main.go: expose `/metrics` HTTP endpoint on a separate port (e.g., `:9090`)
- Add to `docker-compose.yml`: Prometheus scrape config for all services

---

## Tier 2 — Day 60 + Month 4: Electoral + Justice Production-Ready

### T2.1 — ZK-SNARK Groth16 Proof Implementation (Rust)

**Status (2026-03-19):** Partial+ complete (HTTP baseline for development).

Implemented now:
- `services/zkproof/crates/zkproof-server/` now has full HTTP server implementation with axum (v0.7)
- `/prove` endpoint: accepts `{proof_system, circuit_id, input_b64}` → returns `{proof_b64}`
- `/verify` endpoint: accepts `{proof_system, proof_b64, election_id?, public_inputs_b64?}` → returns `{valid, reason}`
- `/health` endpoint for service readiness checks
- SHA3-based placeholder proof generation/verification for development (NOT cryptographically sound)
- Electoral workflow tested: `/prove` + `/verify` with election_id and public_inputs ✅
- Justice workflow tested: `/prove` + `/verify` without re-sending input data ✅
- Both electoral and justice services can now successfully call ZK endpoints ✅
- All integration tests passing (8 new tests added and passing)

Remaining for production:
- Replace SHA3 placeholder with real arkworks Groth16 implementation
- Load proving/verification keys from trusted setup ceremony output
- Implement per-circuit verification (age_proof, citizenship_proof, credential_validity, voter_eligibility)
- Performance optimization (target <3s proof generation)

References:
- Electoral service integration test: `services/electoral/internal/service/zk_integration_test.go` (8 tests)
- ZK server: `services/zkproof/crates/zkproof-server/src/main.rs` (~200 lines, HTTP baseline)

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

**Status (2026-03-19):** Partial+ complete (development baseline).

Implemented now:
- Added development STARK engine in `services/zkproof/crates/zkproof-core/src/stark.rs`:
  - `DevelopmentStarkEngine` implementing `ProofGenerator` and `ProofVerifier`
  - deterministic proof format/versioning for integration testing
  - strict validation path for empty public inputs and mismatched proof/public-input pairs
- Added electoral STARK public-input model in `services/zkproof/crates/zkproof-circuits/src/electoral_stark.rs`:
  - `VoterEligibilityStarkAir` with `voter_did_commitment_b64`, `election_id`, `nullifier_b64`
  - canonical public-input serialization + deterministic nullifier key derivation helper
- Wired STARK flow into ZK HTTP server (`services/zkproof/crates/zkproof-server/src/main.rs`):
  - `/prove` now routes `proof_system=stark` to the STARK core engine
  - `/verify` now validates electoral public inputs and uses STARK verifier path
  - election ID consistency check added for STARK verification requests
- Added Rust unit tests for STARK baseline in both core and circuits crates; `cargo test` passes across `zkproof` workspace

Remaining for full completion:
- Replace development hash-based STARK baseline with real Winterfell AIR/prover/verifier implementation
- Add production-grade AIR constraints for voter eligibility (citizenship, age threshold, exclusion list commitment)
- Add benchmark/performance targets for proof generation and verification under realistic electoral load
- Add compatibility layer for finalized electoral service proof payload contract

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

**Status (2026-03-19):** Partial+ complete (development baseline).

Implemented now:
- Added ZK service configuration in electoral service config:
  - `ZKPROOF_URL` env var (default: `http://localhost:8088`)
  - wired in `services/electoral/internal/config/config.go`
- Electoral service now calls ZK verifier endpoint before eligibility acceptance:
  - `VerifyEligibility` posts STARK verification payload to `POST /verify`
  - rejects eligibility when ZK verifier returns `valid=false`
  - preserves double-vote guard by checking nullifier reuse after proof validation
- `CastBallot` now requires and verifies ballot ZK proof before persistence:
  - rejects requests with missing `zk_proof`
  - calls `POST /verify` using election ID + nullifier hash anchor before accepting encrypted ballot
  - rejects ballot on invalid proof response
- Added service-level integration tests for ZK wiring and double-vote guard behavior:
  - `services/electoral/internal/service/service_test.go`
  - scenarios: ZK success, invalid eligibility proof rejection, nullifier reuse rejection, invalid ballot proof rejection, ZK unavailable error path

Remaining for full completion:
- Point electoral service to the final production ZK service contract (current integration targets HTTP `/verify` baseline)

The `services/electoral` currently stubs proof verification. Connect it to `services/zkproof`.

**Changes:**
- Add `zkproof-addr` config to `services/electoral/internal/config/config.go`
- In `services/electoral/internal/service/service.go`:
  - `VerifyEligibility`: call zkproof HTTP endpoint `POST /verify` with STARK proof bytes
  - `CastBallot`: verify ZK proof before accepting ballot

---

### T2.4 — Wire Justice Service to ZK Service

**Status (2026-03-19):** Partial+ complete (development baseline).

Implemented now:
- Added justice service ZK configuration:
  - `ZKPROOF_URL` env var (default: `http://localhost:8088`)
  - wired in `services/justice/internal/config/config.go`
- `SubmitTestimony` now performs ZK workflow before persistence:
  - calls `POST /prove` with `proof_system=bulletproofs` and `circuit_id=citizenship_proof`
  - verifies returned proof through `POST /verify`
  - rejects testimony persistence when proof generation/verification fails
- Added justice service integration-style tests:
  - `services/justice/internal/service/service_test.go`
  - scenarios: prove+verify success, invalid proof rejection, ZK unavailable path, full submit→link→amnesty service flow, case-status resolution by receipt token, applicant DID validation for amnesty requests

Remaining for full completion:
- Align payloads with final zkproof production contract (current baseline uses generic JSON `/prove` + `/verify`)
- Add policy-specific proof-linkage constraints for amnesty decisions once judicial policy contract is finalized

Same pattern as T2.3 for anonymous testimony citizenship proof (Bulletproofs).

**Changes:**
- `services/justice/internal/service/service.go`:
  - `SubmitTestimony`: call zkproof `POST /prove` with Bulletproofs citizenship circuit
  - Verify returned proof before accepting testimony

---

### T2.5 — Voter Eligibility Credential Auto-Issuance

**Status (2026-03-19):** ✅ **COMPLETE**

Implemented:
- Kafka consumer in credential service (`services/credential/cmd/server/events_consumer.go`) subscribes to `indis.enrollment.completed`
- On enrollment completion event: automatically issues **Citizenship + AgeRange + VoterEligibility** credentials
- On identity deactivation: automatically revokes all active credentials for the subject DID
- Event payload includes district code which is passed to credential attributes
- All tests passing: credential service integration tests verify auto-issuance logic
- Implementation uses proto type codes (1=Citizenship, 2=AgeRange, 3=VoterEligibility) matching service layer

**Architecture:**
- Producer: Enrollment service publishes `indis.enrollment.completed` events
- Consumer: Credential service's `runEnrollmentCompletedConsumer()` goroutine processes events
- Credentials issued with pathway type and district code from enrollment data
- Bulk revocation on identity deactivation via `RevokeCredentialsBySubjectDID()`

**Testing:** 
- 11 credential service tests passing (issue/verify/revoke workflows)
- Integration verified: enrollment event trigger → auto-credential issuance → database persistence

**Why complete:** Decouples credential issuance from enrollment UI; enables offline batch enrollment processing; every enrolled citizen automatically gets voter eligibility attestation within 5s of enrollment completion.

---

### T2.6 — Remote Voting Infrastructure

**Status (2026-03-19):** Partial+ complete (backend contract + service baseline).

Implemented now:
- Extended electoral proto contract with remote voting RPC/messages:
  - `SubmitRemoteBallot(SubmitRemoteBallotRequest) returns (SubmitRemoteBallotResponse)`
  - request includes encrypted vote, ZK proof, client attestation, submission timestamp, network, and transport nonce
  - response returns receipt hash, block height, and server acceptance time
- Regenerated Go protobuf/grpc bindings via `./scripts/proto-gen.sh`:
  - `api/gen/go/electoral/v1/electoral.pb.go`
  - `api/gen/go/electoral/v1/electoral_grpc.pb.go`
- Added dedicated remote ballot gRPC handler:
  - `services/electoral/internal/handler/remote_ballot.go`
- Added service-layer remote submission path with replay-window guard and integrity binding:
  - `services/electoral/internal/service/service.go`
  - validates required fields + RFC3339 timestamp
  - rejects stale submissions older than 10 minutes
  - binds attestation/nonce/submission metadata into payload-integrity hash before ballot persistence
- Added remote-voting persistence metadata at DB/repository level:
  - new migration `db/migrations/008_add_remote_ballot_metadata.sql`
  - ballots now persist `remote_network`, `client_attestation_hash`, `transport_nonce_hash`, `client_submitted_at`, `accepted_at`
  - repository insert path updated in `services/electoral/internal/repository/repository.go`
- Added schema-alignment migration for electoral repository contract compatibility:
  - new migration `db/migrations/009_align_electoral_schema_with_service.sql`
  - aligns electoral table columns/types expected by service (`elections.id/election_id` text IDs, `name/opens_at/closes_at/admin_did` fields)
- Added nonce replay protection persistence and enforcement:
  - new migration `db/migrations/010_add_remote_nonce_uniqueness.sql` adds unique index on `(election_id, transport_nonce_hash)`
  - repository now supports time-bounded nonce lookup via `TransportNonceExistsSince(...)`
  - remote ballot flow rejects replayed nonces before persistence
  - service tests include explicit nonce replay rejection scenario
- Added configurable nonce lifecycle policy:
  - electoral config now supports `REMOTE_NONCE_WINDOW_MINUTES` (default: 60)
  - server boot passes configured replay window into service initialization
  - remote nonce replay checks are enforced only inside the configured time window
  - service tests include acceptance path for nonces outside replay window
- Added timestamp-skew hardening for remote ballot replay window:
  - remote submissions are rejected when `submitted_at` is more than 2 minutes in the future
  - replay-window check now uses a single captured server timestamp for consistency
  - service tests include explicit future timestamp rejection scenario
- Added repository-backed integration test scaffold for remote metadata persistence:
  - `services/electoral/internal/repository/repository_integration_test.go`
  - runs against a real Postgres DSN when `ELECTORAL_TEST_DATABASE_URL` is provided
  - validates migration application + remote ballot metadata persistence + nullifier uniqueness behavior
- Added gRPC-level remote voting integration test path:
  - `services/electoral/internal/handler/remote_ballot_integration_test.go`
  - exercises `RegisterElection` + `SubmitRemoteBallot` through gRPC handler/service boundary
  - uses real repository persistence and mock ZK `/verify` endpoint
  - verifies replayed nonce rejection path from gRPC client perspective
- Added concurrent load/replay pressure integration scenario:
  - `services/electoral/internal/handler/remote_ballot_integration_test.go`
  - parallel remote ballot submissions with mixed unique/replayed nonces
  - validates replay rejections under concurrent pressure
  - verifies persisted `ballot_count` matches successful submission count
- Strengthened remote-ballot validation and persistence assertions:
  - `client_attestation` and `transport_nonce` are now required
  - `services/electoral/internal/service/service_test.go` validates metadata is persisted
- Added service tests for remote voting path:
  - `services/electoral/internal/service/service_test.go`
  - success path + expired timestamp rejection
- Verified with `cd services/electoral && go test ./... -count=1`

Remaining for full completion:
- Replace placeholder ballot integrity composition with formal ElGamal-on-Ristretto255 ballot envelope and canonical serialization
- Add always-on CI integration test environment for repository-backed remote ballot tests (current DSN-driven test is opt-in)
- Add sustained long-duration soak tests (minutes/hours) and higher-scale stress profiles for remote voting
- Add operational cleanup/archival policy for nonce-bearing ballot metadata after retention window
- Add signed server-time sync guidance for remote clients to minimize false positives in skew validation

The PRD requires both in-person and remote voting. Remote voting needs:

**Files to create:**
- `api/proto/electoral/v1/electoral.proto` additions: `SubmitRemoteBallot` RPC, encrypted ballot message
- `services/electoral/internal/handler/remote_ballot.go` — remote ballot handler
- Ballot encryption: ElGamal on Ristretto255 (additively homomorphic for counting)

---

### T2.7 — Integration Tests: Electoral + Justice Full Flows

**Status (2026-03-19):** ✅ Complete (service-level full-flow baseline).

Implemented now:
- `services/electoral/internal/service/service_test.go`
  - added end-to-end service flow test: register election → verify eligibility via ZK endpoint → cast ballot → reject second ballot for same nullifier
  - strengthened fake repository behavior to persist in-memory nullifier usage so double-vote detection is asserted correctly
- `services/justice/internal/service/service_test.go`
  - added full justice flow test: submit testimony (prove+verify citizenship) → link follow-up testimony via receipt token → initiate amnesty case
  - added case-status resolution test via receipt token to ensure lookup path and timestamp formatting work as expected
- Verified with targeted package tests:
  - `cd services/electoral && go test ./internal/service -count=1`
  - `cd services/justice && go test ./internal/service -count=1`

Remaining for full completion:
- Add cross-service integration test harness with real Postgres + ZK HTTP server + gRPC handlers (currently service-layer with fakes/mocks)
- Add repository-backed remote-voting integration tests (current remote tests are service-layer)

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

**Status (2026-03-19):** ✅ **COMPLETE**.

Implemented now:
- Refined Terraform modules with specific bare-metal local storage provisioning rules (`kubernetes_storage_class`) and network privacy (`kubernetes_network_policy`).
- Refined Helm manifests with correct liveness/readiness probes (using `/metrics`), HPA across all 11 core services, and persistent volumes (`volumeClaimTemplates` added to statefulsets) and an initial `ingress` strategy for `gateway`.

**Files structured:**
```
deploy/
  helm/
    indis/
      Chart.yaml
      values.yaml           — default values (image tags, replicas, resource limits)
      values-prod.yaml      — production overrides
      templates/
        identity/           — Deployment, Service, ConfigMap
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
        infra/              — PostgreSQL, Redis, Kafka standalone templates
  terraform/
    main.tf                 — infrastructure (k8s namespace, helm release anchor)
    variables.tf
    outputs.tf
```

---

### T3.9 — CI/CD Pipeline (Self-Hosted GitLab)

**Status (2026-03-19):** ✅ **COMPLETE**.

Implemented now:
- Generated comprehensive `.gitlab-ci.yml` defining pipeline stages supporting multi-language.
- Linting, testing, building for Go, Rust, and Python services.
- Added Trivy container scanning and gosec analysis.
- Includes deployment pipeline using Helm across Staging/Prod contexts.

**Files created:**
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

## Recent Updates / به‌روزرسانی‌های اخیر
- 2026-03-19: Completed T3.8 (Kubernetes Deployment Refinements). Added PVCs for stateful infrastructure, HPAs for scalability, readiness/liveness probes, and an ingress template. Added bare-metal storage and network policies via Terraform.
- 2026-03-19: Completed T3.9 (CI/CD Pipeline). Created comprehensive `.gitlab-ci.yml` supporting multi-language (Go, Rust, Python) build, testing, linting, security scans (Trivy, Gosec), and deployment steps to Helm environments.

---

*نسخه: ۱.۱ | تاریخ: ۲۵۸۵/۱۲ | IranProsperityProject.org*
