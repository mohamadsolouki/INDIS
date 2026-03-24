<!-- GSD:project-start source:PROJECT.md -->
## Project

**INDIS Platform UX and Product Refactor**

This is a phased modernization of the existing INDIS platform to improve usability, trust, and completion rates for high-impact public workflows. The initiative focuses first on citizen-facing enrollment and government-operator decision workflows, while improving backend support for clearer, more reliable experience delivery. The target is a government-premium product quality level rather than a minimal or generic UI.

**Core Value:** Citizens and operators can complete critical identity and credential workflows quickly, confidently, and without confusion.

### Constraints

- **Scope strategy**: Phased upgrade — selected to minimize disruption while improving the highest-impact journeys first
- **Delivery horizon**: 1-2 months — prioritize high-value improvements before broad expansion
- **Quality bar**: Government-premium UX with strong hierarchy, accessibility, and consistency — this is non-negotiable for accepted outcomes
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.22.0 - Core microservices, chaincode, shared libraries, and tooling in `services/*`, `pkg/*`, `chaincode/*`, and `tools/*`.
- Rust 2021 edition - Zero-knowledge proof engine in `services/zkproof/`.
- Python 3.11+ - AI biometric dedup service in `services/ai/`.
- TypeScript 5.x - Web clients and E2E tests in `clients/*` and `tests/e2e/playwright/`.
## Runtime
- Go toolchain 1.22+ from `go.work` and module `go.mod` files.
- Rust toolchain for cargo builds in `services/zkproof/Cargo.toml`.
- Python 3.11+ from `services/ai/pyproject.toml`.
- Node.js 20 in CI from `.github/workflows/e2e.yml`.
- Go modules (`go mod`) with workspace management via `go.work`.
- Cargo for Rust workspace dependencies in `services/zkproof/Cargo.toml`.
- pip/PEP-621 style Python project config in `services/ai/pyproject.toml`.
- npm for frontend clients and Playwright suite in `clients/*/package.json` and `tests/e2e/playwright/package.json`.
- Lockfile: present for Playwright in `tests/e2e/playwright/package-lock.json`; frontend lockfiles vary by subproject.
## Frameworks
- gRPC + protobuf (`google.golang.org/grpc`, generated stubs in `api/gen/go`) for service-to-service communication.
- Axum for Rust ZK HTTP service in `services/zkproof/crates/zkproof-server/src/main.rs`.
- FastAPI for AI service in `services/ai/src/main.py` and deps in `services/ai/pyproject.toml`.
- React 18 + Vite in `clients/gov-portal/`, `clients/verifier/`, `clients/web/citizen-pwa/`, and `clients/web/diaspora/`.
- Go `testing` package for unit/integration tests in `*_test.go` across `pkg/` and `services/`.
- Pytest for AI tests in `services/ai/tests/`.
- Playwright for browser E2E in `tests/e2e/playwright/`.
- Make orchestrates cross-language build/test/lint workflows in `Makefile`.
- Docker Compose for local infra and service orchestration via `docker-compose.yml` and `docker-compose.services.yml`.
## Key Dependencies
- `google.golang.org/grpc` - Internal RPC protocol for Go services.
- `github.com/jackc/pgx/v5` - PostgreSQL access in service repositories.
- `github.com/segmentio/kafka-go` - Event transport used by `pkg/events`.
- `axum`, `serde`, `tokio` - Rust ZK HTTP runtime and serialization.
- `fastapi`, `uvicorn`, `onnxruntime` - AI service API and model runtime.
- Prometheus metrics via shared Go package `pkg/metrics`.
- OpenTelemetry tracing via `pkg/tracing` and Python OTEL packages.
- Redis cache integration via `pkg/cache` and service envs in compose files.
## Configuration
- Environment-variable driven configs in `services/*/internal/config/*.go` and `services/gateway/internal/config/config.go`.
- Infrastructure credentials and endpoints configured in compose files and CI secrets.
- Service transport security toggles (`GRPC_TLS_MODE`, `BACKEND_TLS_MODE`) set in `docker-compose.services.yml`.
- `go.work` coordinates local module replacements.
- Vite and TypeScript configs in frontend roots (`vite.config.ts`, `tsconfig.json`).
- Python lint/test settings in `services/ai/pyproject.toml`.
## Platform Requirements
- Docker + Compose for PostgreSQL, Redis, Kafka, CouchDB, Prometheus, Grafana, Jaeger in `docker-compose.yml`.
- Go, Rust, Python, Node toolchains as listed in `README.md` and used by `Makefile`.
- Kubernetes + Helm deployment patterns in `deploy/helm/` and `.gitlab/ci/deploy-*.yml`.
- Observability stack and infra code in `deploy/prometheus/`, `deploy/grafana/`, and `deploy/terraform/`.
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- Go production files usually follow lowercase descriptive names (`handler.go`, `service.go`, `config.go`).
- Go tests use `*_test.go` suffix.
- React pages/components commonly use PascalCase file names (`EnrollmentReviewPage.tsx`), while utility modules are often lowercase.
- Go uses camelCase/private and PascalCase/exported identifiers.
- Handler methods often use `handleX` naming (`handleIdentity`, `handleCredential`) in gateway.
- Go local variables use short semantic names (`cfg`, `ctx`, `repo`, `svc`).
- Constants use camelCase in Go unless protocol-specific identifiers require uppercase style.
- Go struct and interface names are PascalCase (`Config`, `Gateway`, repository/service interfaces).
- TS types/interfaces follow PascalCase in frontend modules.
## Code Style
- Go code follows gofmt style and idioms.
- TypeScript in `clients/web/citizen-pwa` uses strict TS compiler options (`strict`, `noUnusedLocals`, `noUnusedParameters`) from `clients/web/citizen-pwa/tsconfig.json`.
- Python style is enforced through Ruff config in `services/ai/pyproject.toml` (line length 100, lint groups E/F/I/N/W/UP).
- Go lint command in `Makefile` uses `golangci-lint`.
- Python lint command in `Makefile` uses `ruff check`.
- Frontend lint script present in `clients/web/citizen-pwa/package.json`.
## Import Organization
- Go files generally group imports by category with blank lines.
- TS files typically keep React/framework imports first and local modules after.
- Citizen PWA uses `@/*` alias mapped to `src/*` via `clients/web/citizen-pwa/tsconfig.json`.
## Error Handling
- Go uses explicit error returns and wrapping with context (`fmt.Errorf("...: %w", err)`).
- Config loaders validate env combinations and fail fast.
- Handlers map bad input to HTTP 400 and unsupported methods/routes to 404/405.
- Repository-level sentinel errors appear in service tests and repositories (`ErrNotFound`, `ErrAlreadyRevoked`).
- Service methods map repository and integration failures into domain-level outcomes.
## Logging
- Go services predominantly use standard `log` package.
- Rust service uses `tracing` and subscriber setup.
- Startup logs for service ports, metrics, and dependency health.
- Warning logs for degraded-but-running scenarios (for example gateway privacy DB unavailable).
## Comments
- Comments describe route tables, security posture, and why behavior exists.
- Service code often uses doc comments on exported types/functions.
- TODOs are annotated with production intent, especially for external integrations.
- Examples: notification provider hooks and USSD gateway verification TODO markers.
## Function Design
- Entry-point `main.go` files orchestrate initialization and graceful shutdown; detailed logic is delegated to internal packages.
- Constructor style for dependency injection (`New(repo, ...)`) is common in services.
- Multi-value returns (`value, err`) in Go and explicit tuple-like returns in domain services are common.
## Module Design
- Public APIs are intentionally small at package boundaries (e.g., `config.Load`, `handler.New`, `service.New`).
- Preserve `internal/config`, `internal/repository`, `internal/service`, `internal/handler` boundaries in Go services.
- Use shared cross-cutting concerns from `pkg/` instead of duplicating logic per service.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- Single ingress API gateway routes to many domain services.
- Service code follows layered `config -> repository -> service -> handler -> cmd/server` shape.
- Cross-cutting packages in `pkg/` provide events, metrics, tracing, migration, TLS, crypto, and cache.
- Zero-knowledge proving and verification is delegated to Rust service and consumed by Go services.
## Layers
- Purpose: User-facing routes, auth, rate limiting, CORS, and proxying.
- Contains: Gateway handlers and middleware in `services/gateway/internal/*`.
- Depends on: gRPC client proxies, selected HTTP backend routes, auth middleware.
- Used by: Frontends (`clients/*`), external API consumers, verifier terminals.
- Purpose: Identity, credential, enrollment, electoral, justice, notification, biometric, verifier, govportal, USSD, card logic.
- Contains: service orchestration and transport handlers under `services/*/internal/*`.
- Depends on: repositories, shared packages in `pkg/*`, generated proto stubs in `api/gen/go`.
- Used by: gateway and internal event consumers.
- Purpose: cryptographic proving/verification and biometric dedup ML logic.
- Contains: Rust crates in `services/zkproof/crates/*` and FastAPI app in `services/ai/src/`.
- Depends on: core cryptographic engines and runtime libs.
- Used by: electoral, justice, credential, biometric, and verifier pathways.
- Purpose: persistence, event bus, observability, and blockchain anchors.
- Contains: PostgreSQL/Redis/Kafka wiring in compose and adapters in `pkg/blockchain`, `pkg/events`, `pkg/cache`.
- Used by: all domain services.
## Data Flow
- Durable state in PostgreSQL tables (`db/migrations`).
- Time-bound fast state in Redis.
- Event ordering/propagation via Kafka topics.
## Key Abstractions
- Purpose: isolate persistence logic behind interfaces.
- Examples: `services/identity/internal/repository`, `services/electoral/internal/repository`.
- Pattern: constructor-injected repository implementations for services and tests.
- Purpose: hold domain rules and orchestration.
- Examples: `services/credential/internal/service`, `services/enrollment/internal/service`.
- Pattern: pure Go service structs invoked by handlers.
- Purpose: centralize backend client creation and resilience.
- Examples: `services/gateway/internal/proxy`, `services/gateway/internal/circuitbreaker`.
## Entry Points
- `services/*/cmd/server/main.go`.
- Responsibilities: config load, metrics/tracing init, DB and migration init, transport server start, graceful shutdown.
- `services/zkproof/crates/zkproof-server/src/main.rs`.
- `services/ai/src/main.py`.
- `clients/*/src/main.tsx` and Vite startup scripts in each `package.json`.
## Error Handling
- Guard clauses and config validation in `internal/config/*.go`.
- `context.WithTimeout` around remote calls in handlers and service clients.
- Graceful degradation patterns in gateway when optional DB paths are unavailable.
## Cross-Cutting Concerns
- Standard `log` package used heavily in Go service startup/runtime paths.
- Request decoding + required-field checks in transport handlers.
- Python request model validation via FastAPI/Pydantic.
- JWT + API key middleware at gateway.
- Role-based checks in selected high-risk routes (for example emergency override route in gateway handler).
<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
