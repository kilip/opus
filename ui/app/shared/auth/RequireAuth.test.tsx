import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter, Route, Routes } from "react-router";

import { ApiError } from "../api/client";
import { RequireAuth } from "./RequireAuth";
import { sessionKeys } from "./session";

describe("RequireAuth (AC3)", () => {
  it("renders visible loading state while setup/session queries are pending", () => {
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
          createElement(
            RequireAuth,
            null,
            createElement("div", null, "Protected"),
          ),
        ),
      ),
    );

    assert.match(html, /role="status"/);
    assert.match(html, /Loading\.\.\./);
    assert.doesNotMatch(html, /Protected/);
  });

  it("renders ErrorState with Retry on 500/network error without redirecting", () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, retryOnMount: false, refetchOnMount: false },
      },
    });
    queryClient.setQueryData(sessionKeys.setupStatus(), { needs_setup: false });
    const query = queryClient
      .getQueryCache()
      .build(queryClient, { queryKey: sessionKeys.me() });
    query.setState({
      status: "error",
      error: new ApiError(500, "INTERNAL_ERROR", "Server down"),
      errorUpdatedAt: Date.now(),
      fetchStatus: "idle",
    });

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(
          MemoryRouter,
          { initialEntries: ["/tasks"] },
          createElement(
            Routes,
            null,
            createElement(Route, {
              path: "/login",
              element: createElement("div", null, "LOGIN_PAGE"),
            }),
            createElement(Route, {
              path: "*",
              element: createElement(
                RequireAuth,
                null,
                createElement("div", null, "Protected"),
              ),
            }),
          ),
        ),
      ),
    );

    assert.match(html, /role="alert"/);
    assert.match(html, /Server down/);
    assert.match(html, /Retry/);
    assert.doesNotMatch(html, /LOGIN_PAGE/);
  });

  it("renders protected children when authenticated", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(sessionKeys.setupStatus(), { needs_setup: false });
    queryClient.setQueryData(sessionKeys.me(), {
      id: "u1",
      email: "dev@example.com",
    });

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(
          MemoryRouter,
          { initialEntries: ["/tasks"] },
          createElement(
            RequireAuth,
            null,
            createElement("div", null, "Protected Content"),
          ),
        ),
      ),
    );

    assert.match(html, /Protected Content/);
  });
});
