import base64
import sys
from pathlib import Path

from fastapi.testclient import TestClient

sys.path.append(str(Path(__file__).resolve().parents[1] / "src"))

from main import app


def _b64(data: bytes) -> str:
    return base64.b64encode(data).decode("ascii")


def test_deduplicate_success_roundtrip() -> None:
    client = TestClient(app)

    payload = {
        "enrollment_id": "enr-1",
        "modality": "fingerprint",
        "template_data_b64": _b64(b"template-one"),
    }

    resp = client.post("/v1/biometric/deduplicate", json=payload)
    assert resp.status_code == 200
    body = resp.json()

    assert body["is_duplicate"] is False
    assert body["matched_did"] == ""
    assert isinstance(body["confidence"], float)
    assert body["deduplication_ms"].isdigit()


def test_deduplicate_duplicate_detection() -> None:
    client = TestClient(app)
    payload = {
        "enrollment_id": "enr-2",
        "modality": "face",
        "template_data_b64": _b64(b"template-dup"),
    }

    first = client.post("/v1/biometric/deduplicate", json=payload)
    assert first.status_code == 200

    second = client.post("/v1/biometric/deduplicate", json=payload)
    assert second.status_code == 200
    body = second.json()

    assert body["is_duplicate"] is True
    assert body["matched_did"].startswith("did:indis:")


def test_deduplicate_malformed_payload() -> None:
    client = TestClient(app)

    payload = {
        "enrollment_id": "enr-3",
        "modality": "iris",
        "template_data_b64": "***not-base64***",
    }

    resp = client.post("/v1/biometric/deduplicate", json=payload)
    assert resp.status_code == 400
    assert resp.json()["detail"] == "invalid template_data_b64"
