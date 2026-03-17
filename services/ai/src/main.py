"""INDIS AI/ML Service — FastAPI Application.

Provides REST API endpoints for:
- Biometric deduplication (fingerprint, face, iris matching)
- Fraud pattern detection
- On-device inference model serving (ONNX)

Technology stack (PRD §6.1):
- Python + PyTorch (biometric deduplication)
- ONNX Runtime (on-device inference)
- scikit-learn (fraud pattern detection)

Performance targets (PRD §4.1):
- Biometric deduplication: 30s target, 90s max (full national population)
- Bulk enrollment processing: 500K/day peak
"""

from fastapi import FastAPI

app = FastAPI(
    title="INDIS AI/ML Service",
    description="سرویس هوش مصنوعی سیستم هویت دیجیتال ملی ایران",
    version="0.1.0",
)


@app.get("/health")
async def health_check() -> dict:
    """Health check endpoint."""
    return {"status": "healthy", "service": "indis-ai"}


@app.get("/readiness")
async def readiness_check() -> dict:
    """Readiness check — verifies ML models are loaded."""
    # TODO: Check that biometric models are loaded
    # TODO: Check that fraud detection models are loaded
    return {"ready": False, "reason": "models not yet loaded"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
