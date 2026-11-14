import { useId, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

import { ApiError } from "~/shared/api/client";
import { Button, buttonClass } from "~/shared/components/Button";
import { Chatter, type ChatterMessage } from "~/shared/components/Chatter";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Priority } from "~/shared/components/Priority";
import { Statusbar } from "~/shared/components/Statusbar";
import { type TaskInput, useTaskActions } from "../api/actions";
import { useTask, useTaskActivity } from "../api/queries";
import type { Task, TaskStatus } from "../api/types";
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
  /** 0–3, the star widget level. */
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

  // Render the form only once the task is loaded, and remount per record, so
  // the draft starts from real data and never leaks between tasks.
  return <TaskForm key={isNew ? "new" : taskId} task={task} />;
}

function TaskForm({ task }: { task?: Task }) {
  const navigate = useNavigate();
  const { createTask, updateTask, moveTask, addMessage } = useTaskActions();
  const { data: activity } = useTaskActivity(task?.id);
  const ids = {
    title: useId(),
    assignee: useId(),
    deadline: useId(),
    tags: useId(),
    desc: useId(),
  };

  const baseline = useMemo(() => toDraft(task), [task]);
  const [draft, setDraft] = useState<Draft>(baseline);
  const [newStage, setNewStage] = useState<TaskStatus>(STAGES[0].id);
  const [attempted, setAttempted] = useState(false);

  const messages = useMemo<ChatterMessage[]>(
    () =>
      (activity?.items ?? []).map(({ createdAt, ...rest }) => ({
        ...rest,
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
  const status = task?.status ?? newStage;

  function set<K extends keyof Draft>(key: K, value: Draft[K]) {
    setDraft((d) => ({ ...d, [key]: value }));
  }

  function save() {
    const title = draft.title.trim();
    if (!title) {
      setAttempted(true);
      return;
    }
    const labels = parseTags(draft.tagsText);
    const patch: Omit<TaskInput, "status"> = {
      title,
      priority: priorityFromLevel(draft.priority),
      assignee: draft.assignee || null,
      dueDate: draft.deadline || null,
      labels,
      description: draft.description,
    };

    if (task) {
      updateTask(task.id, patch);
      setDraft({ ...draft, title, tagsText: labels.join(", ") });
    } else {
      const id = createTask({ ...patch, status: newStage });
      navigate(`/tasks/${id}`, { replace: true });
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
              <Button variant="primary" type="submit" form="task-form">
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
        right={task && <span className="text-muted">{task.key}</span>}
      />

      <div className="flex justify-end border-b border-line bg-surface px-4 py-2">
        <Statusbar
          stages={STAGES}
          current={status}
          onSelect={(id) => (task ? moveTask(task.id, id) : setNewStage(id))}
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
          </form>

          <aside className="lg:sticky lg:top-4 lg:self-start">
            <Chatter
              messages={messages}
              disabledReason={
                task ? undefined : "Save the task to start the conversation."
              }
              onPost={(kind, body) => task && addMessage(task.id, kind, body)}
            />
          </aside>
        </div>
      </div>
    </>
  );
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
