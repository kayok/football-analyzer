import { expect, type Page } from "@playwright/test";
export async function loginTestMember(page: Page) {
  await page.goto("/login");
  await page.getByLabel("อีเมล", { exact: true }).fill(process.env.E2E_EMAIL!);
  await page
    .getByLabel("รหัสผ่าน", { exact: true })
    .fill(process.env.E2E_PASSWORD!);
  await page.getByRole("button", { name: "เข้าสู่ระบบ", exact: true }).click();
  await expect(page.getByRole("heading", { name: "สนามวันนี้" })).toBeVisible();
}
