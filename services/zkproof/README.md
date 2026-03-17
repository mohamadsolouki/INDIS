# INDIS — ZK Proof Service

> Zero-knowledge proof generation and verification service (Rust)

## Proof Systems

| Use Case | System | PRD Reference |
|----------|--------|---------------|
| Standard credential verification | **Groth16** (ZK-SNARK) | FR-003 |
| Electoral / referendum | **ZK-STARK** | FR-003, FR-010 |
| Batch credential operations | **PLONK** | FR-003 |
| Anonymous testimony | **Bulletproofs** | FR-003, FR-011 |

## Workspace Structure

```
services/zkproof/
├── Cargo.toml                    # Workspace root
├── crates/
│   ├── zkproof-server/           # gRPC server (tonic)
│   ├── zkproof-core/             # Core proof logic & traits
│   └── zkproof-circuits/         # Circuit bindings (Circom/Cairo)
├── Dockerfile
└── README.md
```

## Quick Start

```bash
cd services/zkproof
cargo build
cargo test
```
