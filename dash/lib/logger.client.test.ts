/**
 * dash/lib/logger.client.test.ts
 * Tests for the client-side console logger implementation.
 */
import { beforeEach, describe, expect, it, vi } from "vitest";
import { clientLogger } from "./logger.client";

describe("clientLogger", () => {
  beforeEach(() => {
    vi.spyOn(console, "info").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(console, "debug").mockImplementation(() => {});
  });

  it("should log info to console", () => {
    clientLogger.info("test message");
    // This will fail with the stub because it doesn't call console.info
    expect(console.info).toHaveBeenCalled();
  });
});
