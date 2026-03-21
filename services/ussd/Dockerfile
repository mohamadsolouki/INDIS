# ── Build stage ──────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app ./cmd/server

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 indis

COPY --from=builder /app /usr/local/bin/app

USER indis
ENTRYPOINT ["/usr/local/bin/app"]
