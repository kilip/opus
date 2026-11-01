import { describe, expect, it } from "bun:test";
import { runMigrations } from "@opus/db";
import { auth } from "./auth";
import { authEnv } from "./env";

describe("packages/auth", () => {
  it("has valid auth environment variables", () => {
    expect(authEnv.BETTER_AUTH_SECRET).toBeDefined();
    expect(authEnv.BETTER_AUTH_URL).toBeDefined();
  });

  it("handles auth requests via auth.handler", async () => {
    runMigrations();

    const request = new Request("http://localhost:3000/api/auth/ok", {
      method: "GET",
    });

    const response = await auth.handler(request);
    expect(response.status).toBe(200);
    const data = await response.json();
    expect(data).toHaveProperty("ok", true);
  });

  it("triggers email OTP flow", async () => {
    runMigrations();

    const testEmail = `otp-test-${Date.now()}@example.com`;
    const request = new Request("http://localhost:3000/api/auth/email-otp/send-verification-otp", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        email: testEmail,
        type: "sign-in",
      }),
    });

    const response = await auth.handler(request);
    expect(response.status).toBe(200);
    const data = await response.json();
    expect(data).toHaveProperty("success", true);
  });
});
