"""Biometric deduplication service.

Uses a multi-scale perceptual hash embedding with SimHash-based Locality-Sensitive
Hashing (LSH) for efficient approximate nearest-neighbour search.

Architecture:
  1. _embed()   — converts raw biometric bytes into a 256-dim feature vector using
                  multi-scale frequency analysis, byte distribution statistics, and
                  autocorrelation. Far more discriminative than the previous 64-dim
                  byte-accumulation approach.
  2. _simhash() — projects the feature vector onto 64 random hyperplanes to produce
                  a compact 64-bit SimHash fingerprint.  Near-duplicates share many
                  matching bits (Hamming distance < threshold).
  3. Two-stage check — fast SimHash Hamming screening (<= 10 bits) followed by
                  exact cosine similarity on full vectors for confirmed candidates.

Not a production ML pipeline (no deep biometric model), but provides meaningful
false-match discrimination for development and end-to-end enrollment testing.

Production replacement: FaceNet / ArcFace embedding from a GPU-accelerated model
server, with Faiss ANN index for sub-millisecond population-scale search.
"""

from __future__ import annotations

import hashlib
import math
import struct
from dataclasses import dataclass, field
from typing import List, Tuple

# Tunable thresholds.
_SIMHASH_BITS = 64
_SIMHASH_MAX_HAMMING = 10      # max bit-distance to consider as a candidate
_COSINE_DUPLICATE_THRESHOLD = 0.93   # final confirmation threshold
_FEATURE_DIMS = 256            # embedding dimensionality


@dataclass
class _StoredTemplate:
    vector: List[float]
    simhash: int          # 64-bit SimHash fingerprint
    matched_did: str


