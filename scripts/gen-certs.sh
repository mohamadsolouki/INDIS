#!/usr/bin/env bash
# gen-certs.sh — Generate self-signed CA and per-service TLS certificates for
# INDIS local development.
#
# Usage: bash scripts/gen-certs.sh
#
# Output layout:
#   certs/ca.key          CA private key (4096-bit RSA)
#   certs/ca.crt          Self-signed CA certificate (10-year validity)
#   certs/<svc>.key       Service private key (2048-bit RSA)
#   certs/<svc>.crt       Service certificate signed by the CA (1-year validity)
#
# The script is idempotent: existing certificate files are not regenerated.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CERTS_DIR="${REPO_ROOT}/certs"

CA_KEY="${CERTS_DIR}/ca.key"
CA_CRT="${CERTS_DIR}/ca.crt"
CA_SUBJECT="/C=IR/ST=Tehran/L=Tehran/O=INDIS/OU=PKI/CN=INDIS-Dev-CA"
CA_DAYS=3650   # 10 years

SVC_DAYS=365   # 1 year

SERVICES=(
    identity
    credential
    enrollment
    biometric
    audit
    notification
    electoral
    justice
    gateway
)

# ── Helpers ────────────────────────────────────────────────────────────────────

info()  { printf '\033[0;34m[INFO]\033[0m  %s\n' "$*"; }
ok()    { printf '\033[0;32m[OK]\033[0m    %s\n' "$*"; }
skip()  { printf '\033[0;33m[SKIP]\033[0m  %s\n' "$*"; }

# ── Setup ──────────────────────────────────────────────────────────────────────

mkdir -p "${CERTS_DIR}"
info "Certificate output directory: ${CERTS_DIR}"

# ── CA ─────────────────────────────────────────────────────────────────────────

if [[ -f "${CA_KEY}" && -f "${CA_CRT}" ]]; then
    skip "CA already exists — skipping CA generation"
else
    info "Generating CA private key (4096-bit RSA)…"
    openssl genrsa -out "${CA_KEY}" 4096 2>/dev/null

    info "Generating self-signed CA certificate (${CA_DAYS}-day validity)…"
    openssl req -new -x509 \
        -key  "${CA_KEY}" \
        -out  "${CA_CRT}" \
        -days "${CA_DAYS}" \
        -subj "${CA_SUBJECT}" \
        2>/dev/null

    ok "CA generated: ${CA_CRT}"
fi

# ── Per-service certificates ───────────────────────────────────────────────────

for SVC in "${SERVICES[@]}"; do
    SVC_KEY="${CERTS_DIR}/${SVC}.key"
    SVC_CSR="${CERTS_DIR}/${SVC}.csr"
    SVC_CRT="${CERTS_DIR}/${SVC}.crt"
    SVC_EXT="${CERTS_DIR}/${SVC}.ext"

    if [[ -f "${SVC_KEY}" && -f "${SVC_CRT}" ]]; then
        skip "${SVC}: certificate already exists — skipping"
        continue
    fi

    info "Generating key for service '${SVC}' (2048-bit RSA)…"
    openssl genrsa -out "${SVC_KEY}" 2048 2>/dev/null

    info "Creating CSR for service '${SVC}'…"
    openssl req -new \
        -key  "${SVC_KEY}" \
        -out  "${SVC_CSR}" \
        -subj "/C=IR/ST=Tehran/L=Tehran/O=INDIS/OU=Services/CN=${SVC}" \
        2>/dev/null

    # Write a temporary ext file for SANs
    cat > "${SVC_EXT}" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=DNS:localhost,DNS:${SVC},IP:127.0.0.1
EOF

    info "Signing certificate for '${SVC}' with CA (${SVC_DAYS}-day validity)…"
    openssl x509 -req \
        -in      "${SVC_CSR}" \
        -CA      "${CA_CRT}" \
        -CAkey   "${CA_KEY}" \
        -CAcreateserial \
        -out     "${SVC_CRT}" \
        -days    "${SVC_DAYS}" \
        -extfile "${SVC_EXT}" \
        2>/dev/null

    # Clean up temporary files
    rm -f "${SVC_CSR}" "${SVC_EXT}"

    ok "${SVC}: ${SVC_CRT}"
done

# ── Summary ───────────────────────────────────────────────────────────────────

printf '\n'
printf '═%.0s' {1..60}
printf '\n'
printf '  INDIS Dev Certificate Summary\n'
printf '═%.0s' {1..60}
printf '\n'
printf '  CA certificate : %s\n' "${CA_CRT}"
printf '  CA key         : %s\n' "${CA_KEY}"
printf '\n'
printf '  Service certificates:\n'
for SVC in "${SERVICES[@]}"; do
    printf '    %-14s %s/%s.crt\n' "${SVC}" "${CERTS_DIR}" "${SVC}"
done
printf '═%.0s' {1..60}
printf '\n'
printf '  NOTE: These certificates are for LOCAL DEVELOPMENT only.\n'
printf '        Never use generated keys in a production environment.\n'
printf '═%.0s' {1..60}
printf '\n'
