import { type FormEvent, type ReactNode, useState } from "react";

import { Button } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Dialog } from "~/shared/components/Dialog";
import { Tag } from "~/shared/components/Tag";
import {
  useCreateJoinToken,
  useDeleteNode,
  useUpdateNode,
  useWakeNode,
} from "../api/actions";
import { useNodes } from "../api/queries";
import type { JoinTokenResponse, NodeItem } from "../api/types";

export const meta = () => [
  { title: "Nodes · Opus" },
  { name: "description", content: "Manage cluster nodes" },
];

export type NodesPageProps = {
  initialJoinToken?: JoinTokenResponse | null;
  initialConfirmingId?: string | null;
  initialEditingNode?: NodeItem | null;
};

export default function NodesPage({
  initialJoinToken = null,
  initialConfirmingId = null,
  initialEditingNode = null,
}: NodesPageProps = {}) {
  const nodesQuery = useNodes();
  const createJoinToken = useCreateJoinToken();
  const updateNode = useUpdateNode();
  const wakeNode = useWakeNode();
  const deleteNode = useDeleteNode();

  const [confirmingId, setConfirmingId] = useState<string | null>(
    initialConfirmingId,
  );
  const [editingNode, setEditingNode] = useState<NodeItem | null>(
    initialEditingNode,
  );
  const [editName, setEditName] = useState(initialEditingNode?.name ?? "");
  const [editIP, setEditIP] = useState(initialEditingNode?.ipAddress ?? "");
  const [editMAC, setEditMAC] = useState(initialEditingNode?.macAddress ?? "");
  const [wakingId, setWakingId] = useState<string | null>(null);
  const [wakeNotice, setWakeNotice] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const joinToken = createJoinToken.data ?? initialJoinToken;
  const nodes = nodesQuery.data ?? [];
  const total = nodes.length;

  const errorMessage =
    wakeNode.error?.message ??
    deleteNode.error?.message ??
    createJoinToken.error?.message ??
    null;

  function openEdit(node: NodeItem) {
    updateNode.reset();
    setEditingNode(node);
    setEditName(node.name ?? "");
    setEditIP(node.ipAddress ?? "");
    setEditMAC(node.macAddress ?? "");
  }

  function closeEdit() {
    setEditingNode(null);
    updateNode.reset();
  }

  function handleEditSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!editingNode?.id) return;
    updateNode.mutate(
      {
        id: editingNode.id,
        body: {
          name: editName.trim(),
          ipAddress: editIP.trim(),
          macAddress: editMAC.trim(),
        },
      },
      {
        onSuccess: () => setEditingNode(null),
      },
    );
  }

  function handleWake(node: NodeItem) {
    if (!node.id) return;
    setWakeNotice(null);
    setWakingId(node.id);
    wakeNode.mutate(node.id, {
      onSuccess: () => {
        setWakeNotice(`Wake-on-LAN packet sent to ${node.name || node.id}.`);
      },
      onSettled: () => setWakingId(null),
    });
  }

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
      <div className="overflow-x-auto rounded-lg border border-line bg-surface">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-line bg-surface-2 text-xs font-semibold uppercase tracking-wider text-muted">
            <tr>
              <th className="px-4 py-3">Node</th>
              <th className="px-4 py-3">Hostname</th>
              <th className="px-4 py-3">IP Address</th>
              <th className="px-4 py-3">MAC Address</th>
              <th className="px-4 py-3">Role</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {nodes.map((node) => {
              const isMaster = node.role === "master";
              const isConfirming = confirmingId === node.id;
              const canWake =
                !isMaster &&
                node.status === "offline" &&
                Boolean(node.macAddress);
              const isWaking = wakingId === node.id && wakeNode.isPending;
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
                  <td className="px-4 py-3 font-mono text-xs text-muted">
                    {node.hostname || "—"}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted">
                    {node.ipAddress || "—"}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted">
                    {node.macAddress || "—"}
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
                    <div className="inline-flex items-center justify-end gap-1.5">
                      {canWake && (
                        <Button
                          variant="secondary"
                          onClick={() => handleWake(node)}
                          disabled={isWaking}
                        >
                          {isWaking ? "Waking…" : "Wake"}
                        </Button>
                      )}
                      <Button
                        variant="secondary"
                        onClick={() => openEdit(node)}
                      >
                        Edit
                      </Button>
                      {!isMaster &&
                        (isConfirming ? (
                          <>
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
                          </>
                        ) : (
                          <Button
                            variant="secondary"
                            onClick={() => setConfirmingId(node.id ?? null)}
                          >
                            Remove
                          </Button>
                        ))}
                    </div>
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

      <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 p-6">
        {errorMessage && (
          <div
            role="alert"
            className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {errorMessage}
          </div>
        )}

        {wakeNotice && (
          <output className="rounded-md border border-success/30 bg-success/10 px-3 py-2 text-sm text-fg">
            {wakeNotice}
          </output>
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

      <Dialog
        open={Boolean(editingNode)}
        onClose={closeEdit}
        titleId="edit-node-title"
        className="w-full max-w-md p-6"
      >
        {editingNode && (
          <form onSubmit={handleEditSubmit} className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <h2
                id="edit-node-title"
                className="text-base font-semibold text-fg"
              >
                Edit node
              </h2>
              <span className="font-mono text-xs text-muted">
                {editingNode.role ?? "worker"}
              </span>
            </div>

            {updateNode.error && (
              <div
                role="alert"
                className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
              >
                {updateNode.error.message}
              </div>
            )}

            <label className="flex flex-col gap-1 text-xs font-medium text-muted">
              Hostname
              <input
                type="text"
                readOnly
                value={editingNode.hostname || "—"}
                className="rounded-md border border-line bg-surface-2 px-3 py-2 font-mono text-sm text-muted focus:outline-none"
              />
            </label>

            <label className="flex flex-col gap-1 text-xs font-medium text-muted">
              Name
              <input
                type="text"
                required
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                className="rounded-md border border-line bg-surface-2 px-3 py-2 text-sm text-fg focus:border-primary focus:outline-none"
              />
            </label>

            <label className="flex flex-col gap-1 text-xs font-medium text-muted">
              IP Address
              <input
                type="text"
                placeholder="e.g. 192.168.1.10"
                value={editIP}
                onChange={(e) => setEditIP(e.target.value)}
                className="rounded-md border border-line bg-surface-2 px-3 py-2 font-mono text-sm text-fg focus:border-primary focus:outline-none"
              />
            </label>

            <label className="flex flex-col gap-1 text-xs font-medium text-muted">
              MAC Address
              <input
                type="text"
                placeholder="e.g. aa:bb:cc:dd:ee:ff"
                value={editMAC}
                onChange={(e) => setEditMAC(e.target.value)}
                className="rounded-md border border-line bg-surface-2 px-3 py-2 font-mono text-sm text-fg focus:border-primary focus:outline-none"
              />
            </label>

            <div className="flex justify-end gap-2 pt-2">
              <Button
                type="button"
                variant="secondary"
                onClick={closeEdit}
                disabled={updateNode.isPending}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={updateNode.isPending}
              >
                {updateNode.isPending ? "Saving…" : "Save"}
              </Button>
            </div>
          </form>
        )}
      </Dialog>
    </>
  );
}
