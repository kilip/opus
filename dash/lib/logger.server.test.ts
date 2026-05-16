/**
 * dash/lib/logger.server.test.ts
 * Tests for the server-side Pino logger implementation.
 */
import { describe, expect, it } from "vitest";
import { serverLogger } from "./logger.server";

describe("serverLogger", () => {
  it("should have all logger methods defined", () => {
    expect(serverLogger.info).toBeDefined();
    expect(serverLogger.warn).toBeDefined();
    expect(serverLogger.error).toBeDefined();
    expect(serverLogger.debug).toBeDefined();
  });
});
