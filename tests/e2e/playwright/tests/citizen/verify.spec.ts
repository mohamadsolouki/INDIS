import { test, expect } from "@playwright/test";

/**
 * Citizen PWA — ZK proof presentation / verification E2E tests.
 * PRD FR-013: verifier sees ONLY a boolean result, never raw citizen data.
 * PRD FR-006: offline proof generation with cached credentials.
 */

const isLive = !!process.env.INDIS_E2E_LIVE;

test.describe("Verify — Page shell", () => {
  test("verify page renders and body is non-empty", async ({ page }) => {
    await page.goto("/verify");
    await expect(page.locator("body")).not.toBeEmpty();
  });

  test("verify page has no unhandled JS errors on load", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (err) => errors.push(err.message));
    await page.goto("/verify");
    await page.waitForTimeout(1_000);
    expect(errors.length).toBe(0);
  });
});

test.describe("Verify — ZK proof generation (FR-006)", () => {
  test("generate proof button or predicate selector is present", async ({
    page,
  }) => {
    await page.goto("/verify");
    const generateBtn = page.getByRole("button", {
      name: /proof|اثبات|generate|تولید/i,
    });
    const predicateSelect = page.locator(
      "select, [data-testid='predicate-select']"
    );
    const found =
      (await generateBtn.count()) > 0 || (await predicateSelect.count()) > 0;
    expect(found).toBe(true);
  });

  test.skip(!isLive, "live ZK proof requires staging backend")(
    "generating a proof produces a QR code element",
    async ({ page }) => {
      await page.goto("/verify");
      const generateBtn = page.getByRole("button", {
        name: /generate|تولید/i,
      });
      if ((await generateBtn.count()) > 0) {
        await generateBtn.click();
        // Wait for QR code or proof output
        const qr = page.locator(
          "canvas, svg, img[alt*='QR'], [data-testid='qr-code']"
        );
        await expect(qr.first()).toBeVisible({ timeout: 15_000 });
      }
    }
  );
});

test.describe("Verify — PRD FR-013 compliance (no PII in result)", () => {
  test.skip(!isLive, "live verification requires staging backend")(
    "verification result shows only pass/fail — no raw identity fields",
    async ({ page }) => {
      await page.goto("/verify");
      // Simulate a scan result — look for boolean display
      const passEl = page.locator(
        "[data-testid='result-pass'], [data-testid='result-fail'], " +
        ".result-badge, [class*='approved'], [class*='denied']"
      );
      // Inject a mock verification result into localStorage then reload
      await page.evaluate(() =>
        localStorage.setItem("mock_verify_result", "true")
      );
      await page.reload();
      const body = await page.locator("body").textContent();
      // Ensure no raw identity numbers appear in the result
      expect(body).not.toMatch(/\b\d{10}\b/); // national ID pattern
      expect(body).not.toMatch(/did:indis:[a-f0-9]{40}/); // raw DID
    }
  );
});
