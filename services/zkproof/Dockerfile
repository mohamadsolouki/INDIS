# ── Build stage ──────────────────────────────────────────────
FROM rust:1.75-alpine AS builder

RUN apk add --no-cache musl-dev

WORKDIR /build
COPY . .
RUN cargo build --release --bin zkproof-server

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 indis

COPY --from=builder /build/target/release/zkproof-server /usr/local/bin/zkproof-server

USER indis
ENTRYPOINT ["/usr/local/bin/zkproof-server"]
