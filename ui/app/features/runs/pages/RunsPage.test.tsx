import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { runKeys } from "../api/keys";
import RunsPage from "./RunsPage";

describe("RunsPage", () => {
  it("renders empty state when there are no runs", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(runKeys.list(), []);

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(MemoryRouter, null, createElement(RunsPage)),
      ),
    );

    assert.match(html, /No runs found/);
  });

  it("renders error state with Retry when runs query fails", () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, retryOnMount: false, refetchOnMount: false },
      },
    });
    const query = queryClient
      .getQueryCache()
      .build(queryClient, { queryKey: runKeys.list() });
    query.setState({
      status: "error",
      error: new Error("Runs fetch failed"),
      errorUpdatedAt: Date.now(),
      fetchStatus: "idle",
    });

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(MemoryRouter, null, createElement(RunsPage)),
      ),
    );

    assert.match(html, /Could not load runs/);
    assert.match(html, /Runs fetch failed/);
    assert.match(html, /Retry/);
  });
});
