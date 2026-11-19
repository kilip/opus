import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { taskKeys } from "../api/keys";
import { KanbanBoard } from "./KanbanBoard";

describe("KanbanBoard", () => {
  it("renders stage columns and task cards", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(taskKeys.projects(), []);

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(
          MemoryRouter,
          null,
          createElement(KanbanBoard, {
            tasks: [
              {
                id: "t1",
                key: "OPS-1",
                title: "Implement feature",
                description: "",
                status: "todo",
                priority: "2",
                projectId: "p1",
                source: "manual",
                type: "feat",
                labels: [],
                createdAt: "2026-01-01T00:00:00Z",
                updatedAt: "2026-01-01T00:00:00Z",
              },
            ],
          }),
        ),
      ),
    );

    assert.match(html, /OPS-1/);
    assert.match(html, /Implement feature/);
  });
});