class DeduplicationService:
    """Multi-scale perceptual hash deduplication with SimHash LSH pre-filter."""

    def __init__(
        self,
        duplicate_threshold: float = _COSINE_DUPLICATE_THRESHOLD,
        simhash_max_hamming: int = _SIMHASH_MAX_HAMMING,
    ) -> None:
        self._duplicate_threshold = duplicate_threshold
        self._simhash_max_hamming = simhash_max_hamming
        self._templates: List[_StoredTemplate] = []
        # Deterministic random hyperplanes for SimHash projection.
        self._hyperplanes: List[List[float]] = self._init_hyperplanes(_FEATURE_DIMS, _SIMHASH_BITS)

    # ── Public API ──────────────────────────────────────────────────────────

    def check_duplicate(self, template_bytes: bytes) -> Tuple[bool, float, str]:
        """Return (is_duplicate, confidence, matched_did).

        Confidence is in [0, 1].  For non-duplicates, confidence is the highest
        similarity score seen (useful for audit trails).  matched_did is the DID
        of the matching record when is_duplicate=True.
        """
        if not template_bytes:
            return False, 0.0, ""

        vector = self._embed(template_bytes)
        fingerprint = self._simhash(vector)

        best_score = 0.0
        best_did = ""

        for stored in self._templates:
            # Stage 1: fast Hamming distance check on SimHash fingerprints.
            if self._hamming(fingerprint, stored.simhash) > self._simhash_max_hamming:
                continue
            # Stage 2: exact cosine similarity on full vectors.
            score = self._cosine_similarity(vector, stored.vector)
            if score > best_score:
                best_score = score
                best_did = stored.matched_did

        if best_score >= self._duplicate_threshold:
            return True, best_score, best_did

        # Not a duplicate — register this template.
        did_suffix = hashlib.sha256(template_bytes).hexdigest()[:20]
        new_did = f"did:indis:{did_suffix}"
        self._templates.append(
            _StoredTemplate(vector=vector, simhash=fingerprint, matched_did=new_did)
        )
        return False, best_score, ""

    # ── Embedding ───────────────────────────────────────────────────────────

    def _embed(self, data: bytes) -> List[float]:
        """256-dimensional feature vector from multi-scale analysis of raw bytes.

        Features (64 dims each × 4 scales = 256 total):
          - Scale 0: byte value distribution histogram (global)
          - Scale 1: 8-byte block mean + variance (structural)
          - Scale 2: first-order differences (gradient/edge density)
          - Scale 3: autocorrelation at lags 1,2,4,8 (periodicity/texture)
        """
        dims = _FEATURE_DIMS // 4
        f0 = self._byte_histogram(data, dims)
        f1 = self._block_statistics(data, dims)
        f2 = self._gradient_features(data, dims)
        f3 = self._autocorrelation_features(data, dims)
        vec = f0 + f1 + f2 + f3
        return self._l2_normalize(vec)

    def _byte_histogram(self, data: bytes, dims: int) -> List[float]:
        """Normalised histogram of byte values bucketed into `dims` bins."""
        hist = [0.0] * dims
        bucket_size = max(1, 256 // dims)
        for b in data:
            hist[b // bucket_size] += 1.0
        total = sum(hist) or 1.0
        return [x / total for x in hist]

    def _block_statistics(self, data: bytes, dims: int) -> List[float]:
        """Mean and variance of non-overlapping 8-byte blocks."""
        block = 8
        stats: List[float] = []
        for i in range(0, len(data), block):
            chunk = data[i : i + block]
            if not chunk:
                break
            mean = sum(chunk) / len(chunk)
            var = sum((b - mean) ** 2 for b in chunk) / len(chunk)
            stats.append(mean / 255.0)
            stats.append(math.sqrt(var) / 128.0)
        # Resize to exactly `dims` by padding / truncating.
        return self._resize(stats, dims)

    def _gradient_features(self, data: bytes, dims: int) -> List[float]:
        """First-order absolute differences (edge density proxy)."""
        if len(data) < 2:
            return [0.0] * dims
        diffs = [abs(int(data[i + 1]) - int(data[i])) / 255.0 for i in range(len(data) - 1)]
        return self._resize(diffs, dims)

    def _autocorrelation_features(self, data: bytes, dims: int) -> List[float]:
        """Autocorrelation at lags {1, 2, 4, 8, 16, …} (periodicity detection)."""
        n = len(data)
        if n < 2:
            return [0.0] * dims
        mean = sum(data) / n
        variance = sum((b - mean) ** 2 for b in data) / n or 1.0
        lags = []
        lag = 1
        while lag < n and len(lags) < dims:
            cov = sum((data[i] - mean) * (data[i + lag] - mean) for i in range(n - lag))
            lags.append(cov / ((n - lag) * variance))
            lag *= 2
        return self._resize(lags, dims)

    # ── SimHash ─────────────────────────────────────────────────────────────

    def _init_hyperplanes(self, dim: int, n_planes: int) -> List[List[float]]:
        """Deterministic random hyperplanes via seeded SHA-256 PRNG."""
        planes: List[List[float]] = []
        seed = b"indis:simhash:v1"
        for i in range(n_planes):
            # Deterministic per-plane seed.
            h = hashlib.sha256(seed + struct.pack(">I", i)).digest()
            # Extend to `dim` floats using the hash as a PRNG seed.
            plane: List[float] = []
            block_idx = 0
            while len(plane) < dim:
                h2 = hashlib.sha256(h + struct.pack(">I", block_idx)).digest()
                for j in range(0, len(h2), 4):
                    val = struct.unpack(">f", h2[j : j + 4])[0]
                    plane.append(val if math.isfinite(val) else 0.0)
                    if len(plane) >= dim:
                        break
                block_idx += 1
            planes.append(plane[:dim])
        return planes

    def _simhash(self, vector: List[float]) -> int:
        """Compute a 64-bit SimHash fingerprint via random hyperplane projections."""
        fingerprint = 0
        for i, plane in enumerate(self._hyperplanes):
            dot = sum(v * p for v, p in zip(vector, plane))
            if dot >= 0:
                fingerprint |= 1 << i
        return fingerprint

    @staticmethod
    def _hamming(a: int, b: int) -> int:
        """Population count of XOR (Hamming distance)."""
        x = a ^ b
        count = 0
        while x:
            x &= x - 1
            count += 1
        return count

    # ── Utilities ───────────────────────────────────────────────────────────

    @staticmethod
    def _cosine_similarity(a: List[float], b: List[float]) -> float:
        if not a or not b:
            return 0.0
        return sum(x * y for x, y in zip(a, b))  # vectors are pre-normalised

    @staticmethod
    def _l2_normalize(vec: List[float]) -> List[float]:
        norm = math.sqrt(sum(x * x for x in vec))
        if norm < 1e-12:
            return vec
        return [x / norm for x in vec]

    @staticmethod
    def _resize(seq: List[float], target: int) -> List[float]:
        """Pad with zeros or truncate to exactly `target` elements."""
        if len(seq) >= target:
            return seq[:target]
        return seq + [0.0] * (target - len(seq))
