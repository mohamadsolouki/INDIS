# INDIS — Verifier Terminal Application

> Application for service counters and verification terminals

## Verification Display (PRD §FR-013)

Verifier terminals display **ONLY a binary result** for ZK-proof verifications:
- ✅ Green screen — APPROVED / تأیید شد
- ❌ Red screen — DENIED / رد شد

**No citizen data is ever displayed to the verifier.**

## Verification Tiers (PRD §2.1.3)

| Level | Method | Data Revealed |
|-------|--------|---------------|
| Level 1 | QR scan + ZK proof | Boolean result only |
| Level 2 | NFC + biometric match | Credential category + validity |
| Level 3 | Full identity check | Full identity (with citizen consent) |
| Level 4 | Emergency override | Full identity (requires senior auth + audit alert) |
