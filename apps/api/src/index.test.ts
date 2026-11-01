import { describe, expect, it } from "bun:test";
import { runMigrations } from "@opus/db";
import { app } from "./index";

describe("apps/api", () => {
  it("serves root health endpoint", async () => {
    const response = await app.handle(new Request("http://localhost:3000/"));
    expect(response.status).toBe(200);
    const body = await response.json();
    expect(body).toEqual({ name: "opus-api", status: "ok" });
  });

  it("serves mounted better-auth endpoints", async () => {
    runMigrations();
    const response = await app.handle(new Request("http://localhost:3000/api/auth/ok"));
    expect(response.status).toBe(200);
    const body = await response.json();
    expect(body).toHaveProperty("ok", true);
  });

  it("returns 401 for /api/me when unauthenticated", async () => {
    const response = await app.handle(new Request("http://localhost:3000/api/me"));
    expect(response.status).toBe(401);
    const body = await response.json();
    expect(body).toEqual({ error: "Unauthorized" });
  });
});
