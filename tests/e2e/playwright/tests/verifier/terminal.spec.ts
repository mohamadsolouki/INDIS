import { test, expect } from "@playwright/test";

/**
 * Verifier Terminal — E2E tests.
 *
 * Tests PRD FR-012 / FR-013 verifier-side flows:
 *   1. Terminal loads and shows the scan interface
 *   2. QR scan input triggers a verification call
 *   3. Result panel displays a boolean eligibility verdict
 *
 * Live flows require INDIS_E2E_LIVE=1 and a valid verifier API key.
 */

const isLive = !!process.env.INDIS_E2E_LIVE;

test.describe("Verifier Terminal — Smoke", () => {
  test("terminal loads scan interface", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/verifier|تأییدکننده/i);
  });

  test("scan button or input is present", async ({ page }) => {
    await page.goto("/");
    // Accept either a QR scanner widget or a manual token input field.
    const scanTarget = page
      .getByRole("button", { name: /scan|اسکن/i })
      .or(page.getByPlaceholder(/token|کد/i));
    await expect(scanTarget).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Verifier Terminal — Verification Flow", () => {
  test.skip(!isLive, "live verification requires staging backend")(
    "shows PASS result for a valid ZK proof",
    async ({ page }) => {
      await page.goto("/");
      // Paste a pre-generated valid token from the test fixture env var.
      const tokenInput = page.getByPlaceholder(/token|کد/i);
      await tokenInput.fill(process.env.INDIS_TEST_VALID_TOKEN!);
      await page.getByRole("button", { name: /verify|تأیید/i }).click();

      const result = page.locator("[data-testid='verification-result']");
      await expect(result).toBeVisible({ timeout: 15_000 });
      await expect(result).toContainText(/PASS|تأیید شد/i);
    }
  );

  test.skip(!isLive, "live verification requires staging backend")(
    "shows FAIL result for an expired credential",
    async ({ page }) => {
      await page.goto("/");
      const tokenInput = page.getByPlaceholder(/token|کد/i);
      await tokenInput.fill(process.env.INDIS_TEST_EXPIRED_TOKEN!);
      await page.getByRole("button", { name: /verify|تأیید/i }).click();

      const result = page.locator("[data-testid='verification-result']");
      await expect(result).toBeVisible({ timeout: 15_000 });
      await expect(result).toContainText(/FAIL|رد شد/i);
    }
  );
});
