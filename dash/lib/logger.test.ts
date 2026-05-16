/**
 * dash/lib/logger.test.ts
 * Tests for the isomorphic logger interface and routing.
 */
import { describe, expect, it } from "vitest";

describe("isomorphic logger routing", () => {
  it("should export a logger instance", async () => {
    const { logger } = await import("./logger");
    expect(logger).toBeDefined();
    expect(logger.info).toBeTypeOf("function");
    expect(logger.warn).toBeTypeOf("function");
    expect(logger.error).toBeTypeOf("function");
    expect(logger.debug).toBeTypeOf("function");
  });
});
