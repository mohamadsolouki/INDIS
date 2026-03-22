# Contributing to INDIS / مشارکت در INDIS

Thank you for your interest in contributing to the Iran National Digital Identity System.

## Development Setup

### Prerequisites

- **Go** 1.22+
- **Rust** 1.75+ (with `cargo`)
- **Python** 3.11+
- **Docker** & Docker Compose
- **protoc** (Protocol Buffers compiler)
- **Make**

### Getting Started

```bash
# Clone the repository
git clone https://github.com/mohamadsolouki/INDIS.git
cd INDIS

# Start infrastructure services
make dev-up

# Build all services
make build

# Run all tests
make test

# Run linters
make lint
```

## Code Standards

### Go Services
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `golangci-lint` for static analysis
- All exported functions must have doc comments

### Rust (ZK Proof Service)
- Follow the [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)
- Use `cargo clippy` with `-D warnings`
- All `unsafe` blocks require justification comments

### Python (AI/ML Service)
- Follow PEP 8 (enforced via `ruff`)
- Type hints required on all function signatures
- Use `pytest` for testing

### All Languages
- Write tests for all new functionality
- Security-sensitive code requires peer review by 2+ maintainers
- Cryptographic code must reference the standard it implements

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(identity): add DID resolution endpoint
fix(credential): correct revocation timestamp handling
docs(api): update enrollment proto comments
```

## Persian-First / فارسی اول

- All user-facing strings must include Persian translations
- UI designs must be RTL-first
- Documentation should be bilingual where feasible

## Security

Please see [SECURITY.md](SECURITY.md) for reporting vulnerabilities.
