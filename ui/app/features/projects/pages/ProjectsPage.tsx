import { type FormEvent, type ReactNode, useEffect, useState } from "react";

import { Button } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Icon } from "~/shared/components/Icon";
import { Tag } from "~/shared/components/Tag";
import { formatAbsoluteTime, formatRelativeTime } from "~/shared/util/dates";
import { useUpdateTrackerSettings } from "../api/actions";
import {
  useProjects,
  useProjectTracker,
  useTrackerSettings,
} from "../api/queries";
import type { Project } from "../api/types";
import { CreateProjectModal } from "../components/CreateProjectModal";
import { EditProjectModal } from "../components/EditProjectModal";
import {
  DEFAULT_STATUS_LABELS,
  TRACKER_STATUS_ORDER,
  type TrackerStatusKey,
} from "../model";

export const meta = () => [
  { title: "Projects · Opus" },
  { name: "description", content: "Manage projects" },
];

export type ProjectsPageProps = {
  initialEditingProject?: Project | null;
  initialIsCreateOpen?: boolean;
};

export default function ProjectsPage({
  initialEditingProject = null,
  initialIsCreateOpen = false,
}: ProjectsPageProps = {}) {
  const projectsQuery = useProjects();
  const [isCreateOpen, setIsCreateOpen] = useState(initialIsCreateOpen);
  const [editingProject, setEditingProject] = useState<Project | null>(
    initialEditingProject,
  );

  const projects = projectsQuery.data ?? [];
  const total = projects.length;

  let content: ReactNode;
  if (projectsQuery.isPending) {
    content = <p className="p-8 text-center text-muted">Loading projects…</p>;
  } else if (projectsQuery.isError) {
    content = (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-lg font-semibold">Could not load projects</p>
        <p className="text-muted">{projectsQuery.error?.message}</p>
        <Button variant="primary" onClick={() => projectsQuery.refetch()}>
          Retry
        </Button>
      </div>
    );
  } else if (total === 0) {
    content = (
      <div className="rounded-lg border border-line bg-surface p-8 text-center text-muted">
        <p className="font-semibold text-fg">No projects yet</p>
        <p className="mt-1 text-sm">
          Create a project to start managing tasks and node paths.
        </p>
        <div className="mt-4">
          <Button variant="primary" onClick={() => setIsCreateOpen(true)}>
            Create Project
          </Button>
        </div>
      </div>
    );
  } else {
    content = (
      <div className="overflow-hidden rounded-lg border border-line bg-surface">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-line bg-surface-2 text-xs font-semibold uppercase tracking-wider text-muted">
            <tr>
              <th className="px-4 py-3">Project</th>
              <th className="px-4 py-3">Repository</th>
              <th className="px-4 py-3">Default Branch</th>
              <th className="px-4 py-3">Sync Health</th>
              <th className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {projects.map((project) => (
              <tr
                key={project.id ?? project.slug}
                className="hover:bg-surface-2/50"
              >
                <td className="px-4 py-3 font-medium text-fg">
                  <div className="flex flex-col">
                    <span>{project.name || "Unnamed Project"}</span>
                    <span className="font-mono text-xs text-muted">
                      {project.slug}
                    </span>
                  </div>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5 font-mono text-xs text-fg">
                    <Icon name="github" size={14} className="text-muted" />
                    <span>{project.repo_ref || "—"}</span>
                  </div>
                </td>
                <td className="px-4 py-3">
                  <Tag tone="neutral">{project.default_branch || "main"}</Tag>
                </td>
                <td className="px-4 py-3">
                  <ProjectTrackerStatusCell projectId={project.id ?? ""} />
                </td>
                <td className="px-4 py-3 text-right">
                  <Button
                    variant="secondary"
                    onClick={() => setEditingProject(project)}
                  >
                    Edit
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <>
      <ControlPanel
        breadcrumbs={[{ label: `Projects (${total})` }]}
        actions={
          <Button variant="primary" onClick={() => setIsCreateOpen(true)}>
            Create Project
          </Button>
        }
      />

      <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-6">
        {content}
        <GlobalTrackerLabelsSection />
      </div>

      <CreateProjectModal
        isOpen={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
      />

      {editingProject && (
        <EditProjectModal
          project={editingProject}
          isOpen={Boolean(editingProject)}
          onClose={() => setEditingProject(null)}
        />
      )}
    </>
  );
}

function ProjectTrackerStatusCell({ projectId }: { projectId: string }) {
  const trackerQuery = useProjectTracker(projectId);
  if (!projectId || trackerQuery.isPending) {
    return <span className="text-xs text-muted">Checking…</span>;
  }
  if (trackerQuery.isError) {
    return <span className="text-xs text-muted">Sync status unavailable</span>;
  }
  const { lastSyncAt, lastError } = trackerQuery.data;
  return (
    <div className="flex flex-col items-start gap-1">
      {lastSyncAt ? (
        <time
          dateTime={lastSyncAt}
          title={formatAbsoluteTime(lastSyncAt)}
          className="text-xs text-muted"
        >
          Synced {formatRelativeTime(lastSyncAt)}
        </time>
      ) : (
        <span className="text-xs text-muted">Never synced</span>
      )}
      {lastError && (
        <div
          role="alert"
          className="inline-flex items-center gap-1 rounded border border-danger/30 bg-danger/10 px-2 py-0.5 text-xs font-medium text-danger"
        >
          <Icon name="x" size={12} aria-hidden="true" />
          <span>Sync error: {lastError}</span>
        </div>
      )}
    </div>
  );
}

function GlobalTrackerLabelsSection() {
  const settingsQuery = useTrackerSettings();
  const updateSettings = useUpdateTrackerSettings();
  const [draft, setDraft] = useState<Partial<Record<TrackerStatusKey, string>>>(
    {},
  );
  const [saved, setSaved] = useState(false);

  const serverLabels = settingsQuery.data?.statusLabels;
  useEffect(() => {
    if (!serverLabels) return;
    const next: Partial<Record<TrackerStatusKey, string>> = {};
    for (const status of TRACKER_STATUS_ORDER) {
      const val = serverLabels[status];
      next[status] = val && val !== DEFAULT_STATUS_LABELS[status] ? val : "";
    }
    setDraft(next);
  }, [serverLabels]);

  function handleSave(e: FormEvent) {
    e.preventDefault();
    setSaved(false);
    const statusLabels: Record<string, string> = {};
    for (const status of TRACKER_STATUS_ORDER) {
      const v = draft[status]?.trim();
      if (v) statusLabels[status] = v;
    }
    updateSettings.mutate(
      { statusLabels },
      {
        onSuccess: () => {
          setSaved(true);
          setTimeout(() => setSaved(false), 2500);
        },
      },
    );
  }

  function handleReset() {
    setSaved(false);
    setDraft({});
    updateSettings.mutate({ statusLabels: {} });
  }

  const errorMessage = updateSettings.error?.message ?? null;

  return (
    <section
      aria-labelledby="tracker-labels-heading"
      className="rounded-lg border border-line bg-surface p-5"
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line pb-3">
        <div>
          <h2
            id="tracker-labels-heading"
            className="text-base font-semibold text-fg"
          >
            Tracker labels
          </h2>
          <p className="text-xs text-muted">
            Global GitHub status-to-label mappings. Leave blank to use the
            default label.
          </p>
        </div>
        <Button
          type="button"
          variant="ghost"
          onClick={handleReset}
          disabled={updateSettings.isPending}
        >
          Reset to defaults
        </Button>
      </div>

      <form onSubmit={handleSave} className="mt-4 flex flex-col gap-4">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {TRACKER_STATUS_ORDER.map((status) => (
            <div key={status} className="flex flex-col gap-1">
              <label
                htmlFor={`global-tracker-label-${status}`}
                className="text-xs font-semibold uppercase tracking-wider text-muted"
              >
                {status}
              </label>
              <input
                id={`global-tracker-label-${status}`}
                type="text"
                value={draft[status] ?? ""}
                placeholder={DEFAULT_STATUS_LABELS[status]}
                onChange={(e) =>
                  setDraft((prev) => ({ ...prev, [status]: e.target.value }))
                }
                className="field text-sm"
              />
            </div>
          ))}
          <div className="flex flex-col gap-1">
            <span className="text-xs font-semibold uppercase tracking-wider text-muted">
              done
            </span>
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

        {errorMessage && (
          <p role="alert" className="text-xs font-medium text-danger">
            {errorMessage}
          </p>
        )}
        {saved && (
          <p className="text-xs font-medium text-success">
            Tracker labels saved.
          </p>
        )}

        <div className="flex justify-end">
          <Button
            type="submit"
            variant="primary"
            disabled={updateSettings.isPending}
          >
            {updateSettings.isPending ? "Saving…" : "Save tracker labels"}
          </Button>
        </div>
      </form>
    </section>
  );
}
