# INDIS — Local Development Setup Guide

This guide documents everything needed to run and test the full INDIS stack locally — backend services, infrastructure, and all three frontend apps.

---

## Prerequisites

### Required tools

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.22+ | [go.dev/dl](https://go.dev/dl/) — install to `~/go-install/go/` if no sudo |
| Rust + Cargo | 1.75+ | `curl https://sh.rustup.rs -sSf \| sh` |
| Python | 3.11+ | system package or `pyenv` |
| Node.js | 18+ | [nodejs.org](https://nodejs.org/) or `nvm` |
| Docker + Compose | 28+ / v2 | Ubuntu: `sudo apt install docker.io docker-compose-v2` |
| protoc | 29+ | pre-built from github.com/protocolbuffers/protobuf/releases, install to `~/.local/bin/` |
| golangci-lint | 1.64+ | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |

### PATH setup

Add these to `~/.bashrc` (or `~/.profile`) if tools were installed to user-space:

```bash
export PATH=$PATH:$HOME/go-install/go/bin:$HOME/go/bin:$HOME/.local/bin
```

Reload: `source ~/.bashrc`

---

## Project structure overview

```
INDIS/
├── docker-compose.yml           # Infrastructure (Postgres, Redis, Kafka, etc.)
├── docker-compose.override.yml  # Local fixes (Kafka cluster ID — auto-applied)
├── docker-compose.services.yml  # Application services (containerised)
├── Makefile                     # All build/test/run targets
├── go.work                      # Go workspace (all Go modules)
├── .env                         # Local env (copied from .env.example, gitignored)
├── services/                    # Backend microservices
│   ├── identity/    :50051 gRPC  :8080 HTTP  :9101 metrics
│   ├── credential/  :50052 gRPC  :8081 HTTP  :9102 metrics
│   ├── enrollment/  :50053 gRPC  :8082 HTTP  :9103 metrics
│   ├── biometric/   :50054 gRPC              :9104 metrics
│   ├── audit/       :50055 gRPC  :9200 HTTP  :9105 metrics
│   ├── notification/:50056 gRPC              :9106 metrics
│   ├── electoral/   :50057 gRPC  :9116 HTTP  :9107 metrics
│   ├── justice/     :50058 gRPC  :9300 HTTP  :9108 metrics
│   ├── gateway/               :8085 HTTP  :9109 metrics   ← :8085 on this machine
│   ├── verifier/    :9110 HTTP  :9111 metrics
│   ├── govportal/   :8200 HTTP  :8201 metrics
│   ├── ussd/        :8300 HTTP  :8301 metrics
│   ├── card/        :8400 HTTP  :8401 metrics
│   ├── zkproof/     :8088 HTTP  :8089 metrics             (Rust)
│   └── ai/          :8000 HTTP  :8001 metrics             (Python)
├── pkg/                         # Shared Go libraries
├── db/migrations/               # SQL migrations (001–011)
├── api/proto/                   # Protobuf definitions
├── api/gen/go/                  # Generated Go stubs (never edit)
├── clients/
│   ├── web/citizen-pwa/         # Citizen PWA (React + Vite)  → :5173
│   ├── verifier/                # Verifier terminal PWA        → :5174
│   └── gov-portal/              # Gov portal (React + GraphQL) → :5175
├── tools/devtoken/              # Dev JWT generator
└── tools/seed/                  # Dev data seeder
```

---

## Machine-specific port notes

This machine has other services running that occupy standard ports. The INDIS dev setup is adjusted accordingly:

| Port conflict | Occupied by | INDIS uses instead |
|---------------|-------------|-------------------|
| 5432 (Postgres) | `zkcn-postgres` container | **5435** |
| 8080 (HTTP) | `wp-geocheckout` (WordPress) | **8085** (gateway) |

These overrides are already baked into:

- `docker-compose.yml` — Postgres bound to 5435
- `.env` — `DATABASE_URL` uses 5435, `HTTP_PORT=8085`
- `clients/*/  .env.local` — `VITE_GATEWAY_URL=http://localhost:8085`

If you're on a clean machine without conflicts, revert those to 5432 / 8080.

---

## Quick start (first time)

### 1. Clone and enter the repo

```bash
git clone https://github.com/mohamadsolouki/INDIS.git
cd INDIS
```

### 2. Copy the env file

```bash
cp .env.example .env
# On this machine the .env already has port fixes applied.
# On a fresh machine with no port conflicts no changes needed.
```

### 3. Start infrastructure containers

```bash
make dev-up
```

Starts: PostgreSQL (:5435), Redis (:6379), Kafka (:9092), CouchDB (:5984), Prometheus (:9090), Grafana (:3000), Jaeger (:16686).

Verify all 7 are healthy:

```bash
docker ps --filter "name=indis-" --format "table {{.Names}}\t{{.Status}}"
```

### 4. Run database migrations

```bash
DATABASE_URL="postgres://indis:indis_dev_password@localhost:5435/indis_identity?sslmode=disable" \
  make migrate
```

Output: `migrations applied successfully from .../db/migrations`

Runs once; re-run only after adding new migration files.

### 5. Build all services

```bash
make build-go       # All 13 Go services
make build-rust     # ZK proof service (Rust)
make build-python   # Python AI service (syntax check)
```

### 6. Install frontend dependencies

```bash
cd clients/web/citizen-pwa && npm install && cd ../../..
cd clients/verifier         && npm install && cd ../..
cd clients/gov-portal       && npm install && cd ../..
```

Or use the Makefile (does all three):

```bash
make build-frontend
```

---

## Running the full stack locally

### Step 1 — Start backend services

Open a terminal for each service (or use tmux/a process manager). First export the common env block:

```bash
export DATABASE_URL="postgres://indis:indis_dev_password@localhost:5435/indis_identity?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export KAFKA_BROKERS="localhost:9092"
export GRPC_TLS_MODE=plaintext
export BACKEND_TLS_MODE=plaintext
export BLOCKCHAIN_TYPE=mock
export HSM_BACKEND=software
export LOG_LEVEL=info
export JWT_SECRET=indis-dev-secret-change-in-prod
export HTTP_PORT=8085          # gateway only; others use their own defaults
```

Then start each service from the repo root:

```bash
(cd services/identity    && go run ./cmd/server) &
(cd services/credential  && ZKPROOF_URL=http://localhost:8088 go run ./cmd/server) &
(cd services/enrollment  && go run ./cmd/server) &
(cd services/biometric   && AI_SERVICE_URL=http://localhost:8000 go run ./cmd/server) &
(cd services/audit       && go run ./cmd/server) &
(cd services/notification && go run ./cmd/server) &
(cd services/electoral   && ZKPROOF_URL=http://localhost:8088 go run ./cmd/server) &
(cd services/justice     && go run ./cmd/server) &
(cd services/verifier    && IDENTITY_ADDR=localhost:50051 CREDENTIAL_ADDR=localhost:50052 ZKPROOF_URL=http://localhost:8088 go run ./cmd/server) &
(cd services/govportal   && IDENTITY_ADDR=localhost:50051 CREDENTIAL_ADDR=localhost:50052 ENROLLMENT_ADDR=localhost:50053 AUDIT_ADDR=localhost:50055 ELECTORAL_ADDR=localhost:50057 JUSTICE_ADDR=localhost:50058 go run ./cmd/server) &
(cd services/ussd        && IDENTITY_ADDR=localhost:50051 CREDENTIAL_ADDR=localhost:50052 go run ./cmd/server) &
(cd services/card        && IDENTITY_ADDR=localhost:50051 CREDENTIAL_ADDR=localhost:50052 go run ./cmd/server) &
(cd services/gateway     && \
  IDENTITY_ADDR=localhost:50051 CREDENTIAL_ADDR=localhost:50052 \
  ENROLLMENT_ADDR=localhost:50053 BIOMETRIC_ADDR=localhost:50054 \
  AUDIT_ADDR=localhost:50055 NOTIFICATION_ADDR=localhost:50056 \
  ELECTORAL_ADDR=localhost:50057 JUSTICE_ADDR=localhost:50058 \
  VERIFIER_HTTP_URL=http://localhost:9110 CARD_HTTP_URL=http://localhost:8400 \
  USSD_HTTP_URL=http://localhost:8300 GOV_PORTAL_HTTP_URL=http://localhost:8200 \
  CORS_ALLOWED_ORIGINS="*" API_KEYS=dev-api-key-1 RATE_LIMIT_RPS=200 \
  go run ./cmd/server) &
```

### Step 2 — Start the Rust ZK proof service

```bash
cd services/zkproof
RUST_LOG=info HTTP_PORT=8088 cargo run --bin zkproof-server
```

### Step 3 — Start the Python AI service

```bash
cd services/ai
# First time only:
python3 -m venv .venv
.venv/bin/pip install -e ".[dev]"
# Run:
.venv/bin/uvicorn src.main:app --host 0.0.0.0 --port 8000 --reload
```

### Step 4 — Start the frontend dev servers

Each in a separate terminal from the repo root:

```bash
# Citizen PWA — http://localhost:5173
make dev-pwa

# Verifier terminal — http://localhost:5174
make dev-verifier

# Gov portal — http://localhost:5175
make dev-gov-portal
```

Each Vite dev server proxies `/v1` and `/graphql` to the gateway at `http://localhost:8085` (via `.env.local` in each client directory).

---

## Running services in Docker (full containerised stack)

The `docker-compose.services.yml` builds and runs all backend services as Docker containers.

> **Note:** On this machine port 8080 is occupied. If you use the services compose file, the gateway container will fail to bind 8080. Add a port override or stop the conflicting container first.

```bash
# Build all service images (takes several minutes first time):
make docker-build

# Start infra + all services:
docker compose -f docker-compose.yml -f docker-compose.services.yml up -d

# Tear down services only (keep infra):
docker compose -f docker-compose.services.yml down

# Full teardown including infra:
make dev-down && docker compose -f docker-compose.services.yml down
```

---

## Frontend apps

### Citizen PWA — `clients/web/citizen-pwa/`

React + Vite + Tailwind + PWA. RTL-first (Persian/Farsi).

```bash
# Dev server (hot reload, proxies API to gateway)
make dev-pwa          # → http://localhost:5173

# Production build
cd clients/web/citizen-pwa && npm run build
# Output: clients/web/citizen-pwa/dist/

# Lint
cd clients/web/citizen-pwa && npm run lint
```

**Pages:** Login, Home (dashboard), Wallet (credentials), Enrollment wizard (3 pathways), Verify (ZK proof presentation), Privacy (consent log), Settings.

**Key features being tested:**

- Login → generates Ed25519 key pair stored in IndexedDB, obtains JWT
- Wallet → shows verifiable credentials fetched from gateway
- Enrollment → 5-step wizard (pathway → doc capture → biometric → review → success)
- Verify → QR scan of incoming ZK proof request, approve/deny

**Dev login:** any DID string works; the gateway validates JWTs signed with `indis-dev-secret-change-in-prod`. Or generate a token with:

```bash
go run tools/devtoken/main.go --role citizen --did did:indis:test-001
```

Then paste it in the Login page's "Dev Token" field (visible when `import.meta.env.DEV` is true).

---

### Verifier Terminal — `clients/verifier/`

React + Vite + PWA. QR-code scanner for verifier operators.

```bash
make dev-verifier       # → http://localhost:5174

cd clients/verifier && npm run build
# Output: clients/verifier/dist/
```

**Pages:** Login, Scan (camera QR), Result (boolean pass/fail ZK output), History.

**Dev login:** uses the same gateway JWT. Generate a verifier token:

```bash
go run tools/devtoken/main.go --role verifier --did did:indis:verifier-001
```

---

### Gov Portal — `clients/gov-portal/`

React + Vite + Apollo GraphQL. Ministry and admin dashboard.

```bash
make dev-gov-portal     # → http://localhost:5175

cd clients/gov-portal && npm run build
# Output: clients/gov-portal/dist/
```

**Pages:** Login, Dashboard, Users, Enrollment Review, Credential Issuance, Electoral Authority, Audit Log, Transitional Justice, Bulk Operations.

**Dev login:** admin or ministry token:

```bash
go run tools/devtoken/main.go --role admin
go run tools/devtoken/main.go --role ministry --ministry MOI
```

---

## Testing

### Backend unit and integration tests

```bash
make test-go      # All Go service tests
make test-rust    # Rust ZK proof tests
make test-python  # Python AI tests (requires .venv in services/ai/)
```

Single service:

```bash
cd services/identity   && go test ./...
cd services/credential && go test ./...
cd services/zkproof    && cargo test
cd services/ai         && .venv/bin/pytest tests/ -v
```

Tests that touch the database need `DATABASE_URL`:

```bash
cd services/identity && \
  DATABASE_URL="postgres://indis:indis_dev_password@localhost:5435/indis_identity?sslmode=disable" \
  go test ./...
```

### Linting

```bash
make lint-go      # golangci-lint on all Go services
make lint-rust    # cargo clippy -D warnings
make lint-python  # ruff check
```

### Frontend build check (catches TypeScript errors)

```bash
cd clients/web/citizen-pwa && npm run build
cd clients/verifier         && npm run build
cd clients/gov-portal       && npm run build
```

---

## Protobuf code generation

After editing any `.proto` file in `api/proto/`:

```bash
make proto-gen
```

Generated files land in `api/gen/go/{identity,credential,enrollment}/v1/`. Never edit these directly.

---

## Development tooling

### Generate a dev JWT

```bash
# Default: citizen role, 24h expiry, did:indis:dev-test-001
go run tools/devtoken/main.go

# Roles
go run tools/devtoken/main.go --role admin
go run tools/devtoken/main.go --role ministry --ministry MOI
go run tools/devtoken/main.go --role verifier --did did:indis:verifier-001 --expiry 1h
```

### Seed test data into a running stack

Requires the gateway to be up on `:8085`:

```bash
GATEWAY_URL=http://localhost:8085 make dev-seed
```

Creates: 3 test citizens, 1 test election (open), 1 verifier org, 1 test card.

---

## Monitoring

| Dashboard | URL | Credentials |
|-----------|-----|-------------|
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | — |
| Jaeger tracing | http://localhost:16686 | — |
| CouchDB | http://localhost:5984/_utils | admin / adminpassword |

Every Go service exposes Prometheus metrics at `http://localhost:<METRICS_PORT>/metrics`.

Enable distributed tracing:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

---

## Port reference

### Infrastructure (docker-compose.yml)

| Container | Host port | Notes |
|-----------|-----------|-------|
| indis-postgres | **5435** | Remapped from 5432 (conflict) |
| indis-redis | 6379 | |
| indis-kafka | 9092 | |
| indis-couchdb | 5984 | |
| indis-prometheus | 9090 | |
| indis-grafana | 3000 | |
| indis-jaeger | 16686, 4317, 4318 | UI, OTLP gRPC, OTLP HTTP |

### Backend services (local run)

| Service | gRPC | HTTP | Metrics |
|---------|------|------|---------|
| identity | 50051 | 8080 | 9101 |
| credential | 50052 | 8081 | 9102 |
| enrollment | 50053 | 8082 | 9103 |
| biometric | 50054 | — | 9104 |
| audit | 50055 | 9200 | 9105 |
| notification | 50056 | — | 9106 |
| electoral | 50057 | 9116 | 9107 |
| justice | 50058 | 9300 | 9108 |
| **gateway** | — | **8085** | 9109 |
| verifier | — | 9110 | 9111 |
| govportal | — | 8200 | 8201 |
| ussd | — | 8300 | 8301 |
| card | — | 8400 | 8401 |
| zkproof (Rust) | — | 8088 | 8089 |
| ai (Python) | — | 8000 | 8001 |

### Frontend dev servers

| App | URL |
|-----|-----|
| Citizen PWA | <http://localhost:5173> |
| Verifier terminal | <http://localhost:5174> |
| Gov portal | <http://localhost:5175> |

---

## Troubleshooting

### `make dev-up` fails with "port already allocated"

```bash
ss -tlnp | grep <port>
docker ps   # see which container owns it
```

For Postgres: if 5435 is also taken, change `ports: "5435:5432"` in `docker-compose.yml` and update `DATABASE_URL` in `.env`.

For other containers: update the relevant `ports:` entry in `docker-compose.yml`.

### Kafka fails to start ("Cluster ID is not a valid UUID")

`docker-compose.override.yml` fixes this. Make sure it is present with `CLUSTER_ID: um36b869SC2VWN4Y1LXp9A`. If you need a fresh ID:

```bash
python3 -c "import uuid,base64; print(base64.urlsafe_b64encode(uuid.uuid4().bytes).rstrip(b'=').decode())"
```

### Migration fails

```bash
# Drop and recreate the database then re-run:
docker exec -it indis-postgres psql -U indis -c "DROP DATABASE indis_identity;"
docker exec -it indis-postgres psql -U indis -c "CREATE DATABASE indis_identity;"
DATABASE_URL="postgres://indis:indis_dev_password@localhost:5435/indis_identity?sslmode=disable" make migrate
```

### Service fails with "connection refused" on DB/Redis

Make sure env vars use the correct local ports before running a service natively:

```bash
export DATABASE_URL="postgres://indis:indis_dev_password@localhost:5435/indis_identity?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
```

### Frontend shows blank page / API errors

1. Confirm the gateway is running: `curl http://localhost:8085/healthz`
2. Confirm `.env.local` exists in the client directory with `VITE_GATEWAY_URL=http://localhost:8085`
3. Check browser DevTools Network tab — all `/v1/` requests should proxy to the gateway.

### Frontend TypeScript build errors

```bash
# Rebuild after dependency install
cd clients/<app> && npm install && npm run build
```

The common causes are a missing `node_modules/` or a missing `vite-env.d.ts`. Both are present in this repo.

### Python AI service import errors

```bash
cd services/ai
python3 -m venv .venv
.venv/bin/pip install -e ".[dev]"
```

### Go build fails with module errors

Always run from the repo root where `go.work` is located:

```bash
cd /path/to/INDIS
make build-go    # uses go.work workspace automatically
```

---

## Daily workflow

```bash
# Morning: start infra
make dev-up

# Export common env (or add to a sourced file)
export DATABASE_URL="postgres://indis:indis_dev_password@localhost:5435/indis_identity?sslmode=disable"
export REDIS_URL=redis://localhost:6379/0
export KAFKA_BROKERS=localhost:9092
export GRPC_TLS_MODE=plaintext
export BACKEND_TLS_MODE=plaintext
export BLOCKCHAIN_TYPE=mock
export HSM_BACKEND=software
export JWT_SECRET=indis-dev-secret-change-in-prod

# Start whichever backend services you're working on
(cd services/identity && go run ./cmd/server) &
(cd services/gateway  && HTTP_PORT=8085 ... go run ./cmd/server) &

# Start the frontend you're working on
make dev-pwa          # http://localhost:5173

# Run tests
make test-go

# End of day
make dev-down
```
