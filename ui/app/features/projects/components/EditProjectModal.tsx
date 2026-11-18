import { type FormEvent, useEffect, useState } from "react";

import { Button } from "~/shared/components/Button";
import { Icon } from "~/shared/components/Icon";
import { Tag } from "~/shared/components/Tag";
import {
  useDeleteProject,
  useSetProjectNodePath,
  useUpdateProject,
} from "../api/actions";
import { useProjectNodeMappings, useTrackerSettings } from "../api/queries";
import type { Project, ProjectNodeResponse } from "../api/types";
import {
  DEFAULT_BRANCH,
  DEFAULT_STATUS_LABELS,
  TRACKER_STATUS_ORDER,
  type TrackerStatusKey,
} from "../model";

export type EditProjectModalProps = {
  project: Project | null;
  isOpen: boolean;
  onClose: () => void;
};

function extractTrackerLabels(
  project: Project | null,
): Partial<Record<TrackerStatusKey, string>> {
  const next: Partial<Record<TrackerStatusKey, string>> = {};
  for (const status of TRACKER_STATUS_ORDER) {
    const v = project?.trackerLabels?.[status];
    if (v) next[status] = v;
  }
  return next;
}

function NodePathRow({
  node,
  projectId,
}: {
  node: ProjectNodeResponse;
  projectId: string;
}) {
  const [path, setPath] = useState(node.path ?? "");
  const setNodePath = useSetProjectNodePath();

  useEffect(() => {
    setPath(node.path ?? "");
  }, [node.path]);

  function handleSave() {
    setNodePath.mutate({
      projectId,
      nodeId: node.nodeId ?? "",
      path: path.trim() || "",
    });
  }

  function handleClear() {
    setPath("");
    setNodePath.mutate({
      projectId,
      nodeId: node.nodeId ?? "",
      path: "",
    });
  }

  const isMaster = node.nodeRole === "master";
  const isOnline = node.nodeStatus === "online";
  const errorMessage = setNodePath.error?.message ?? null;

  return (
    <div className="flex flex-col gap-1 border-b border-line py-3 last:border-b-0">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="font-medium text-fg">
            {node.nodeName || "Unnamed Node"}
          </span>
          <Tag tone={isMaster ? "primary" : "neutral"}>
            {node.nodeRole ?? "worker"}
          </Tag>
          <Tag tone={isOnline ? "success" : "neutral"}>
            {node.nodeStatus ?? "offline"}
          </Tag>
        </div>
      </div>

      <div className="mt-1 flex items-center gap-2">
        <input
          type="text"
          value={path}
          onChange={(e) => setPath(e.target.value)}
          placeholder="/path/to/project/root"
          aria-label={`Path on ${node.nodeName || "node"}`}
          className="field flex-1 font-mono text-xs"
        />
        <Button
          variant="secondary"
          onClick={handleSave}
          disabled={setNodePath.isPending}
        >
          {setNodePath.isPending ? "Saving…" : "Save"}
        </Button>
        {path && (
          <Button
            variant="ghost"
            onClick={handleClear}
            disabled={setNodePath.isPending}
            title="Clear mapping"
          >
            Clear
          </Button>
        )}
      </div>

      {errorMessage && (
        <p role="alert" className="text-xs text-danger">
          {errorMessage}
        </p>
      )}
    </div>
  );
}

