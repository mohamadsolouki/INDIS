/**
 * INDIS — k6 Load Test: Credential Verification (PRD §4.1 target: 556 req/s)
 *
 * Simulates the peak load scenario for Level-1 ZK-proof verification:
 *   - 556 virtual users ramp up over 2 minutes
 *   - Sustained at 556 VUs for 5 minutes
 *   - Ramp down over 1 minute
 *
 * PRD performance targets:
 *   - p95 response time < 200 ms
 *   - Error rate < 0.1%
 *   - Throughput ≥ 556 req/s
 *
 * Run:
 *   k6 run tests/load/k6/verify_load.js
 *   k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6/verify_load.js
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

export const options = {
  stages: [
    { duration: "2m", target: 556 },  // ramp up to PRD peak
    { duration: "5m", target: 556 },  // sustain peak
    { duration: "1m", target: 0 },    // ramp down
  ],
  thresholds: {
    http_req_duration: ["p(95)<200"],   // PRD §4.1: p95 < 200 ms
    http_req_failed:   ["rate<0.001"],  // < 0.1% error rate
    verify_errors:     ["rate<0.001"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const API_KEY  = __ENV.API_KEY  || "dev-load-test-key";

const verifyErrors = new Rate("verify_errors");
const verifyDuration = new Trend("verify_duration_ms", true);

// Pre-generated test ZK proof token (base64 encoded; backend accepts it in dev mode).
const TEST_PROOF_TOKEN = __ENV.TEST_PROOF_TOKEN || "dGVzdC1wcm9vZi10b2tlbg==";

export default function () {
  const url = `${BASE_URL}/v1/verify`;
  const payload = JSON.stringify({
    proof_token: TEST_PROOF_TOKEN,
    verifier_did: "did:indis:verifier:load-test",
    claims: ["age_over_18", "citizenship"],
  });
  const params = {
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": API_KEY,
    },
    timeout: "5s",
  };

  const start = Date.now();
  const res = http.post(url, payload, params);
  verifyDuration.add(Date.now() - start);

  const ok = check(res, {
    "status 200": (r) => r.status === 200,
    "result present": (r) => {
      try {
        return JSON.parse(r.body).result !== undefined;
      } catch {
        return false;
      }
    },
  });

  verifyErrors.add(!ok);
  sleep(0); // no think time — measure raw throughput
}

export function handleSummary(data) {
  return {
    "tests/load/k6/results/verify_summary.json": JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: " ", enableColors: true }),
  };
}

// Inline minimal text summary (avoids external import in offline environments).
function textSummary(data, opts) {
  const metrics = data.metrics;
  const p95 = metrics.http_req_duration?.values?.["p(95)"]?.toFixed(2) ?? "N/A";
  const rps = metrics.http_reqs?.values?.rate?.toFixed(1) ?? "N/A";
  const errRate = ((metrics.http_req_failed?.values?.rate ?? 0) * 100).toFixed(3);
  return [
    "─── INDIS Verify Load Test Summary ───",
    `  p95 latency : ${p95} ms  (target < 200 ms)`,
    `  throughput  : ${rps} req/s  (target ≥ 556)`,
    `  error rate  : ${errRate}%  (target < 0.1%)`,
    "───────────────────────────────────────",
  ].join("\n");
}
