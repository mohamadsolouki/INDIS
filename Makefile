# ============================================================
# INDIS — Iran National Digital Identity System
# Top-level Makefile
# ============================================================

.PHONY: all build test lint clean proto-gen docker-build dev-up dev-down dev-seed dev-token migrate help

# Go services (order: shared packages first, then services)
GO_SERVICES := identity credential enrollment biometric audit notification electoral justice gateway verifier govportal ussd card
RUST_SERVICES := zkproof
PYTHON_SERVICES := ai

# ── Help ─────────────────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ── Build ────────────────────────────────────────────────────
all: build ## Build everything

build: build-go build-rust build-python build-frontend ## Build all services

build-go: ## Build all Go services
	@echo "▸ Building Go services..."
	@for svc in $(GO_SERVICES); do \
		echo "  → services/$$svc"; \
		cd services/$$svc && go build ./... && cd ../..; \
	done

build-rust: ## Build Rust ZK proof service
	@echo "▸ Building Rust ZK proof service..."
	cd services/zkproof && cargo build

build-python: ## Validate Python AI service
	@echo "▸ Checking Python AI service..."
	cd services/ai && python3 -m py_compile src/main.py

# ── Test ─────────────────────────────────────────────────────
test: test-go test-rust test-python ## Run all tests

test-go: ## Run Go tests
	@echo "▸ Running Go tests..."
	@for svc in $(GO_SERVICES); do \
		echo "  → services/$$svc"; \
		cd services/$$svc && go test ./... && cd ../..; \
	done

test-rust: ## Run Rust tests
	@echo "▸ Running Rust tests..."
	cd services/zkproof && cargo test

test-python: ## Run Python tests
	@echo "▸ Running Python tests..."
	cd services/ai && .venv/bin/pytest tests/ -v

# ── Lint ─────────────────────────────────────────────────────
lint: lint-go lint-rust lint-python ## Lint all code

lint-go: ## Lint Go code
	@echo "▸ Linting Go..."
	@for svc in $(GO_SERVICES); do \
		cd services/$$svc && golangci-lint run ./... && cd ../..; \
	done

lint-rust: ## Lint Rust code
	@echo "▸ Linting Rust..."
	cd services/zkproof && cargo clippy -- -D warnings

lint-python: ## Lint Python code
	@echo "▸ Linting Python..."
	cd services/ai && .venv/bin/ruff check src/

# ── Protobuf ────────────────────────────────────────────────
proto-gen: ## Generate code from protobuf definitions
	@echo "▸ Generating protobuf code..."
	@./scripts/proto-gen.sh

# ── Docker ───────────────────────────────────────────────────
docker-build: ## Build all Docker images
	@echo "▸ Building Docker images..."
	@for svc in $(GO_SERVICES); do \
		docker build -t indis-$$svc:dev -f services/$$svc/Dockerfile services/$$svc; \
	done
	docker build -t indis-zkproof:dev -f services/zkproof/Dockerfile services/zkproof
	docker build -t indis-ai:dev -f services/ai/Dockerfile services/ai

# ── Dev Environment ──────────────────────────────────────────
dev-up: ## Start local development environment
	docker compose up -d

dev-down: ## Stop local development environment
	docker compose down

dev-seed: ## Seed local database with test data for frontend development
	@echo "▸ Seeding development database..."
	GOWORK=off go run tools/devtoken/main.go --help 2>/dev/null || true
	GOWORK=off go run tools/seed/main.go

dev-token: ## Generate a development JWT for local testing
	@echo "▸ Generating dev JWT..."
	GOWORK=off go run tools/devtoken/main.go $(ARGS)

# ── Database Migrations ─────────────────────────────────────
migrate: ## Run SQL migrations (requires DATABASE_URL; optional MIGRATIONS_DIR)
	@if [ -z "$$DATABASE_URL" ]; then \
		echo "DATABASE_URL is required"; \
		exit 1; \
	fi
	cd pkg/migrate && go run ./cmd/indis-migrate --database-url "$$DATABASE_URL" $${MIGRATIONS_DIR:+--migrations-dir "$$MIGRATIONS_DIR"}

# ── Frontend ─────────────────────────────────────────────────
build-frontend: ## Build all frontend apps (requires Node.js)
	@echo "▸ Building citizen PWA..."
	cd clients/pwa && npm install --silent && npm run build
	@echo "▸ Building verifier terminal..."
	cd clients/verifier && npm install --silent && npm run build
	@echo "▸ Building gov portal..."
	cd clients/gov-portal && npm install --silent && npm run build

dev-pwa: ## Start citizen PWA dev server (port 5173)
	cd clients/pwa && npm install && npm run dev

dev-verifier: ## Start verifier terminal dev server (port 5174)
	cd clients/verifier && npm install && npm run dev -- --port 5174

dev-gov-portal: ## Start gov portal dev server (port 5175)
	cd clients/gov-portal && npm install && npm run dev -- --port 5175

# ── Clean ────────────────────────────────────────────────────
clean: ## Clean build artifacts
	@echo "▸ Cleaning..."
	@for svc in $(GO_SERVICES); do \
		cd services/$$svc && go clean && cd ../..; \
	done
	cd services/zkproof && cargo clean
	rm -rf bin/ out/ tmp/
