import { useId, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

import { ApiError } from "~/shared/api/client";
import { Button, buttonClass } from "~/shared/components/Button";
import { Chatter, type ChatterMessage } from "~/shared/components/Chatter";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Priority } from "~/shared/components/Priority";
import { Statusbar } from "~/shared/components/Statusbar";
import { Tag } from "~/shared/components/Tag";
import {
  useAddTaskActivity,
  useCreateTask,
  useMoveTask,
  useUpdateTask,
} from "../api/actions";
import {
  useProjects,
  useTask,
  useTaskActivity,
  useTaskPRs,
} from "../api/queries";
import type {
  ChecksSummary,
  PRState,
  Task,
  TaskStatus,
  UpdateTaskRequest,
} from "../api/types";
import {
  ASSIGNEES,
  CURRENT_USER,
  priorityFromLevel,
  priorityLevel,
  STAGES,
} from "../model";

export const meta = () => [{ title: "Task · Opus" }];

type Draft = {
  title: string;
  projectId: string;
  /** 0–3, star widget level. */
  priority: number;
  /** Assignee display name; empty for unassigned. */
  assignee: string;
  deadline: string;
  tagsText: string;
  description: string;
};

function toDraft(task?: Task): Draft {
  return {
    title: task?.title ?? "",
    projectId: task?.projectId ?? "",
    priority: task ? priorityLevel(task.priority) : 0,
    assignee: task ? (task.assignee?.name ?? "") : CURRENT_USER,
    deadline: task?.dueDate ?? "",
    tagsText: task?.labels.join(", ") ?? "",
    description: task?.description ?? "",
  };
}

function parseTags(text: string) {
  return [
    ...new Set(
      text
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
    ),
  ];
}

export default function TaskFormPage() {
  const { taskId } = useParams();
  const isNew = taskId === "new";
  const {
    data: task,
    isLoading,
    error,
    refetch,
  } = useTask(isNew ? undefined : taskId);

  if (!isNew && isLoading) {
    return <p className="p-8 text-center text-muted">Loading task…</p>;
  }

  if (!isNew && !task) {
    const notFound = error instanceof ApiError && error.status === 404;
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-lg font-semibold">
          {notFound ? "Task not found" : "Could not load task"}
        </p>
        {!notFound && error && <p className="text-muted">{error.message}</p>}
        <div className="flex items-center gap-2">
          {!notFound && (
            <Button variant="secondary" onClick={() => refetch()}>
              Retry
            </Button>
          )}
          <Link to="/tasks" className={buttonClass("primary")}>
            Back to tasks
          </Link>
        </div>
      </div>
    );
  }

  // Render form only once task is loaded, and remount per record, so
  // draft starts with real data and never leaks between tasks.
  return <TaskForm key={isNew ? "new" : taskId} task={task} />;
}

