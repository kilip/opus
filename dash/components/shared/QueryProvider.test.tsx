import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { QueryProvider } from "./QueryProvider";

describe("QueryProvider", () => {
  it("renders children correctly", () => {
    render(
      <QueryProvider>
        <div>Test Child</div>
      </QueryProvider>,
    );
    expect(screen.getByText("Test Child")).toBeInTheDocument();
  });
});
