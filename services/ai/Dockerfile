# ── Build stage ──────────────────────────────────────────────
FROM python:3.11-slim AS builder

WORKDIR /build
COPY pyproject.toml .
RUN pip install --no-cache-dir --prefix=/install .

# ── Runtime stage ────────────────────────────────────────────
FROM python:3.11-slim

RUN adduser --disabled-password --uid 1000 indis
COPY --from=builder /install /usr/local
COPY src/ /app/src/

WORKDIR /app
USER indis

ENTRYPOINT ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8000"]
