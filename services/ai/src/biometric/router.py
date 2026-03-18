"""FastAPI router for biometric endpoints."""

from __future__ import annotations

import base64
import time

from fastapi import APIRouter, HTTPException

from .dedup import DeduplicationService
from .models import DeduplicationRequest, DeduplicationResponse

router = APIRouter(prefix="/v1/biometric", tags=["biometric"])
_service = DeduplicationService()


@router.post("/deduplicate", response_model=DeduplicationResponse)
async def deduplicate(req: DeduplicationRequest) -> DeduplicationResponse:
    """Check whether the submitted biometric template is a duplicate."""
    try:
        template_bytes = base64.b64decode(req.template_data_b64, validate=True)
    except Exception as exc:  # pragma: no cover - defensive branch
        raise HTTPException(status_code=400, detail="invalid template_data_b64") from exc

    started = time.perf_counter()
    is_duplicate, confidence, matched_did = _service.check_duplicate(template_bytes)
    elapsed_ms = int((time.perf_counter() - started) * 1000)

    return DeduplicationResponse(
        is_duplicate=is_duplicate,
        confidence=confidence,
        matched_did=matched_did,
        deduplication_ms=str(elapsed_ms),
    )
