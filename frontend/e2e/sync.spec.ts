import { test, expect } from "@playwright/test";

for (const outcome of ["succeeded", "failed"]) {
  test(`mobile sync ${outcome}: retains cards and prevents repeats`, async ({
    page,
  }) => {
    let posts = 0;
    let complete = false;
    let reads = 0;
    const retry = new Date(Date.now() + 30 * 60_000).toISOString();
    await page.route("**/api/v1/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      let body: unknown;
      let status = 200;
      if (path === "/api/v1/auth/me")
        body = { id: "owner", name: "เจ้าของ", email: "owner@example.test" };
      else if (path === "/api/v1/sync") {
        if (route.request().method() === "POST") {
          posts++;
          status = 202;
        }
        body = {
          state: posts ? (complete ? outcome : "running") : "idle",
          message: posts
            ? complete
              ? outcome === "succeeded"
                ? "ซิงก์สำเร็จ"
                : "ซิงก์ไม่สำเร็จ ข้อมูลเดิมยังอยู่"
              : "กำลังซิงก์ ข้อมูลเดิมยังแสดงอยู่"
            : "ยังไม่ได้ซิงก์",
          retry_at: posts ? retry : null,
          last_success_at:
            complete && outcome === "succeeded"
              ? new Date().toISOString()
              : null,
          automatic: true,
          next_scheduled_at: retry,
        };
      } else if (path === "/api/v1/matches/today") {
        reads++;
        body = {
          items: [
            {
              id: "m1",
              home:
                complete && outcome === "succeeded"
                  ? "Updated team"
                  : "Original team",
              away: "Away team",
              competition: "Test League",
              kickoff: new Date().toISOString(),
              status: "scheduled",
              recommendation: null,
              pick: null,
              result: null,
            },
          ],
          total: 1,
          offset: 0,
          limit: 200,
        };
      } else {
        status = 404;
        body = {};
      }
      await route.fulfill({
        status,
        contentType: "application/json",
        body: JSON.stringify(body),
      });
    });
    await page.setViewportSize({ width: 320, height: 844 });
    await page.goto("/");
    const card = page.locator("article.match-card");
    const sync = page.getByRole("button", { name: "ซิงก์ข้อมูล", exact: true });
    await expect(card).toContainText("Original team");
    await expect(sync).toBeEnabled();
    await sync.click();
    await expect(
      page.getByRole("button", { name: "กำลังซิงก์…", exact: true }),
    ).toBeDisabled();
    await expect(card).toContainText("Original team");
    expect(posts).toBe(1);
    const readsBefore = reads;
    complete = true;
    await expect(sync).toBeVisible({ timeout: 10_000 });
    await expect(sync).toBeDisabled();
    await expect(card).toContainText(
      outcome === "succeeded" ? "Updated team" : "Original team",
    );
    if (outcome === "succeeded") expect(reads).toBeGreaterThan(readsBefore);
    else expect(reads).toBe(readsBefore);
    expect(posts).toBe(1);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: `test-results/sync-${outcome}-mobile.png`,
      fullPage: true,
    });
  });
}
