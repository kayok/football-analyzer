import { test, expect } from "@playwright/test";
import { loginTestMember } from "./login";
test.beforeEach(async ({ page }) => {
  test.skip(
    !process.env.E2E_EMAIL || !process.env.E2E_PASSWORD,
    "Set E2E_EMAIL and E2E_PASSWORD for an existing dedicated test account",
  );
  await loginTestMember(page);
});

for (const timezoneId of ["Asia/Bangkok", "America/New_York"]) {
  test.describe(`hydration in ${timezoneId}`, () => {
    test.use({ timezoneId });

    test("initial page loads and match details have no hydration diagnostics", async ({
      page,
    }) => {
      const hydrationErrors: string[] = [];
      const pattern =
        /hydrat|server rendered HTML|server-rendered HTML|didn't match|did not match/i;
      page.on("console", (message) => {
        if (message.type() === "error" && pattern.test(message.text())) {
          hydrationErrors.push(message.text());
        }
      });
      page.on("pageerror", (error) => {
        if (pattern.test(error.message)) hydrationErrors.push(error.message);
      });

      let detailURL: string | null = null;
      for (const route of ["/", "/picks", "/history"]) {
        await page.goto(route);
        await expect(page.locator("main h1")).toBeVisible();
        await expect(page.getByRole("status")).toHaveCount(0);
        if (route === "/") {
          detailURL = await page
            .locator(".detail-link")
            .first()
            .getAttribute("href");
        }
        expect(hydrationErrors, `Hydration errors on ${route}`).toEqual([]);
      }
      expect(
        detailURL,
        "Seed data must provide a match detail link",
      ).not.toBeNull();
      await page.goto(detailURL!);
      await expect(
        page.getByRole("heading", { name: "ข้อมูลโมเดล" }),
      ).toBeVisible();
      expect(hydrationErrors, "Hydration errors on match detail").toEqual([]);
    });
  });
}
