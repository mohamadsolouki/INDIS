"""Pydantic models for biometric deduplication endpoints."""

from pydantic import BaseModel, Field


class DeduplicationRequest(BaseModel):
    """Request payload for biometric deduplication."""

    enrollment_id: str = Field(min_length=1)
    modality: str = Field(default="unspecified")
    template_data_b64: str = Field(min_length=1)


class DeduplicationResponse(BaseModel):
    """Response payload for biometric deduplication."""

    is_duplicate: bool
    confidence: float
    matched_did: str = ""
    deduplication_ms: str
