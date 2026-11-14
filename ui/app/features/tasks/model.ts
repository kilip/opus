import { isAgent } from "~/shared/components/Avatar";
import type { FilterDef } from "~/shared/components/SearchBar";
import type { StageDef } from "~/shared/components/Statusbar";
import type { Tone } from "~/shared/components/Tag";
import { todayISO } from "~/shared/util/dates";
import type { Task, TaskPriority, TaskStatus } from "./api/types";
/** Display name of the signed-in user (there is no auth yet). */
export const CURRENT_USER = "You";

export const ASSIGNEES = [
  CURRENT_USER,
  "planner-agent",
  "coder-agent",
  "reviewer-agent",
];

export type TaskStage = StageDef<TaskStatus> & { tone: Tone };

/** Board columns, in workflow order. Every `TaskStatus` needs an entry here. */
export const STAGES: TaskStage[] = [
  { id: "backlog", name: "Backlog", tone: "neutral" },
  { id: "todo", name: "To Do", tone: "info" },
  { id: "in_progress", name: "In Progress", tone: "accent" },
  { id: "review", name: "Review", tone: "primary" },
  { id: "done", name: "Done", tone: "success" },
];

export function stageName(status: TaskStatus) {
  return STAGES.find((s) => s.id === status)?.name ?? status;
}

/** Lowest first; the index is the level shown by the 0–3 star widget. */
export const PRIORITIES: TaskPriority[] = ["0", "1", "2", "3"];

export function priorityLevel(priority: TaskPriority) {
  const idx = PRIORITIES.indexOf(priority);
  return Math.max(idx, 0);
}

export function priorityFromLevel(level: number): TaskPriority {
  const clamped = Math.min(Math.max(level, 0), PRIORITIES.length - 1);
  return PRIORITIES[clamped] ?? "0";
}

export const TASK_FILTERS: FilterDef[] = [
  { id: "mine", label: "My tasks" },
  { id: "agents", label: "Assigned agents" },
  { id: "high", label: "High priority" },
  { id: "overdue", label: "Overdue" },
];

const predicates: Record<string, (t: Task, today: string) => boolean> = {
  mine: (t) => t.assignee?.name === CURRENT_USER,
  agents: (t) => t.assignee !== undefined && isAgent(t.assignee.name),
  high: (t) => priorityLevel(t.priority) >= 2,
  overdue: (t, today) =>
    t.status !== "done" && t.dueDate !== undefined && t.dueDate < today,
};

/** Text search AND (any selected filters). */
export function filterTasks(
  tasks: Task[],
  query: string,
  active: ReadonlySet<string>,
): Task[] {
  const q = query.trim().toLowerCase();
  const today = todayISO();
  const selected = [...active].map((id) => predicates[id]).filter(Boolean);

  return tasks.filter((t) => {
    if (selected.length > 0 && !selected.some((p) => p(t, today))) return false;
    if (!q) return true;
    return [t.title, t.key, t.assignee?.name ?? "", ...t.labels].some((s) =>
      s.toLowerCase().includes(q),
    );
  });
}
