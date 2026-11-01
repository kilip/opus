import { describe, expect, it } from "bun:test";
import { eq } from "drizzle-orm";
import { createDatabase } from "./client";
import { runMigrations } from "./migrate";
import * as schema from "./schema";

describe("packages/db", () => {
  it("initializes an in-memory database and creates tables via migration", () => {
    const memoryDb = createDatabase(":memory:");
    expect(memoryDb).toBeDefined();

    runMigrations(memoryDb);
  });

  it("can insert and query user record", async () => {
    const memoryDb = createDatabase(":memory:");
    runMigrations(memoryDb);

    const testId = `test-user-${Date.now()}`;
    const now = new Date();

    await memoryDb.insert(schema.user).values({
      id: testId,
      name: "Test User",
      email: `${testId}@example.com`,
      emailVerified: true,
      createdAt: now,
      updatedAt: now,
    });

    const users = await memoryDb.select().from(schema.user).where(eq(schema.user.id, testId));

    expect(users).toHaveLength(1);
    expect(users[0]?.name).toBe("Test User");
    expect(users[0]?.email).toBe(`${testId}@example.com`);
  });
});
