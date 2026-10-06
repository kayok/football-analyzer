import { test, expect } from "@playwright/test";
import { loginTestMember } from "./login";
test.beforeEach(async ({ page }) => {
  test.skip(
    !process.env.E2E_EMAIL || !process.env.E2E_PASSWORD,
    "Set E2E_EMAIL and E2E_PASSWORD for an existing dedicated test account",
  );
  await loginTestMember(page);
});
test("Today → select → My Picks → cancel → History and Match Detail", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "สนามวันนี้" })).toBeVisible();
  const card = page
    .locator("article.match-card")
    .filter({ hasText: "Barcelona" });
  await expect(card.getByText("WATCH", { exact: true })).toBeVisible();
  await page.screenshot({
    path: "test-results/today-desktop.png",
    fullPage: true,
  });
  await card.getByRole("button", { name: "เลือกเล่น", exact: true }).click();
  await expect(
    card.getByRole("button", { name: "เลือกแล้ว ✓" }),
  ).toBeDisabled();
  await page
    .getByRole("link", { name: "รายการที่เลือก", exact: false })
    .first()
    .click();
  const pick = page
    .locator("article.pick-row")
    .filter({ hasText: "Barcelona" });
  await expect(pick.getByText("ราคา ณ เลือก")).toBeVisible();
  await pick.getByRole("button", { name: "ยกเลิกรายการ" }).click();
  await expect(pick).toHaveCount(0);
  await page
    .getByRole("link", { name: "ประวัติและผลลัพธ์", exact: false })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "ประวัติและผลลัพธ์" }),
  ).toBeVisible();
  await expect(
    page.getByRole("cell", { name: "ยกเลิกรายการ", exact: true }).first(),
  ).toBeVisible();
  await page
    .getByRole("link", { name: "Barcelona vs Atlético Madrid", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "ประวัติ prediction" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "รายชื่อผู้เล่น · ข้อมูลจำลอง" }),
  ).toBeVisible();
  await page.screenshot({
    path: "test-results/detail-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "สนามวันนี้" })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "test-results/today-mobile.png",
    fullPage: true,
  });
});
test("PASS has no pick action", async ({ page }) => {
  await page.goto("/");
  const card = page
    .locator("article.match-card")
    .filter({ hasText: "Juventus" });
  await expect(card.getByText("PASS", { exact: true })).toBeVisible();
  await expect(card.getByRole("button", { name: "เลือกเล่น" })).toHaveCount(0);
});
