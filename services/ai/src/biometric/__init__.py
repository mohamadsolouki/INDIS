"""Biometric deduplication module.

Handles fingerprint, facial recognition, and iris matching
for national-scale deduplication during enrollment.

Standards (PRD §FR-004):
- False Match Rate: ≤ 0.0001% (ISO/IEC 29794)
- False Non-Match Rate: ≤ 0.1% (ISO/IEC 29794)
- Liveness Detection (IAPAR): ≤ 0.5% (ISO/IEC 30107-3)

SDK candidates (PRD §6.1):
- NIST BOZORTH3 (fingerprint matching)
- OpenFace / ArcFace (facial recognition)
"""
