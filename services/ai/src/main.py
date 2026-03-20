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
from fastapi.responses import JSONResponse

from biometric.router import router as biometric_router, _service as _biometric_service

# Startup flag: set to a non-None error string on init failure, empty string on success.
_biometric_init_error: str = ""

try:
    # Verify the biometric service initialised by exercising its hyperplane setup.
    # _service is constructed at import time in biometric.router; if that import
    # succeeded and the hyperplanes are populated we consider the module ready.
    if not _biometric_service._hyperplanes:
        _biometric_init_error = "biometric hyperplanes not initialised"
except Exception as exc:  # pragma: no cover
    _biometric_init_error = str(exc)

app = FastAPI(
    title="INDIS AI/ML Service",
    description="سرویس هوش مصنوعی سیستم هویت دیجیتال ملی ایران",
    version="0.1.0",
)

app.include_router(biometric_router)


@app.get("/health")
async def health_check() -> dict:
    """Health check endpoint."""
    return {"status": "healthy", "service": "indis-ai"}


@app.get("/readiness")
async def readiness_check() -> JSONResponse:
    """Readiness check — verifies the biometric deduplication module initialised."""
    if _biometric_init_error:
        return JSONResponse(
            status_code=503,
            content={"ready": False, "error": _biometric_init_error},
        )
    return JSONResponse(
        status_code=200,
        content={"ready": True, "model": "perceptual_hash_lsh"},
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
