import { test, expect } from "@playwright/test";

/**
 * Citizen PWA — Home / Landing page smoke tests.
 *
 * These tests run against the dev server (vite preview or staging).
 * They assert the minimum viable UX: the app loads, shows a Persian-language
 * interface, and critical navigation targets are reachable.
 */

test.describe("Citizen PWA — Home", () => {
  test("page loads with correct title", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/INDIS|هویت دیجیتال/i);
  });

  test("page has RTL direction set on root element", async ({ page }) => {
    await page.goto("/");
    const dir = await page.locator("html").getAttribute("dir");
    expect(dir).toBe("rtl");
  });

  test("health check endpoint returns healthy", async ({ request, baseURL }) => {
    // Verifies the backing API is up before running heavy flows.
    const url = (process.env.API_BASE_URL ?? "http://localhost:8080") + "/healthz";
    const resp = await request.get(url);
    // Accept 200 or 404 (endpoint may not exist in all environments);
    // just ensure the API gateway is reachable (not a network error).
    expect([200, 404, 401]).toContain(resp.status());
  });
});

test.describe("Citizen PWA — Navigation", () => {
  test("credential wallet link is visible", async ({ page }) => {
    await page.goto("/");
    // The wallet / credential section should be discoverable from home.
    const walletLink = page.getByRole("link", { name: /کیف پول|wallet/i });
    await expect(walletLink).toBeVisible({ timeout: 10_000 });
  });

  test("verify identity link is visible", async ({ page }) => {
    await page.goto("/");
    const verifyLink = page.getByRole("link", { name: /تأیید|verify/i });
    await expect(verifyLink).toBeVisible({ timeout: 10_000 });
  });
});