function TaskForm({ task }: { task?: Task }) {
  const navigate = useNavigate();
  const createTaskMutation = useCreateTask();
  const updateTaskMutation = useUpdateTask();
  const moveTaskMutation = useMoveTask();
  const addActivityMutation = useAddTaskActivity();
  const { data: activity } = useTaskActivity(task?.id);
  const { data: prs } = useTaskPRs(task?.id);
  const { data: projects = [] } = useProjects();
  const ids = {
    title: useId(),
    project: useId(),
    assignee: useId(),
    deadline: useId(),
    tags: useId(),
    desc: useId(),
  };

  const projectOptions = useMemo(() => {
    if (task?.projectId && !projects.some((p) => p.id === task.projectId)) {
      return [
        {
          id: task.projectId,
          name: task.projectId,
          slug: task.projectId,
          repo_provider: "",
          repo_ref: "",
          tracker_provider: "",
          tracker_ref: "",
          default_branch: "main",
          created_at: "",
          updated_at: "",
        },
        ...projects,
      ];
    }
    return projects;
  }, [projects, task]);

  const baseline = useMemo(() => toDraft(task), [task]);
  const [draft, setDraft] = useState<Draft>(baseline);
  const [newStage, setNewStage] = useState<TaskStatus>(STAGES[0].id);
  const [attempted, setAttempted] = useState(false);

  const messages = useMemo<ChatterMessage[]>(
    () =>
      (activity ?? []).map(({ createdAt, kind, ...rest }) => ({
        ...rest,
        kind:
          kind === "status" || kind === "assign"
            ? "tracking"
            : kind === "comment"
              ? "message"
              : "note",
        at: createdAt,
      })),
    [activity],
  );

  // An existing assignee outside the known list must still be selectable.
  const assigneeOptions = useMemo(
    () => [
      ...new Set([
        ...ASSIGNEES,
        ...(task?.assignee ? [task.assignee.name] : []),
      ]),
    ],
    [task],
  );

  const dirty = !task || JSON.stringify(draft) !== JSON.stringify(baseline);
  const titleInvalid = attempted && !draft.title.trim();
  const projectInvalid = attempted && !draft.projectId.trim();
  const status = task?.status ?? newStage;

  function set<K extends keyof Draft>(key: K, value: Draft[K]) {
    setDraft((d) => ({ ...d, [key]: value }));
  }

  async function save() {
    const title = draft.title.trim();
    const projectId = draft.projectId.trim();
    if (!title || (!task && !projectId)) {
      setAttempted(true);
      return;
    }
    const labels = parseTags(draft.tagsText);
    const assigneeKey = draft.assignee.trim();
    const assigneeObj = assigneeKey
      ? {
          id: assigneeKey.toLowerCase().replace(/\s+/g, "-"),
          name: assigneeKey,
        }
      : undefined;

    if (task) {
      const patch: UpdateTaskRequest & { id: string } = {
        id: task.id,
        title,
        priority: priorityFromLevel(draft.priority),
        assignee: assigneeObj,
        clearAssignee: !assigneeKey,
        dueDate: draft.deadline || undefined,
        labels,
        description: draft.description,
      };
      await updateTaskMutation.mutateAsync(patch);
      setDraft({ ...draft, title, tagsText: labels.join(", ") });
    } else {
      const created = await createTaskMutation.mutateAsync({
        projectId,
        title,
        status: newStage,
        priority: priorityFromLevel(draft.priority),
        assignee: assigneeObj,
        dueDate: draft.deadline || undefined,
        labels,
        description: draft.description,
      });
      navigate(`/tasks/${created.id}`, { replace: true });
    }
  }

  return (
    <>
      <ControlPanel
        breadcrumbs={[
          { label: "Tasks", to: "/tasks" },
          { label: task?.title || "New" },
        ]}
        actions={
          dirty && (
            <div className="flex items-center gap-2">
              <Button
                variant="primary"
                type="submit"
                form="task-form"
                disabled={
                  createTaskMutation.isPending || updateTaskMutation.isPending
                }
              >
                Save
              </Button>
              <Button
                variant="secondary"
                onClick={() => {
                  if (task) {
                    setDraft(baseline);
                    setAttempted(false);
                  } else {
                    navigate("/tasks");
                  }
                }}
              >
                Discard
              </Button>
            </div>
          )
        }
        right={
          task && (
            <div className="flex items-center gap-2">
              {task.source && task.source !== "manual" && task.sourceStatus && (
                <Tag tone="neutral">{task.sourceStatus}</Tag>
              )}
              {task.source && task.source !== "manual" && task.externalUrl && (
                <a
                  href={task.externalUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs font-medium text-primary hover:underline"
                >
                  {task.externalId || task.source}
                </a>
              )}
              <span className="text-muted">{task.key}</span>
            </div>
          )
        }
      />

      <div className="flex justify-end border-b border-line bg-surface px-4 py-2">
        <Statusbar
          stages={STAGES}
          current={status}
          onSelect={(id) =>
            task
              ? moveTaskMutation.mutate({ id: task.id, status: id })
              : setNewStage(id)
          }
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto grid max-w-6xl gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_380px]">
          <form
            id="task-form"
            noValidate
            onSubmit={(e) => {
              e.preventDefault();
              save();
            }}
            className="flex flex-col gap-6 rounded-lg border border-line bg-surface p-6 shadow-card"
          >
            <div>
              <label htmlFor={ids.title} className="sr-only">
                Title
              </label>
              <input
                id={ids.title}
                className="field !border-0 !border-b !border-line-strong !rounded-none !px-0 text-2xl font-semibold !shadow-none focus:!border-primary"
                placeholder="Task title…"
                value={draft.title}
                aria-invalid={titleInvalid}
                onChange={(e) => set("title", e.target.value)}
              />
              {titleInvalid && (
                <p className="mt-1 text-danger-ink">A title is required.</p>
              )}
            </div>

            <div className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
              <Field id={ids.project} label="Project">
                <select
                  id={ids.project}
                  className="field"
                  required
                  disabled={Boolean(task)}
                  value={draft.projectId}
                  aria-invalid={projectInvalid}
                  onChange={(e) => set("projectId", e.target.value)}
                >
                  <option value="">Select project…</option>
                  {projectOptions.map((p) => (
                    <option key={p.id} value={p.id}>
                      {`${p.name} (${p.slug})`}
                    </option>
                  ))}
                </select>
                {projectInvalid && (
                  <p className="mt-1 text-danger-ink">A project is required.</p>
                )}
              </Field>
              <Field id={ids.assignee} label="Assignee">
                <select
                  id={ids.assignee}
                  className="field"
                  value={draft.assignee}
                  onChange={(e) => set("assignee", e.target.value)}
                >
                  <option value="">Unassigned</option>
                  {assigneeOptions.map((a) => (
                    <option key={a}>{a}</option>
                  ))}
                </select>
              </Field>
              <Field id={ids.deadline} label="Deadline">
                <input
                  id={ids.deadline}
                  type="date"
                  className="field"
                  value={draft.deadline}
                  onChange={(e) => set("deadline", e.target.value)}
                />
              </Field>
              <Field id={ids.tags} label="Tags">
                <input
                  id={ids.tags}
                  className="field"
                  placeholder="backend, agent"
                  value={draft.tagsText}
                  onChange={(e) => set("tagsText", e.target.value)}
                />
              </Field>
              <div className="flex flex-col gap-1.5">
                <span className="font-medium text-muted">Priority</span>
                <Priority
                  value={draft.priority}
                  size={20}
                  onChange={(p) => set("priority", p)}
                />
              </div>
            </div>

            <Field id={ids.desc} label="Description">
              <textarea
                id={ids.desc}
                className="field min-h-40 resize-y"
                placeholder="What needs to happen, and how will we know it is done?"
                value={draft.description}
                onChange={(e) => set("description", e.target.value)}
              />
            </Field>

            {task && prs && prs.length > 0 && (
              <section
                aria-label="Linked Pull Requests"
                className="flex flex-col gap-2 border-t border-line pt-4"
              >
                <h2 className="text-sm font-semibold text-fg">Pull Requests</h2>
                <ul className="divide-y divide-line rounded-md border border-line">
                  {prs.map((pr) => (
                    <li
                      key={pr.id}
                      className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm"
                    >
                      <div className="flex items-center gap-2">
                        {pr.url ? (
                          <a
                            href={pr.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="font-mono font-medium text-primary hover:underline"
                          >
                            #{pr.number}
                          </a>
                        ) : (
                          <span className="font-mono font-medium">
                            #{pr.number}
                          </span>
                        )}
                        <Tag tone={prStateTone(pr.state)}>{pr.state}</Tag>
                        {pr.isDraft && <Tag tone="neutral">draft</Tag>}
                      </div>
                      <div className="flex items-center gap-2">
                        {pr.reviewState && (
                          <Tag tone="neutral">{pr.reviewState}</Tag>
                        )}
                        <Tag tone={checksTone(pr.checksSummary)}>
                          checks: {pr.checksSummary}
                        </Tag>
                      </div>
                    </li>
                  ))}
                </ul>
              </section>
            )}
          </form>

          <aside className="lg:sticky lg:top-4 lg:self-start">
            <Chatter
              messages={messages}
              disabledReason={
                task ? undefined : "Save task to start conversation."
              }
              onPost={(kind, body) =>
                task &&
                addActivityMutation.mutate({
                  id: task.id,
                  kind: kind === "message" ? "comment" : "review",
                  body,
                })
              }
            />
          </aside>
        </div>
      </div>
    </>
  );
}

function prStateTone(state: PRState) {
  switch (state) {
    case "merged":
      return "primary" as const;
    case "closed":
      return "danger" as const;
    default:
      return "success" as const;
  }
}

function checksTone(summary: ChecksSummary) {
  switch (summary) {
    case "passing":
      return "success" as const;
    case "failing":
      return "danger" as const;
    case "pending":
      return "accent" as const;
    default:
      return "neutral" as const;
  }
}

function Field({
  id,
  label,
  children,
}: {
  id: string;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="font-medium text-muted">
        {label}
      </label>
      {children}
    </div>
  );
}
