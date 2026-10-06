import { test, expect } from "@playwright/test";
// Mock only test requests: never register the first member in the user's database.
test("register, validate passwords, log out and log in", async ({ page }) => {
  let authenticated = false;
  let registered = false;
  const user = {
    id: "test-member",
    name: "ผู้ทดสอบ",
    email: "test@example.test",
    created_at: "2026-10-06T00:00:00Z",
  };
  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    let status = 200;
    let body: unknown;
    if (path === "/api/v1/auth/me") {
      status = authenticated ? 200 : 401;
      body = authenticated ? user : { error: { message: "กรุณาเข้าสู่ระบบ" } };
    } else if (path === "/api/v1/auth/register") {
      const input = route.request().postDataJSON();
      expect(input.name).toBe(user.name);
      expect(input.email).toBe(user.email);
      registered = authenticated = true;
      status = 201;
      body = user;
    } else if (path === "/api/v1/auth/logout") {
      authenticated = false;
      status = 204;
    } else if (path === "/api/v1/auth/login") {
      const input = route.request().postDataJSON();
      authenticated =
        registered &&
        input.email === user.email &&
        input.password === "Test!1234";
      status = authenticated ? 200 : 401;
      body = authenticated
        ? user
        : { error: { message: "อีเมลหรือรหัสผ่านไม่ถูกต้อง" } };
    } else if (path === "/api/v1/matches/today") {
      body = { items: [], total: 0, offset: 0, limit: 200 };
    } else {
      status = 404;
      body = { error: { message: "ไม่มี fixture สำหรับ test" } };
    }
    await route.fulfill({
      status,
      contentType: "application/json",
      ...(status === 204 ? {} : { body: JSON.stringify(body) }),
    });
  });
  await page.goto("/");
  await expect(page).toHaveURL(/\/login$/);
  await page
    .getByRole("button", { name: "ยังไม่มีบัญชี? สมัครสมาชิก", exact: true })
    .click();
  await page.getByLabel("ชื่อที่แสดง").fill(user.name);
  await page.getByLabel("อีเมล", { exact: true }).fill(user.email);
  await page.getByLabel("รหัสผ่าน", { exact: true }).fill("ภาษาไทยทดสอบ");
  await page.getByLabel("ยืนยันรหัสผ่าน").fill("ภาษาไทยทดสอบ");
  await page.getByRole("button", { name: "สมัครสมาชิก", exact: true }).click();
  expect(
    await page
      .getByLabel("รหัสผ่าน", { exact: true })
      .evaluate((input) => (input as HTMLInputElement).checkValidity()),
  ).toBe(false);
  expect(registered).toBe(false);
  await page.getByLabel("รหัสผ่าน", { exact: true }).fill("Test!1234");
  await page.getByLabel("ยืนยันรหัสผ่าน").fill("different-password");
  await page.getByRole("button", { name: "สมัครสมาชิก", exact: true }).click();
  await expect(page.locator(".auth-panel").getByRole("alert")).toHaveText(
    "รหัสผ่านทั้งสองช่องไม่ตรงกัน",
  );
  expect(registered).toBe(false);
  await page.getByLabel("ยืนยันรหัสผ่าน").fill("Test!1234");
  await page.getByRole("button", { name: "สมัครสมาชิก", exact: true }).click();
  await expect(page.getByRole("heading", { name: "สนามวันนี้" })).toBeVisible();
  await expect(page.locator(".member-menu")).toContainText(user.name);
  await page.reload();
  await expect(page.getByRole("heading", { name: "สนามวันนี้" })).toBeVisible();
  await page.getByRole("button", { name: "ออกจากระบบ", exact: true }).click();
  await expect(page).toHaveURL(/\/login$/);
  await page.getByLabel("อีเมล", { exact: true }).fill(user.email);
  await page.getByLabel("รหัสผ่าน", { exact: true }).fill("wrong-password");
  await page.getByRole("button", { name: "เข้าสู่ระบบ", exact: true }).click();
  await expect(page.locator(".auth-panel").getByRole("alert")).toHaveText(
    "อีเมลหรือรหัสผ่านไม่ถูกต้อง",
  );
  await page.getByLabel("รหัสผ่าน", { exact: true }).fill("Test!1234");
  await page.getByRole("button", { name: "เข้าสู่ระบบ", exact: true }).click();
  await expect(page.getByRole("heading", { name: "สนามวันนี้" })).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "ออกจากระบบ", exact: true }).click();
  await expect(page).toHaveURL(/\/login$/);
  await page.screenshot({
    path: "test-results/member-mobile.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
