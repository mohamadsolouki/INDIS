import { defineConfig, devices } from "@playwright/test";

/**
 * INDIS End-to-End Test Suite — Playwright configuration.
 *
 * Covers:
 *   - Citizen PWA (http://localhost:5173)
 *   - Verifier Terminal (http://localhost:5174)
 *   - Diaspora Portal (http://localhost:5175)
 *
 * Run with:
 *   npx playwright test                      # all tests
 *   npx playwright test --project=citizen    # PWA only
 *   npx playwright test --project=verifier   # Verifier Terminal only
 *   npx playwright test --project=diaspora   # Diaspora Portal only
 *
 * The BASE_URL env vars let CI point at deployed staging instances.
 */
export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 4 : undefined,
  reporter: [
    ["html", { outputFolder: "playwright-report", open: "never" }],
    ["junit", { outputFile: "playwright-results.xml" }],
    process.env.CI ? ["github"] : ["list"],
  ],

  use: {
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },

  projects: [
    {
      name: "citizen",
      use: {
        ...devices["Desktop Chrome"],
        baseURL: process.env.CITIZEN_PWA_URL ?? "http://localhost:5173",
      },
      testMatch: "**/citizen/**/*.spec.ts",
    },
    {
      name: "verifier",
      use: {
        ...devices["Desktop Chrome"],
        baseURL: process.env.VERIFIER_URL ?? "http://localhost:5174",
      },
      testMatch: "**/verifier/**/*.spec.ts",
    },
    {
      name: "diaspora",
      use: {
        ...devices["Desktop Chrome"],
        baseURL: process.env.DIASPORA_BASE_URL ?? "http://localhost:5175",
      },
      testMatch: "**/diaspora/**/*.spec.ts",
    },
    // Mobile viewport for citizen PWA (RTL / Persian UI)
    {
      name: "citizen-mobile",
      use: {
        ...devices["Pixel 7"],
        baseURL: process.env.CITIZEN_PWA_URL ?? "http://localhost:5173",
        locale: "fa-IR",
      },
      testMatch: "**/citizen/**/*.spec.ts",
    },
  ],
});
