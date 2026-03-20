/**
 * INDIS — k6 Load Test: Enrollment Submission
 *
 * Tests the enrollment API under load for the standard pathway.
 * PRD §4.1 target: 500K enrollments/day ≈ 5.8 req/s average; burst to 50 req/s.
 *
 * Run:
 *   k6 run tests/load/k6/enrollment_load.js
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate } from "k6/metrics";

export const options = {
  stages: [
    { duration: "1m", target: 50 },   // ramp to burst
    { duration: "3m", target: 50 },   // sustain burst
    { duration: "30s", target: 6 },   // step down to average
    { duration: "2m", target: 6 },    // sustain average
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"],  // enrollment is heavier; allow 500 ms
    http_req_failed:   ["rate<0.005"], // < 0.5% errors
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const API_KEY  = __ENV.API_KEY  || "dev-load-test-key";

const enrollErrors = new Rate("enroll_errors");

export default function () {
  const payload = JSON.stringify({
    national_id_hash: `sha256:load-test-${__VU}-${__ITER}`,
    pathway: "standard",
    biometric_hash: "sha256:biometric-placeholder",
    document_type: "national_card",
  });
  const params = {
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": API_KEY,
    },
    timeout: "10s",
  };

  const res = http.post(`${BASE_URL}/v1/enrollments`, payload, params);
  const ok = check(res, {
    "accepted": (r) => r.status === 202 || r.status === 200 || r.status === 409,
  });
  enrollErrors.add(!ok);
  sleep(0.1);
}

export function handleSummary(data) {
  return {
    "tests/load/k6/results/enrollment_summary.json": JSON.stringify(data, null, 2),
  };
}
