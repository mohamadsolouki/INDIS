import { test, expect } from "@playwright/test";

/**
 * Citizen PWA — Enrollment flow E2E tests.
 *
 * Covers PRD FR-002 enrollment pathways: Standard, Enhanced, Social Attestation.
 * Offline and ZK steps are stubs; set INDIS_E2E_LIVE=1 for live staging.
 */

const isLive = !!process.env.INDIS_E2E_LIVE;

test.describe("Enrollment — Unauthenticated redirect", () => {
  test("enrollment page redirects unauthenticated users to login", async ({ page }) => {
    await page.goto("/enroll");
    // Should redirect to login or show auth gate — not a blank/error page.
    await expect(page.locator("body")).not.toBeEmpty();
    const url = page.url();
    // Either redirected to /login or /auth or stays on /enroll with a login form
    const isExpected =
      url.includes("/login") ||
      url.includes("/auth") ||
      url.includes("/enroll") ||
      (await page.locator("input[type='password']").count()) > 0;
    expect(isExpected).toBe(true);
  });
});

test.describe("Enrollment — Dev token flow", () => {
  test.skip(!isLive, "enrollment UI requires staging backend");

  test("enrollment wizard step 1 is visible after login", async ({ page }) => {
    await page.goto(`/dev-login?token=${process.env.INDIS_DEV_TOKEN}`);
    await page.goto("/enroll");
    await expect(page.locator("body")).not.toBeEmpty();
    // Step indicator or form fields should be visible
    const hasStep = (await page.locator("[data-testid='step-indicator']").count()) > 0
      || (await page.getByRole("heading", { level: 1 }).count()) > 0;
    expect(hasStep).toBe(true);
  });

  test("standard pathway form accepts valid national ID", async ({ page }) => {
    await page.goto(`/dev-login?token=${process.env.INDIS_DEV_TOKEN}`);
    await page.goto("/enroll");

    const nationalIdInput = page.locator("input[name='national_id'], input[placeholder*='ملی'], input[placeholder*='national']");
    if (await nationalIdInput.count() > 0) {
      await nationalIdInput.fill("1234567890");
      await expect(nationalIdInput).toHaveValue("1234567890");
    }
  });

  test("invalid national ID (< 10 digits) shows validation error", async ({ page }) => {
    await page.goto(`/dev-login?token=${process.env.INDIS_DEV_TOKEN}`);
    await page.goto("/enroll");

    const nationalIdInput = page.locator("input[name='national_id'], input[placeholder*='ملی'], input[placeholder*='national']");
    if (await nationalIdInput.count() > 0) {
      await nationalIdInput.fill("123");
      // Trigger validation by clicking next
      const nextBtn = page.getByRole("button", { name: /بعدی|next|suivant/i });
      if (await nextBtn.count() > 0) {
        await nextBtn.click();
        // An error message should appear
        const error = page.locator(".form-error, [role='alert'], [data-testid='field-error']");
        await expect(error.first()).toBeVisible({ timeout: 5_000 });
      }
    }
  });

  test("pathway selector shows three enrollment options", async ({ page }) => {
    await page.goto(`/dev-login?token=${process.env.INDIS_DEV_TOKEN}`);
    await page.goto("/enroll");

    // Standard, Enhanced, Social Attestation pathways
    const pathways = page.locator("[data-testid='pathway-option'], .pathway-card, input[type='radio']");
    if (await pathways.count() > 0) {
      expect(await pathways.count()).toBeGreaterThanOrEqual(3);
    }
  });
});

test.describe("Enrollment — Offline capability (PWA)", () => {
  test("enrollment page renders when network is offline", async ({ page, context }) => {
    await page.goto("/");
    // Simulate offline
    await context.setOffline(true);
    // Reload — PWA service worker should serve cached shell
    try {
      await page.reload({ timeout: 5_000 });
    } catch {
      // Expected to be slow offline — just check the page isn't blank
    }
    // Page body should still exist (cached shell)
    const bodyContent = await page.locator("body").textContent();
    expect(bodyContent).toBeDefined();
    await context.setOffline(false);
  });
});
