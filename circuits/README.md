# INDIS — ZK Circuits

## Circom Circuits (Groth16 / PLONK)

| Circuit | PRD Reference | Purpose |
|---------|---------------|---------|
| `age_proof` | FR-003 | Proves age ≥ threshold without revealing exact age |
| `citizenship_proof` | FR-003 | Proves citizenship without revealing any identifier |
| `voter_eligibility` | FR-003, FR-010 | Atomic proof: citizenship + age + not excluded |
| `credential_validity` | FR-003 | Proves credential is valid, not revoked, not expired |

## STARK Electoral Proof (Rust / Winterfell)

The Cairo electoral proof approach was superseded by a Rust implementation using the
[Winterfell](https://github.com/facebook/winterfell) STARK library (see `services/zkproof/`).

- **`WinterfellStarkEngine`** — `services/zkproof/crates/zkproof-core/src/stark/` — STARK prove/verify engine
- **`VoterEligibilityAir`** — same directory — AIR encoding voter eligibility constraints

**Rationale:** Winterfell is a production-grade, memory-safe Rust STARK library with 95-bit
post-quantum security. It requires no trusted setup (unlike Groth16/PLONK) and is already
integrated end-to-end with `services/electoral`. The `circuits/cairo/` directory was removed.

## Building Circuits

```bash
# Compile a Circom circuit (requires circom 2.0+)
cd circuits/circom/age_proof
circom age_proof.circom --r1cs --wasm --sym

# Generate proving/verification keys (requires snarkjs)
snarkjs groth16 setup age_proof.r1cs pot_final.ptau age_proof.zkey
snarkjs zkey export verificationkey age_proof.zkey verification_key.json
```

## Security Requirements (PRD §FR-003)

- ALL ZK circuit code is **open source and publicly audited**
- Trusted setup (SNARK) uses **multi-party computation** with international observers
- **Formal verification** of circuits required before production deployment
- Circuit audit reports published publicly
