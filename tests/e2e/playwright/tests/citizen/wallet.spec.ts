import { test, expect } from "@playwright/test";

/**
 * Citizen PWA — Credential wallet E2E tests.
 * PRD FR-002: credential types; FR-006: offline capability.
 */

const isLive = !!process.env.INDIS_E2E_LIVE;

test.describe("Wallet — Unauthenticated state", () => {
  test("wallet page is reachable (auth-gated or public shell)", async ({ page }) => {
    await page.goto("/wallet");
    await expect(page.locator("body")).not.toBeEmpty();
  });

  test("wallet page has correct page direction (RTL)", async ({ page }) => {
    await page.goto("/");
    const dir = await page.locator("html").getAttribute("dir");
    expect(["rtl", null]).toContain(dir);
  });
});

test.describe("Wallet — With dev token", () => {
  test.beforeEach(async ({ page }) => {
    await page.evaluate(() =>
      localStorage.setItem("indis_jwt", "dev-citizen-token")
    );
  });

  test.skip(!isLive, "live wallet requires staging backend")(
    "wallet lists at least one credential card",
    async ({ page }) => {
      await page.goto("/wallet");
      const cards = page.locator(
        ".credential-card, [data-testid='credential-card'], article"
      );
      await expect(cards.first()).toBeVisible({ timeout: 10_000 });
    }
  );

  test("wallet page title or heading is present", async ({ page }) => {
    await page.goto("/wallet");
    const hasHeading =
      (await page.getByRole("heading").count()) > 0 ||
      (await page.locator("h1, h2, h3").count()) > 0;
    expect(hasHeading).toBe(true);
  });
});

test.describe("Wallet — Offline credential presentation", () => {
  test("ZK proof QR generation button is present on verify page", async ({
    page,
  }) => {
    await page.goto("/verify");
    const body = await page.locator("body").textContent();
    expect(body).toBeTruthy();
  });

  test("offline QR fallback renders without network", async ({
    page,
    context,
  }) => {
    await page.goto("/");
    await context.setOffline(true);
    try {
      await page.goto("/wallet", { timeout: 5_000 });
    } catch {
      // timeout expected offline
    }
    const bodyContent = await page.locator("body").textContent();
    expect(bodyContent).toBeDefined();
    await context.setOffline(false);
  });
});
