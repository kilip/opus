import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { dashboardKeys } from "../api/keys";
import DashboardPage from "./DashboardPage";

describe("DashboardPage (AC6)", () => {
  it("renders ErrorState with Retry when any dashboard query fails", () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, retryOnMount: false, refetchOnMount: false },
      },
    });
    const nodesQuery = queryClient
      .getQueryCache()
      .build(queryClient, { queryKey: dashboardKeys.nodes() });
    nodesQuery.setState({
      status: "error",
      error: new Error("Nodes API unavailable"),
      errorUpdatedAt: Date.now(),
      fetchStatus: "idle",
    });
    queryClient.setQueryData(dashboardKeys.tasks(), []);
    queryClient.setQueryData(dashboardKeys.runs(), []);

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(MemoryRouter, null, createElement(DashboardPage)),
      ),
    ).replaceAll("<!-- -->", "");

    assert.match(html, /role="alert"/);
    assert.match(html, /Failed to load dashboard/);
    assert.match(html, /Nodes API unavailable/);
    assert.match(html, /Retry/);
    assert.doesNotMatch(html, /No nodes online\./);
  });

  it("renders online nodes and counts when queries succeed", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(dashboardKeys.nodes(), [
      { id: "n1", name: "worker-01", role: "worker", status: "online" },
    ]);
    queryClient.setQueryData(dashboardKeys.tasks(), [
      { id: "t1", status: "ready" },
    ]);
    queryClient.setQueryData(dashboardKeys.runs(), []);

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(MemoryRouter, null, createElement(DashboardPage)),
      ),
    ).replaceAll("<!-- -->", "");

    assert.match(html, /worker-01/);
    assert.match(html, /1 \/ 1 online/);
  });
});
