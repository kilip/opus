import { isAgent } from "~/shared/components/Avatar";
import type { FilterDef } from "~/shared/components/SearchBar";
import type { Task } from "./api/types";
import { todayISO } from "./dates";
import { CURRENT_USER, priorityLevel } from "./model";

export const TASK_FILTERS: FilterDef[] = [
  { id: "mine", label: "My tasks" },
  { id: "agents", label: "Assigned to agents" },
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

/** Text search AND (any of the selected filters). */
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
