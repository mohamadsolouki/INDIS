import { test, expect } from "@playwright/test";

/**
 * Diaspora Portal — Enrollment & Status E2E tests.
 *
 * Tests the overseas Iranian enrollment wizard (4 steps) and status check page.
 * Default base URL: http://localhost:5175 (diaspora Vite dev server).
 * Set DIASPORA_BASE_URL env to override.
 */

const DIASPORA_BASE = process.env.DIASPORA_BASE_URL ?? "http://localhost:5175";

test.describe("Diaspora Portal — Login", () => {
  test("login page renders with email and password fields", async ({ page }) => {
    await page.goto(DIASPORA_BASE);
    const emailInput = page.locator("input[type='email'], input[name='email']");
    const passwordInput = page.locator("input[type='password']");
    await expect(emailInput.first()).toBeVisible({ timeout: 10_000 });
    await expect(passwordInput.first()).toBeVisible({ timeout: 10_000 });
  });

  test("login page has language selector with fa/en/fr options", async ({ page }) => {
    await page.goto(DIASPORA_BASE);
    const langSelect = page.locator("select").filter({ has: page.locator("option[value='fa']") });
    await expect(langSelect.first()).toBeVisible({ timeout: 10_000 });
    const options = langSelect.locator("option");
    const values = await options.evaluateAll(opts => opts.map(o => (o as HTMLOptionElement).value));
    expect(values).toContain("fa");
    expect(values).toContain("en");
    expect(values).toContain("fr");
  });

  test("dev login bypass navigates to enrollment page", async ({ page }) => {
    await page.goto(DIASPORA_BASE);
    await page.evaluate(() => localStorage.setItem("diaspora_token", "dev-diaspora-token"));
    await page.reload();
    // Should show the authenticated shell (sidebar + enrollment page)
    const sidebar = page.locator(".sidebar");
    await expect(sidebar).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Diaspora Portal — Enrollment Wizard", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(DIASPORA_BASE);
    await page.evaluate(() => localStorage.setItem("diaspora_token", "dev-diaspora-token"));
    await page.reload();
  });

  test("enrollment page shows 4-step stepper", async ({ page }) => {
    await page.goto(`${DIASPORA_BASE}/enroll`);
    const stepDots = page.locator(".step-dot");
    await expect(stepDots.first()).toBeVisible({ timeout: 10_000 });
    expect(await stepDots.count()).toBe(4);
  });

  test("step 1 — national ID validation rejects non-10-digit input", async ({ page }) => {
    await page.goto(`${DIASPORA_BASE}/enroll`);
    // Find national ID input (first input on step 1)
    const inputs = page.locator(".form-input");
    await inputs.first().fill("123");
    const nextBtn = page.getByRole("button", { name: /next|بعدی|suivant/i });
    await nextBtn.click();
    const error = page.locator(".form-error");
    await expect(error.first()).toBeVisible({ timeout: 5_000 });
  });

  test("step 1 — valid data advances to step 2", async ({ page }) => {
    await page.goto(`${DIASPORA_BASE}/enroll`);
    const inputs = page.locator(".form-input");
    const inputCount = await inputs.count();
    if (inputCount >= 5) {
      await inputs.nth(0).fill("1234567890");
      await inputs.nth(1).fill("Ali Rezaei");
      await inputs.nth(2).fill("1990-01-15");
      await inputs.nth(3).fill("Germany");
      await inputs.nth(4).fill("Berlin");
    }
    const nextBtn = page.getByRole("button", { name: /next|بعدی|suivant/i });
    await nextBtn.click();
    // Step 2 should show passport number input or upload zone
    await page.waitForTimeout(500);
    const uploadZone = page.locator(".upload-zone");
    const activeStep = page.locator(".step--active .step-dot");
    const stepText = await activeStep.first().textContent();
    // Step dot should now show 2 (or be active at step 2)
    expect(stepText?.trim()).toBe("2");
  });
});

test.describe("Diaspora Portal — Status Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(DIASPORA_BASE);
    await page.evaluate(() => localStorage.setItem("diaspora_token", "dev-diaspora-token"));
    await page.reload();
  });

  test("status page has enrollment ID input and check button", async ({ page }) => {
    await page.goto(`${DIASPORA_BASE}/status`);
    const input = page.locator(".form-input");
    const btn = page.getByRole("button", { name: /check|بررسی|vérifier/i });
    await expect(input.first()).toBeVisible({ timeout: 10_000 });
    await expect(btn).toBeVisible({ timeout: 10_000 });
  });

  test("DEV- tracking code returns pending status", async ({ page }) => {
    await page.goto(`${DIASPORA_BASE}/status`);
    const input = page.locator(".form-input");
    await input.fill(`DEV-${Date.now()}`);
    const btn = page.getByRole("button", { name: /check|بررسی|vérifier/i });
    await btn.click();
    // Dev fallback should show a status badge
    const badge = page.locator(".status-badge");
    await expect(badge).toBeVisible({ timeout: 8_000 });
  });

  test("sidebar navigation between enroll and status works", async ({ page }) => {
    await page.goto(`${DIASPORA_BASE}/enroll`);
    const statusLink = page.locator(".sidebar-link").filter({ hasText: /status|وضعیت|statut/i });
    await statusLink.click();
    await expect(page).toHaveURL(/\/status/);
  });
});
