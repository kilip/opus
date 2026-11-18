import { type ReactNode, useState } from "react";

import { Button } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Tag } from "~/shared/components/Tag";
import { useCreateJoinToken, useDeleteNode } from "../api/actions";
import { useNodes } from "../api/queries";
import type { JoinTokenResponse } from "../api/types";

export const meta = () => [
  { title: "Nodes · Opus" },
  { name: "description", content: "Manage cluster nodes" },
];

export type NodesPageProps = {
  initialJoinToken?: JoinTokenResponse | null;
  initialConfirmingId?: string | null;
};

export default function NodesPage({
  initialJoinToken = null,
  initialConfirmingId = null,
}: NodesPageProps = {}) {
  const nodesQuery = useNodes();
  const createJoinToken = useCreateJoinToken();
  const deleteNode = useDeleteNode();

  const [confirmingId, setConfirmingId] = useState<string | null>(
    initialConfirmingId,
  );
  const [copied, setCopied] = useState(false);

  const joinToken = createJoinToken.data ?? initialJoinToken;
  const nodes = nodesQuery.data ?? [];
  const total = nodes.length;

  const errorMessage =
    deleteNode.error?.message ?? createJoinToken.error?.message ?? null;

  function handleDelete(id?: string) {
    if (!id) return;
    deleteNode.mutate(id, {
      onSuccess: () => setConfirmingId(null),
    });
  }

  function handleCopy(url?: string) {
    if (!url) return;
    navigator.clipboard?.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  let content: ReactNode;
  if (nodesQuery.isPending) {
    content = <p className="p-8 text-center text-muted">Loading nodes…</p>;
  } else if (nodesQuery.isError) {
    content = (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-lg font-semibold">Could not load nodes</p>
        <p className="text-muted">{nodesQuery.error?.message}</p>
        <Button variant="primary" onClick={() => nodesQuery.refetch()}>
          Retry
        </Button>
      </div>
    );
  } else if (total === 0) {
    content = (
      <div className="rounded-lg border border-line bg-surface p-8 text-center text-muted">
        <p className="font-semibold text-fg">No nodes yet</p>
        <p className="mt-1 text-sm">
          Generate a join URL to connect worker nodes.
        </p>
      </div>
    );
  } else {
    content = (
      <div className="overflow-hidden rounded-lg border border-line bg-surface">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-line bg-surface-2 text-xs font-semibold uppercase tracking-wider text-muted">
            <tr>
              <th className="px-4 py-3">Node</th>
              <th className="px-4 py-3">Role</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {nodes.map((node) => {
              const isMaster = node.role === "master";
              const isConfirming = confirmingId === node.id;
              return (
                <tr
                  key={node.id ?? node.name}
                  className="hover:bg-surface-2/50"
                >
                  <td className="px-4 py-3 font-medium text-fg">
                    <div className="flex flex-col">
                      <span>{node.name || "Unnamed Node"}</span>
                      {node.id && (
                        <span className="font-mono text-xs text-muted">
                          {node.id}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <Tag tone={isMaster ? "primary" : "neutral"}>
                      {node.role ?? "worker"}
                    </Tag>
                  </td>
                  <td className="px-4 py-3">
                    <Tag
                      tone={node.status === "online" ? "success" : "neutral"}
                    >
                      {node.status ?? "offline"}
                    </Tag>
                  </td>
                  <td className="px-4 py-3 text-right">
                    {!isMaster &&
                      (isConfirming ? (
                        <div className="inline-flex items-center gap-1.5">
                          <Button
                            variant="primary"
                            className="bg-danger text-white hover:bg-danger/90"
                            onClick={() => handleDelete(node.id)}
                            disabled={deleteNode.isPending}
                          >
                            {deleteNode.isPending ? "Removing…" : "Confirm"}
                          </Button>
                          <Button
                            variant="secondary"
                            onClick={() => setConfirmingId(null)}
                            disabled={deleteNode.isPending}
                          >
                            Cancel
                          </Button>
                        </div>
                      ) : (
                        <Button
                          variant="secondary"
                          onClick={() => setConfirmingId(node.id ?? null)}
                        >
                          Remove
                        </Button>
                      ))}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <>
      <ControlPanel
        breadcrumbs={[{ label: `Nodes (${total})` }]}
        actions={
          <Button
            variant="primary"
            onClick={() => createJoinToken.mutate()}
            disabled={createJoinToken.isPending}
          >
            {createJoinToken.isPending ? "Generating…" : "Generate join URL"}
          </Button>
        }
      />

      <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-6">
        {errorMessage && (
          <div
            role="alert"
            className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {errorMessage}
          </div>
        )}

        {joinToken && (
          <section
            aria-label="Node join token"
            className="flex flex-col gap-2 rounded-lg border border-line bg-surface p-4"
          >
            <div className="flex items-center justify-between">
              <span className="text-sm font-semibold text-fg">
                Worker Join URL
              </span>
              {joinToken.expiresAt && (
                <span className="text-xs text-muted">
                  Expires:{" "}
                  <time dateTime={joinToken.expiresAt}>
                    {joinToken.expiresAt}
                  </time>
                </span>
              )}
            </div>
            <p className="text-xs text-muted">
              Use this URL on a worker machine to connect to this cluster.
            </p>
            <div className="flex items-center gap-2">
              <input
                type="text"
                readOnly
                value={joinToken.url ?? ""}
                className="flex-1 rounded-md border border-line bg-surface-2 px-3 py-1.5 font-mono text-xs text-fg select-all focus:outline-none"
                aria-label="Node join URL"
              />
              <Button
                variant="secondary"
                onClick={() => handleCopy(joinToken.url)}
              >
                {copied ? "Copied!" : "Copy"}
              </Button>
            </div>
          </section>
        )}

        {content}
      </div>
    </>
  );
}
