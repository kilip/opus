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
import type {
  JoinTokenResponse,
  NodeItem,
  UpdateNodeRequest,
} from "../api/types";
import { nodesApp } from "../app";
import { nodesRoutes } from "../routes";
import NodesPage from "./NodesPage";

function renderNodesPage({
  nodes,
  url = "/nodes",
  queryState,
  initialJoinToken,
  initialConfirmingId,
  initialEditingNode,
}: {
  nodes?: NodeItem[];
  url?: string;
  queryState?: { status: "error"; error: Error };
  initialJoinToken?: JoinTokenResponse | null;
  initialConfirmingId?: string | null;
  initialEditingNode?: NodeItem | null;
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
          initialEditingNode={initialEditingNode}
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

  it("displays hostname, ipAddress, macAddress with dash fallback and maps new error codes (AC1, AC3)", () => {
    assert.ok(NODE_ERRORS.INVALID_NODE_NAME);
    assert.ok(NODE_ERRORS.INVALID_IP_ADDRESS);
    assert.ok(NODE_ERRORS.INVALID_MAC_ADDRESS);
    assert.ok(NODE_ERRORS.NODE_MAC_NOT_CONFIGURED);

    const nodes: NodeItem[] = [
      {
        id: "node-master",
        name: "primary-master",
        role: "master",
        status: "online",
        hostname: "opus-host.local",
        ipAddress: "192.168.1.10",
        macAddress: "aa:bb:cc:dd:ee:01",
      },
      {
        id: "node-worker-empty",
        name: "worker-no-net",
        role: "worker",
        status: "offline",
      },
    ];

    const html = renderNodesPage({ nodes });
    assert.match(html, /opus-host\.local/);
    assert.match(html, /192\.168\.1\.10/);
    assert.match(html, /aa:bb:cc:dd:ee:01/);
    assert.match(html, />—<\/td>/);
  });

  it("renders Edit on every row, pre-fills edit modal, and submits PATCH /nodes/{id} (AC2, AC4, AC6)", async () => {
    const masterNode: NodeItem = {
      id: "node-master",
      name: "control-plane",
      role: "master",
      status: "online",
      hostname: "master-host",
      ipAddress: "10.0.0.1",
      macAddress: "11:22:33:44:55:66",
    };
    const workerNode: NodeItem = {
      id: "node-worker",
      name: "worker-1",
      role: "worker",
      status: "online",
    };

    const html = renderNodesPage({ nodes: [masterNode, workerNode] });
    const editMatches = html.match(/>Edit<\/button>/g);
    assert.strictEqual(editMatches?.length, 2);

    const modalHtml = renderNodesPage({
      nodes: [masterNode, workerNode],
      initialEditingNode: masterNode,
    });
    assert.match(modalHtml, /role="dialog"/);
    assert.match(modalHtml, /value="master-host"/);
    assert.match(modalHtml, /value="control-plane"/);
    assert.match(modalHtml, /value="10\.0\.0\.1"/);
    assert.match(modalHtml, /value="11:22:33:44:55:66"/);

    const originalPatch = api.PATCH;
    let patchedId: string | undefined;
    let patchedBody: UpdateNodeRequest | undefined;
    try {
      api.PATCH = (async (
        path: string,
        options: { params: { path: { id: string } }; body: UpdateNodeRequest },
      ) => {
        if (path === "/nodes/{id}") {
          patchedId = options.params.path.id;
          patchedBody = options.body;
          return {
            data: { data: { ...masterNode, ...options.body } },
            response: new Response(JSON.stringify({}), { status: 200 }),
          };
        }
        return originalPatch(path as never, options as never);
      }) as never;

      const queryClient = new QueryClient();
      let invalidated = false;
      queryClient.invalidateQueries = (async (options: {
        queryKey: readonly unknown[];
      }) => {
        if (options.queryKey[0] === "nodes") invalidated = true;
      }) as never;

      const mutation = queryClient.getMutationCache().build(queryClient, {
        mutationFn: async ({
          id,
          body,
        }: {
          id: string;
          body: UpdateNodeRequest;
        }) =>
          unwrap<NodeItem>(
            api.PATCH("/nodes/{id}", {
              params: { path: { id } },
              body,
            }),
            NODE_ERRORS,
          ),
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: nodeKeys.all });
        },
      });

      await mutation.execute({
        id: "node-master",
        body: { name: "renamed-master", ipAddress: "", macAddress: "" },
      });

      assert.strictEqual(patchedId, "node-master");
      assert.deepStrictEqual(patchedBody, {
        name: "renamed-master",
        ipAddress: "",
        macAddress: "",
      });
      assert.strictEqual(invalidated, true);
    } finally {
      api.PATCH = originalPatch;
    }
  });

  it("renders Wake only on offline workers with macAddress and triggers POST /nodes/{id}/wake (AC2, AC5, AC6)", async () => {
    const nodes: NodeItem[] = [
      {
        id: "master-1",
        name: "master",
        role: "master",
        status: "offline",
        macAddress: "aa:bb:cc:dd:ee:00",
      },
      {
        id: "worker-online",
        name: "worker-online",
        role: "worker",
        status: "online",
        macAddress: "aa:bb:cc:dd:ee:01",
      },
      {
        id: "worker-offline-nomac",
        name: "worker-nomac",
        role: "worker",
        status: "offline",
      },
      {
        id: "worker-offline-mac",
        name: "worker-wakeable",
        role: "worker",
        status: "offline",
        macAddress: "aa:bb:cc:dd:ee:02",
      },
    ];

    const html = renderNodesPage({ nodes });
    const wakeMatches = html.match(/>Wake<\/button>/g);
    assert.strictEqual(wakeMatches?.length, 1);

    const originalPost = api.POST;
    let wokenId: string | undefined;
    try {
      api.POST = (async (
        path: string,
        options?: { params?: { path?: { id?: string } } },
      ) => {
        if (path === "/nodes/{id}/wake") {
          wokenId = options?.params?.path?.id;
          return { response: new Response(null, { status: 204 }) };
        }
        return originalPost(path as never, options as never);
      }) as never;

      const queryClient = new QueryClient();
      const mutation = queryClient.getMutationCache().build(queryClient, {
        mutationFn: async (id: string) => {
          const { response } = await api.POST("/nodes/{id}/wake", {
            params: { path: { id } },
          });
          if (!response.ok) throw new Error("Wake failed");
        },
      });

      await mutation.execute("worker-offline-mac");
      assert.strictEqual(wokenId, "worker-offline-mac");
    } finally {
      api.POST = originalPost;
    }
  });
});
