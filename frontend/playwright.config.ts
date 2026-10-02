import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  timeout: 30000,
  use: {
    baseURL: "http://localhost:3000",
    channel: "chrome",
    headless: true,
    timezoneId: "Asia/Bangkok",
    screenshot: "only-on-failure",
  },
});
