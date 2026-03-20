/**
 * INDIS — k6 Load Test: Credential Issuance
 *
 * Tests the credential issuance API under concurrent load.
 * PRD §4.1 target: support 10K credentials issued per hour ≈ 2.8 req/s; burst to 30 req/s.
 *
 * Run:
 *   k6 run tests/load/k6/credential_issue_load.js
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

export const options = {
  stages: [
    { duration: "1m", target: 30 },
    { duration: "3m", target: 30 },
    { duration: "30s", target: 3 },
    { duration: "1m", target: 3 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<1000"], // issuance includes signing; allow 1 s
    http_req_failed:   ["rate<0.005"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const API_KEY  = __ENV.API_KEY  || "dev-load-test-key";

const issueErrors = new Rate("issue_errors");
const issueDuration = new Trend("issue_duration_ms", true);

export default function () {
  const payload = JSON.stringify({
    subject_did: `did:indis:citizen:load-test-${__VU}`,
    credential_type: "NationalIdentityCredential",
    claims: {
      name: "Test User",
      age_over_18: true,
      citizen: true,
    },
  });
  const params = {
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": API_KEY,
    },
    timeout: "15s",
  };

  const start = Date.now();
  const res = http.post(`${BASE_URL}/v1/credentials`, payload, params);
  issueDuration.add(Date.now() - start);

  const ok = check(res, {
    "issued or queued": (r) => r.status === 201 || r.status === 202,
  });
  issueErrors.add(!ok);
  sleep(0.2);
}

export function handleSummary(data) {
  return {
    "tests/load/k6/results/credential_issue_summary.json": JSON.stringify(data, null, 2),
  };
}
