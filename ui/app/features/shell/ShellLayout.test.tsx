import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { ErrorBoundary as RootErrorBoundary } from "~/root";
import routes from "~/routes";
import { ErrorBoundary as ShellErrorBoundary } from "./ShellLayout";

describe("ErrorBoundary & NotFound routing (AC8)", () => {
  it("ShellLayout ErrorBoundary renders ErrorState inside the shell navbar", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(
          MemoryRouter,
          { initialEntries: ["/tasks"] },
          createElement(ShellErrorBoundary, {
            error: new Error("Boom inside route"),
          }),
        ),
      ),
    );

    assert.match(html, /aria-label="Apps"/);
    assert.match(html, /role="alert"/);
    assert.match(html, /Boom inside route/);
    assert.match(html, /Retry/);
  });

  it("Root ErrorBoundary renders 404 details for route error responses", () => {
    const html = renderToString(
      createElement(RootErrorBoundary, {
        params: {},
        error: {
          status: 404,
          statusText: "Not Found",
          internal: false,
          data: null,
        },
      }),
    );

    assert.match(html, /404/);
    assert.match(html, /The requested page could not be found\./);
  });

  it("places catch-all NotFound route outside RequireAuth", () => {
    const topLevelCatchAll = routes.find((r) => r.path === "*");
    assert.ok(
      topLevelCatchAll,
      "expected top-level * route outside RequireAuth",
    );
    assert.strictEqual(topLevelCatchAll.file, "shared/pages/NotFound.tsx");
  });
});
