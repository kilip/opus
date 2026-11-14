import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

import { CURRENT_USER, stageName } from "../model";
import { taskKeys } from "./keys";
import { taskActivityQuery } from "./queries";
import type {
  Activity,
  ActivityKind,
  ActivityList,
  Task,
  TaskList,
  TaskPriority,
  TaskStatus,
} from "./types";

/*
 * INTERIM, LOCAL-ONLY WRITES.
 *
 * The API is GET-only for now, so these actions edit the TanStack Query cache
 * and nothing is persisted: a reload restores the server data. They keep the
 * board's create / edit / drag / chatter interactions working. When write
 * endpoints land, replace each action with a `useMutation` that calls the API
 * and invalidates `taskKeys`; the hook's return shape can stay the same.
 */

export type TaskInput = {
  title: string;
  status: TaskStatus;
  priority: TaskPriority;
  /** Assignee display name; `null` for unassigned. */
  assignee: string | null;
  /** YYYY-MM-DD */
  dueDate: string | null;
  labels: string[];
  description: string;
};

let seq = 0;

function localId() {
  seq += 1;
  return `local-${Date.now().toString(36)}-${seq}`;
}

function toAssignee(name: string | null): Task["assignee"] {
  return name
    ? { id: name.toLowerCase().replace(/\s+/g, "-"), name }
    : undefined;
}

function activity(kind: ActivityKind, body: string): Activity {
  return {
    id: localId(),
    kind,
    author: CURRENT_USER,
    body,
    createdAt: new Date().toISOString(),
  };
}

const tracking = (body: string) => activity("tracking", body);

function nextKeyNumber(tasks: Task[]) {
  let max = 0;
  for (const task of tasks) {
    const match = /^OPUS-(\d+)$/.exec(task.key);
    if (match) max = Math.max(max, Number(match[1]));
  }
  return max + 1;
}

function findTask(queryClient: QueryClient, id: string) {
  return (
    queryClient.getQueryData<Task>(taskKeys.detail(id)) ??
    queryClient
      .getQueryData<TaskList>(taskKeys.list())
      ?.items.find((task) => task.id === id)
  );
}

/** Apply `patch` to a task in both the list and the detail cache. */
function patchTask(
  queryClient: QueryClient,
  id: string,
  patch: (task: Task) => Task,
) {
  queryClient.setQueryData<TaskList>(
    taskKeys.list(),
    (old) =>
      old && {
        ...old,
        items: old.items.map((task) => (task.id === id ? patch(task) : task)),
      },
  );
  queryClient.setQueryData<Task>(
    taskKeys.detail(id),
    (old) => old && patch(old),
  );
}

async function appendActivity(
  queryClient: QueryClient,
  id: string,
  entries: Activity[],
) {
  if (entries.length === 0) return;
  try {
    // Make sure the server timeline is in the cache first, so local entries
    // are appended to it instead of replacing it.
    await queryClient.ensureQueryData(taskActivityQuery(id));
  } catch {
    // Local tracking is best effort; skip it if the timeline cannot load.
    return;
  }
  queryClient.setQueryData<ActivityList>(taskKeys.activity(id), (old) => ({
    items: [...(old?.items ?? []), ...entries],
  }));
}

function createTask(queryClient: QueryClient, input: TaskInput): string {
  const items =
    queryClient.getQueryData<TaskList>(taskKeys.list())?.items ?? [];
  const number = nextKeyNumber(items);
  const now = new Date().toISOString();

  const task: Task = {
    id: `local-${number}`,
    key: `OPUS-${number}`,
    source: "custom",
    title: input.title,
    description: input.description,
    status: input.status,
    priority: input.priority,
    type: "feature",
    assignee: toAssignee(input.assignee),
    dueDate: input.dueDate ?? undefined,
    labels: input.labels,
    createdAt: now,
    updatedAt: now,
  };

  queryClient.setQueryData<TaskList>(taskKeys.list(), (old) => ({
    items: [...(old?.items ?? []), task],
    total: (old?.total ?? 0) + 1,
  }));
  queryClient.setQueryData(taskKeys.detail(task.id), task);
  queryClient.setQueryData<ActivityList>(taskKeys.activity(task.id), {
    items: [tracking("Task created")],
  });
  return task.id;
}

function updateTask(
  queryClient: QueryClient,
  id: string,
  patch: Partial<TaskInput>,
) {
  const current = findTask(queryClient, id);
  if (!current) return;

  const { assignee, dueDate, ...rest } = patch;
  const entries: Activity[] = [];

  if (patch.status && patch.status !== current.status) {
    entries.push(
      tracking(
        `Stage: ${stageName(current.status)} → ${stageName(patch.status)}`,
      ),
    );
  }
  if (assignee !== undefined && assignee !== (current.assignee?.name ?? null)) {
    entries.push(
      tracking(
        `Assignee: ${current.assignee?.name ?? "Unassigned"} → ${assignee ?? "Unassigned"}`,
      ),
    );
  }

  patchTask(queryClient, id, (task) => ({
    ...task,
    ...rest,
    ...(assignee !== undefined && { assignee: toAssignee(assignee) }),
    ...(dueDate !== undefined && { dueDate: dueDate ?? undefined }),
    updatedAt: new Date().toISOString(),
  }));
  void appendActivity(queryClient, id, entries);
}

function moveTask(queryClient: QueryClient, id: string, status: TaskStatus) {
  if (findTask(queryClient, id)?.status !== status) {
    updateTask(queryClient, id, { status });
  }
}

function addMessage(
  queryClient: QueryClient,
  id: string,
  kind: "message" | "note",
  body: string,
) {
  void appendActivity(queryClient, id, [activity(kind, body)]);
}

/** Write actions for tasks. See the note at the top of this file. */
export function useTaskActions() {
  const queryClient = useQueryClient();
  return useMemo(
    () => ({
      createTask: (input: TaskInput) => createTask(queryClient, input),
      updateTask: (id: string, patch: Partial<TaskInput>) =>
        updateTask(queryClient, id, patch),
      moveTask: (id: string, status: TaskStatus) =>
        moveTask(queryClient, id, status),
      addMessage: (id: string, kind: "message" | "note", body: string) =>
        addMessage(queryClient, id, kind, body),
    }),
    [queryClient],
  );
}
