# Security Policy / سیاست امنیتی

## Reporting Vulnerabilities / گزارش آسیب‌پذیری‌ها

INDIS takes security extremely seriously. If you discover a security vulnerability, please report it responsibly.

### Process

1. **DO NOT** create a public GitHub issue for security vulnerabilities.
2. Email: `security@indis.ir` (placeholder — to be established)
3. Include a detailed description, reproduction steps, and potential impact.
4. We will acknowledge receipt within **24 hours**.
5. We will provide an initial assessment within **72 hours**.

### Scope

The following are in scope for our security policy:

- All backend services (Go, Rust, Python)
- ZK proof circuits (Circom, Cairo)
- Blockchain abstraction layer and chaincode
- API gateway and authentication
- Mobile and web applications
- Key management infrastructure

### Bug Bounty Program

A public bug bounty program will be established before production launch, as required by the INDIS PRD. Details will be published at `https://indis.ir/security/bounty` (placeholder).

### Standards Compliance

- FIPS 140-2 Level 3 (HSM)
- ISO/IEC 30107-3 (Biometric PAD)
- NIST PQC (Post-quantum readiness)

---

*See [INDIS PRD v1.0 — §4.3 Security Requirements](INDIS_PRD_v1.0.md#43-security-requirements--نیازمندیهای-امنیتی) for the complete security framework.*
