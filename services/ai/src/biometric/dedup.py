"""Minimal biometric deduplication service for development.

This implementation is intentionally simple and uses an in-memory vector store.
It is a non-production placeholder to unblock end-to-end enrollment flow.
"""

from __future__ import annotations

import hashlib
import math
from dataclasses import dataclass
from typing import List, Tuple


@dataclass
class _StoredTemplate:
    vector: List[float]
    matched_did: str


class DeduplicationService:
    """Simple cosine-similarity deduplicator with in-memory storage."""

    def __init__(self, duplicate_threshold: float = 0.97) -> None:
        self._duplicate_threshold = duplicate_threshold
        self._templates: List[_StoredTemplate] = []

    def check_duplicate(self, template_bytes: bytes) -> Tuple[bool, float, str]:
        """Return (is_duplicate, confidence, matched_did)."""
        vector = self._embed(template_bytes)

        best_score = 0.0
        best_did = ""
        for stored in self._templates:
            score = self._cosine_similarity(vector, stored.vector)
            if score > best_score:
                best_score = score
                best_did = stored.matched_did

        is_duplicate = best_score >= self._duplicate_threshold
        if not is_duplicate:
            # Create a deterministic pseudo-DID for development traceability.
            did_suffix = hashlib.sha256(template_bytes).hexdigest()[:20]
            self._templates.append(_StoredTemplate(vector=vector, matched_did=f"did:indis:{did_suffix}"))
            return False, best_score, ""

        return True, best_score, best_did

    def _embed(self, template_bytes: bytes) -> List[float]:
        # Convert bytes into a fixed-length 64-dim normalized vector.
        dims = 64
        vec = [0.0] * dims
        if not template_bytes:
            return vec
        for idx, value in enumerate(template_bytes):
            vec[idx % dims] += float(value) / 255.0

        norm = math.sqrt(sum(x * x for x in vec))
        if norm == 0:
            return vec
        return [x / norm for x in vec]

    def _cosine_similarity(self, a: List[float], b: List[float]) -> float:
        if not a or not b:
            return 0.0
        return sum(x * y for x, y in zip(a, b))
