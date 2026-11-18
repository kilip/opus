import { type ReactNode, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";

import { Button, buttonClass } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Icon } from "~/shared/components/Icon";
import { SearchBar } from "~/shared/components/SearchBar";
import { useProjects, useTasks } from "../api/queries";
import { EmptyTasks } from "../components/EmptyTasks";
import { KanbanBoard } from "../components/KanbanBoard";
import { TaskList } from "../components/TaskList";
import { filterTasks, TASK_FILTERS } from "../model";

export const meta = () => [
  { title: "Tasks · Opus" },
  {
    name: "description",
    content: "AI mission control for your software factory",
  },
];

export default function TasksPage() {
  const [params, setParams] = useSearchParams();
  const view = params.get("view") === "list" ? "list" : "kanban";
  const selectedProject = params.get("projectId") ?? "";

  const { data: projects = [] } = useProjects();
  const tasksQuery = useTasks(
    selectedProject ? { projectId: selectedProject } : undefined,
  );

  const [query, setQuery] = useState("");
  const [active, setActive] = useState<ReadonlySet<string>>(new Set());

  const tasks = tasksQuery.data ?? [];
  const total = tasks.length;

  const visible = useMemo(
    () => filterTasks(tasks, query, active),
    [tasks, query, active],
  );

  function toggleFilter(id: string) {
    setActive((prev) => {
      const next = new Set(prev);
      if (!next.delete(id)) next.add(id);
      return next;
    });
  }

  let content: ReactNode;
  if (tasksQuery.isPending) {
    content = <p className="p-8 text-center text-muted">Loading tasks…</p>;
  } else if (tasksQuery.isError) {
    content = (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-lg font-semibold">Could not load tasks</p>
        <p className="text-muted">{tasksQuery.error?.message}</p>
        <Button variant="primary" onClick={() => tasksQuery.refetch()}>
          Retry
        </Button>
      </div>
    );
  } else if (total === 0) {
    content = <EmptyTasks />;
  } else if (view === "kanban") {
    content = <KanbanBoard tasks={visible} />;
  } else {
    content = <TaskList tasks={visible} />;
  }

  return (
    <>
      <ControlPanel
        breadcrumbs={[{ label: `Tasks (${total})` }]}
        actions={
          <Link to="/tasks/new" className={buttonClass("primary")}>
            New
          </Link>
        }
        right={
          <>
            <SearchBar
              value={query}
              onChange={setQuery}
              placeholder="Search tasks…"
              filters={TASK_FILTERS}
              active={active}
              onToggleFilter={toggleFilter}
            />
            <select
              aria-label="Filter by project"
              className="field h-8 py-0 text-sm"
              value={selectedProject}
              onChange={(e) => {
                const next = new URLSearchParams(params);
                if (e.target.value) {
                  next.set("projectId", e.target.value);
                } else {
                  next.delete("projectId");
                }
                setParams(next);
              }}
            >
              <option value="">All projects</option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
            <fieldset
              aria-label="View"
              className="m-0 inline-flex min-w-0 overflow-hidden rounded-md border border-line-strong p-0"
            >
              {(
                [
                  { id: "kanban", label: "Kanban", icon: "kanban" },
                  { id: "list", label: "List", icon: "list" },
                ] as const
              ).map((v) => (
                <button
                  key={v.id}
                  type="button"
                  aria-label={`${v.label} view`}
                  aria-pressed={view === v.id}
                  onClick={() => {
                    const next = new URLSearchParams(params);
                    if (v.id === "list") {
                      next.set("view", "list");
                    } else {
                      next.delete("view");
                    }
                    setParams(next);
                  }}
                  className="inline-flex h-8 w-9 items-center justify-center bg-surface text-muted hover:bg-surface-2 aria-pressed:bg-primary-soft aria-pressed:text-primary-ink"
                >
                  <Icon name={v.icon} size={17} />
                </button>
              ))}
            </fieldset>
          </>
        }
      />
      {content}
    </>
  );
}
