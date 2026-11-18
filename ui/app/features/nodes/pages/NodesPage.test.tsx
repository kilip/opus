import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { apps } from "~/apps";
import { api, unwrap } from "~/shared/api/client";
import { NODE_ERRORS } from "../api/errors";
import { nodeKeys } from "../api/keys";
import { nodesQueryOptions } from "../api/queries";
import type { JoinTokenResponse, NodeItem } from "../api/types";
import { nodesApp } from "../app";
import { nodesRoutes } from "../routes";
import NodesPage from "./NodesPage";

function renderNodesPage({
  nodes,
  url = "/nodes",
  queryState,
  initialJoinToken,
  initialConfirmingId,
}: {
  nodes?: NodeItem[];
  url?: string;
  queryState?: { status: "error"; error: Error };
  initialJoinToken?: JoinTokenResponse | null;
  initialConfirmingId?: string | null;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        retryOnMount: false,
        refetchOnMount: false,
      },
    },
  });

  if (queryState) {
    queryClient
      .getQueryCache()
      .build(queryClient, { queryKey: nodeKeys.list() })
      .setState({
        status: queryState.status,
        error: queryState.error,
        errorUpdatedAt: Date.now(),
        fetchStatus: "idle",
      });
  } else if (nodes !== undefined) {
    queryClient.setQueryData(nodeKeys.list(), nodes);
  }

  return renderToString(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[url]}>
        <NodesPage
          initialJoinToken={initialJoinToken}
          initialConfirmingId={initialConfirmingId}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("NodesPage (AC1, AC2, AC3, AC4, AC5)", () => {
  it("registers route /nodes and configures apps switcher entry (AC1)", () => {
    assert.strictEqual(nodesRoutes.length, 1);
    assert.strictEqual(nodesRoutes[0].path, "nodes");
    assert.strictEqual(
      nodesRoutes[0].file,
      "features/nodes/pages/NodesPage.tsx",
    );

    assert.strictEqual(nodesApp.id, "nodes");
    assert.strictEqual(nodesApp.name, "Nodes");
    assert.strictEqual(nodesApp.path, "/nodes");
    assert.strictEqual(nodesApp.icon, "monitor");
    assert.strictEqual(nodesApp.enabled, true);

    const registeredInApps = apps.find((a) => a.id === "nodes");
    assert.ok(registeredInApps);
    assert.strictEqual(registeredInApps?.enabled, true);
  });

  it("lists nodes with name, role badge and online/offline state, refetches every 15s (AC2)", () => {
    const nodes: NodeItem[] = [
      {
        id: "01933a3c-1b77-7000-8000-000000000001",
        name: "master-alpha",
        role: "master",
        status: "online",
      },
      {
        id: "01933a3c-1b77-7000-8000-000000000002",
        name: "worker-beta",
        role: "worker",
        status: "offline",
      },
    ];

    const html = renderNodesPage({ nodes });

    // Name rendered
    assert.match(html, /master-alpha/);
    assert.match(html, /worker-beta/);
    assert.match(html, /01933a3c-1b77-7000-8000-000000000001/);

    // Role badge rendered
    assert.match(html, />master<\/span>/);
    assert.match(html, />worker<\/span>/);

    // Online/offline status rendered
    assert.match(html, />online<\/span>/);
    assert.match(html, />offline<\/span>/);

    // Query options refetchInterval is 15 seconds
    const options = nodesQueryOptions();
    assert.strictEqual(options.refetchInterval, 15_000);
  });

  it("renders loading, error and empty states (AC2)", () => {
    // Loading state
    const loadingHtml = renderNodesPage();
    assert.match(loadingHtml, /Loading nodes…/);

    // Error state
    const errorHtml = renderNodesPage({
      queryState: {
        status: "error",
        error: new Error("Connection failed"),
      },
    });
    assert.match(errorHtml, /Could not load nodes/);
    assert.match(errorHtml, /Connection failed/);
    assert.match(errorHtml, />Retry<\/button>/);

    // Empty state
    const emptyHtml = renderNodesPage({ nodes: [] });
    assert.match(emptyHtml, /No nodes yet/);
    assert.match(emptyHtml, /Generate a join URL to connect worker nodes\./);
  });

  it("mocks join-token API and asserts the URL is displayed with copy button and expiry (AC3)", async () => {
    const mockUrl =
      "ws://localhost:7880/api/nodes/connect?token=join-token-secret-123";
    const mockExpiry = "2026-11-16T18:30:00Z";

    const originalPost = api.POST;
    try {
      api.POST = (async (path: string) => {
        if (path === "/nodes/join-tokens") {
          return {
            data: {
              data: {
                url: mockUrl,
                expiresAt: mockExpiry,
              },
            },
            response: new Response(JSON.stringify({}), { status: 201 }),
          };
        }
        return originalPost(path as never);
      }) as never;

      const queryClient = new QueryClient();
      const mutation = queryClient.getMutationCache().build(queryClient, {
        mutationFn: async () =>
          unwrap<JoinTokenResponse>(
            api.POST("/nodes/join-tokens"),
            NODE_ERRORS,
          ),
      });
      const result = (await mutation.execute(undefined)) as JoinTokenResponse;

      assert.strictEqual(result.url, mockUrl);
      assert.strictEqual(result.expiresAt, mockExpiry);

      const html = renderNodesPage({
        nodes: [],
        initialJoinToken: result,
      });

      // Assert "Generate join URL" button exists
      assert.match(html, />Generate join URL<\/button>/);

      // Assert mocked join URL is displayed
      assert.ok(html.includes(mockUrl));

      // Assert copy button is present
      assert.match(html, />Copy<\/button>/);

      // Assert expiry time is present
      assert.match(html, /Expires:/);
      assert.match(html, new RegExp(mockExpiry));
    } finally {
      api.POST = originalPost;
    }
  });

  it("master row has no remove button, worker row has remove button with confirm step (AC4)", async () => {
    const nodes: NodeItem[] = [
      {
        id: "node-master",
        name: "master-node",
        role: "master",
        status: "online",
      },
      {
        id: "node-worker",
        name: "worker-node",
        role: "worker",
        status: "online",
      },
    ];

    const html = renderNodesPage({ nodes });

    // Exactly one Remove button (for the worker, none for the master)
    const removeMatches = html.match(/>Remove<\/button>/g);
    assert.strictEqual(removeMatches?.length, 1);

    // Worker with confirm step shows Confirm and Cancel buttons
    const confirmHtml = renderNodesPage({
      nodes,
      initialConfirmingId: "node-worker",
    });
    assert.match(confirmHtml, />Confirm<\/button>/);
    assert.match(confirmHtml, />Cancel<\/button>/);

    // Verify delete endpoint invocation and query invalidation
    const originalDelete = api.DELETE;
    let deletedNodeId: string | undefined;
    try {
      api.DELETE = (async (
        path: string,
        options: { params: { path: { id: string } } },
      ) => {
        if (path === "/nodes/{id}") {
          deletedNodeId = options.params.path.id;
          return { response: new Response(null, { status: 204 }) };
        }
        return originalDelete(path as never, options as never);
      }) as never;

      const queryClient = new QueryClient();
      let invalidated = false;
      queryClient.invalidateQueries = (async (options: {
        queryKey: readonly unknown[];
      }) => {
        if (options.queryKey[0] === "nodes") invalidated = true;
      }) as never;

      const mutation = queryClient.getMutationCache().build(queryClient, {
        mutationFn: async (id: string) => {
          const { response } = await api.DELETE("/nodes/{id}", {
            params: { path: { id } },
          });
          if (!response.ok) throw new Error("Delete failed");
        },
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: nodeKeys.all });
        },
      });

      await mutation.execute("node-worker");
      assert.strictEqual(deletedNodeId, "node-worker");
      assert.strictEqual(invalidated, true);
    } finally {
      api.DELETE = originalDelete;
    }
  });
});