export function EditProjectModal({
  project,
  isOpen,
  onClose,
}: EditProjectModalProps) {
  const updateProject = useUpdateProject();
  const deleteProject = useDeleteProject();

  const [name, setName] = useState("");
  const [defaultBranch, setDefaultBranch] = useState(DEFAULT_BRANCH);
  const [trackerProvider, setTrackerProvider] = useState("");
  const [trackerRef, setTrackerRef] = useState("");
  const [trackerLabels, setTrackerLabels] = useState<
    Partial<Record<TrackerStatusKey, string>>
  >(() => extractTrackerLabels(project));
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  useEffect(() => {
    if (project) {
      setName(project.name ?? "");
      setDefaultBranch(project.default_branch ?? DEFAULT_BRANCH);
      setTrackerProvider(project.tracker_provider ?? "");
      setTrackerRef(project.tracker_ref ?? "");
      const nextLabels: Partial<Record<TrackerStatusKey, string>> = {};
      for (const status of TRACKER_STATUS_ORDER) {
        const v = project.trackerLabels?.[status];
        if (v) nextLabels[status] = v;
      }
      setTrackerLabels(nextLabels);
      setConfirmingDelete(false);
      setSaveSuccess(false);
    }
  }, [project]);

  const nodesQuery = useProjectNodeMappings(project?.id ?? "");
  const trackerSettingsQuery = useTrackerSettings();

  if (!isOpen || !project) {
    return null;
  }

  const updateErrorMessage = updateProject.error?.message ?? null;
  const deleteErrorMessage = deleteProject.error?.message ?? null;

  function handleSaveMetadata(e: FormEvent) {
    e.preventDefault();
    if (!project) return;
    setSaveSuccess(false);
    const overrides: Record<string, string> = {};
    for (const status of TRACKER_STATUS_ORDER) {
      const v = trackerLabels[status]?.trim();
      if (v) overrides[status] = v;
    }
    updateProject.mutate(
      {
        id: project.id ?? "",
        name: name.trim(),
        default_branch: defaultBranch.trim() || DEFAULT_BRANCH,
        tracker_provider: trackerProvider.trim(),
        tracker_ref: trackerRef.trim(),
        trackerLabels: overrides,
      },
      {
        onSuccess: () => {
          setSaveSuccess(true);
          setTimeout(() => setSaveSuccess(false), 2500);
        },
      },
    );
  }

  function handleResetTrackerLabels() {
    if (!project) return;
    setSaveSuccess(false);
    setTrackerLabels({});
    updateProject.mutate(
      {
        id: project.id ?? "",
        trackerLabels: {},
      },
      {
        onSuccess: () => {
          setSaveSuccess(true);
          setTimeout(() => setSaveSuccess(false), 2500);
        },
      },
    );
  }

  function handleConfirmDelete() {
    if (!project?.id) return;
    deleteProject.mutate(project.id, {
      onSuccess: () => {
        onClose();
      },
    });
  }

  const nodes = nodesQuery.data ?? [];

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="edit-project-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
    >
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg border border-line bg-surface p-6 shadow-pop">
        <div className="flex items-center justify-between border-b border-line pb-3">
          <div className="flex items-center gap-2">
            <h2
              id="edit-project-title"
              className="text-lg font-semibold text-fg"
            >
              Edit Project: {project.name || project.slug}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-muted hover:bg-surface-2 hover:text-fg"
            aria-label="Close"
          >
            <Icon name="x" size={16} />
          </button>
        </div>

        {/* Metadata form */}
        <form
          onSubmit={handleSaveMetadata}
          className="mt-4 flex flex-col gap-4"
        >
          {updateErrorMessage && (
            <div
              role="alert"
              className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
            >
              {updateErrorMessage}
            </div>
          )}
          {saveSuccess && (
            <div className="rounded-md border border-success/30 bg-success/10 px-3 py-2 text-sm text-success">
              Project settings saved successfully.
            </div>
          )}

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label
                htmlFor="edit-project-slug"
                className="block text-xs font-semibold uppercase tracking-wider text-muted"
              >
                Slug (read-only)
              </label>
              <input
                id="edit-project-slug"
                name="slug"
                type="text"
                readOnly
                disabled
                value={project.slug ?? ""}
                className="field mt-1 cursor-not-allowed bg-surface-2 text-muted"
              />
            </div>

            <div>
              <label
                htmlFor="edit-project-repo"
                className="block text-xs font-semibold uppercase tracking-wider text-muted"
              >
                Repository (read-only)
              </label>
              <input
                id="edit-project-repo"
                name="repo_ref"
                type="text"
                readOnly
                disabled
                value={project.repo_ref ?? ""}
                className="field mt-1 cursor-not-allowed bg-surface-2 text-muted"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label
                htmlFor="edit-project-name"
                className="block text-xs font-semibold uppercase tracking-wider text-muted"
              >
                Name
              </label>
              <input
                id="edit-project-name"
                name="name"
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="field mt-1"
              />
            </div>

            <div>
              <label
                htmlFor="edit-project-branch"
                className="block text-xs font-semibold uppercase tracking-wider text-muted"
              >
                Default Branch
              </label>
              <input
                id="edit-project-branch"
                name="default_branch"
                type="text"
                value={defaultBranch}
                onChange={(e) => setDefaultBranch(e.target.value)}
                className="field mt-1"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label
                htmlFor="edit-project-tracker-provider"
                className="block text-xs font-semibold uppercase tracking-wider text-muted"
              >
                Tracker Provider
              </label>
              <input
                id="edit-project-tracker-provider"
                name="tracker_provider"
                type="text"
                placeholder="e.g. github, linear"
                value={trackerProvider}
                onChange={(e) => setTrackerProvider(e.target.value)}
                className="field mt-1"
              />
            </div>

            <div>
              <label
                htmlFor="edit-project-tracker-ref"
                className="block text-xs font-semibold uppercase tracking-wider text-muted"
              >
                Tracker Ref
              </label>
              <input
                id="edit-project-tracker-ref"
                name="tracker_ref"
                type="text"
                placeholder="e.g. owner/repo"
                value={trackerRef}
                onChange={(e) => setTrackerRef(e.target.value)}
                className="field mt-1"
              />
            </div>
          </div>

          <div className="border-t border-line pt-4">
            <div className="flex items-center justify-between gap-2">
              <div>
                <h3 className="text-xs font-semibold uppercase tracking-wider text-muted">
                  Tracker labels
                </h3>
                <p className="text-xs text-muted">
                  Override global status labels for this project.
                </p>
              </div>
              <Button
                type="button"
                variant="ghost"
                onClick={handleResetTrackerLabels}
                disabled={updateProject.isPending}
              >
                Reset to global
              </Button>
            </div>

            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
              {TRACKER_STATUS_ORDER.map((status) => (
                <div key={status} className="flex flex-col gap-1">
                  <label
                    htmlFor={`edit-project-tracker-label-${status}`}
                    className="text-xs font-medium text-muted"
                  >
                    {status}
                  </label>
                  <input
                    id={`edit-project-tracker-label-${status}`}
                    type="text"
                    value={trackerLabels[status] ?? ""}
                    placeholder={
                      trackerSettingsQuery.data?.statusLabels?.[status] ??
                      DEFAULT_STATUS_LABELS[status]
                    }
                    onChange={(e) =>
                      setTrackerLabels((prev) => ({
                        ...prev,
                        [status]: e.target.value,
                      }))
                    }
                    className="field text-sm"
                  />
                </div>
              ))}
              <div className="flex flex-col gap-1">
                <span className="text-xs font-medium text-muted">done</span>
                <input
                  type="text"
                  readOnly
                  disabled
                  value="closes the issue"
                  aria-label="done"
                  className="field cursor-not-allowed bg-surface-2 text-sm text-muted"
                />
              </div>
            </div>
            {updateErrorMessage && (
              <p role="alert" className="mt-2 text-xs font-medium text-danger">
                {updateErrorMessage}
              </p>
            )}
          </div>

          <div className="flex items-center justify-end gap-2 pt-2">
            <Button
              type="submit"
              variant="primary"
              disabled={updateProject.isPending}
            >
              {updateProject.isPending ? "Saving…" : "Save Changes"}
            </Button>
          </div>
        </form>

        {/* Node path mapping section */}
        <section className="mt-6 border-t border-line pt-4">
          <div className="flex flex-col gap-1">
            <h3 className="text-sm font-semibold text-fg">
              Node Path Mappings
            </h3>
            <p className="text-xs text-muted">
              Configure workspace repository root path on each node in your
              cluster.
            </p>
          </div>

          <div className="mt-3 rounded-md border border-line bg-surface-2/30 p-3">
            {nodesQuery.isPending && (
              <p className="p-4 text-center text-xs text-muted">
                Loading node mappings…
              </p>
            )}
            {nodesQuery.isError && (
              <p className="p-4 text-center text-xs text-danger">
                Failed to load node mappings: {nodesQuery.error?.message}
              </p>
            )}
            {!nodesQuery.isPending &&
              !nodesQuery.isError &&
              nodes.length === 0 && (
                <p className="p-4 text-center text-xs text-muted">
                  No nodes connected to cluster.
                </p>
              )}
            {nodes.map((node) => (
              <NodePathRow
                key={node.nodeId}
                node={node}
                projectId={project.id ?? ""}
              />
            ))}
          </div>
        </section>

        {/* Danger zone: delete */}
        <section className="mt-6 border-t border-line pt-4">
          <h3 className="text-sm font-semibold text-danger">Delete Project</h3>
          <p className="mt-1 text-xs text-muted">
            Permanently delete this project. Projects with associated tasks
            cannot be deleted.
          </p>

          {deleteErrorMessage && (
            <div
              role="alert"
              className="mt-3 rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
            >
              {deleteErrorMessage}
            </div>
          )}

          <div className="mt-3">
            {confirmingDelete ? (
              <div className="flex flex-col gap-2 rounded-md border border-danger/30 bg-danger/5 p-3">
                <p className="text-xs font-medium text-danger">
                  Are you sure you want to delete this project? This action
                  cannot be undone.
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="primary"
                    className="bg-danger text-white hover:bg-danger/90"
                    onClick={handleConfirmDelete}
                    disabled={deleteProject.isPending}
                  >
                    {deleteProject.isPending ? "Deleting…" : "Confirm Delete"}
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() => setConfirmingDelete(false)}
                    disabled={deleteProject.isPending}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                variant="secondary"
                className="border-danger/40 text-danger hover:bg-danger/10"
                onClick={() => setConfirmingDelete(true)}
              >
                Delete Project
              </Button>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
