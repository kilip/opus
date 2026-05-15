import { render, screen } from "@testing-library/react";
import { QueryProvider } from "./QueryProvider";
import { describe, it, expect } from "vitest";

describe("QueryProvider", () => {
  it("renders children correctly", () => {
    render(
      <QueryProvider>
        <div>Test Child</div>
      </QueryProvider>
    );
    expect(screen.getByText("Test Child")).toBeInTheDocument();
  });
});
