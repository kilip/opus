import { describe, expect, it } from "vitest";
import { cn } from "../utils";

describe("cn", () => {
  it("merges tailwind classes correctly", () => {
    expect(cn("px-2", "py-2")).toBe("px-2 py-2");
  });

  it("handles conditional classes", () => {
    expect(cn("px-2", true && "py-2", false && "m-2")).toBe("px-2 py-2");
  });

  it("handles tailwind conflicts (merging)", () => {
    // tailwind-merge should handle this
    expect(cn("p-4", "p-2")).toBe("p-2");
  });
});
