# INDIS — AI/ML Service

> Biometric deduplication and fraud detection (Python)

## Quick Start

```bash
cd services/ai
pip install -e ".[dev]"
uvicorn src.main:app --reload
```

## Structure

```
services/ai/
├── src/
│   ├── __init__.py
│   ├── main.py               # FastAPI entrypoint
│   ├── biometric/             # Deduplication models
│   └── fraud/                 # Fraud detection
├── tests/
├── pyproject.toml
├── Dockerfile
└── README.md
```
