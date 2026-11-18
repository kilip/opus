import type { NodeItem, RunItem, TaskItem, TaskStatus } from "./api/types";

export const TASK_STATUS_ORDER: TaskStatus[] = [
  "backlog",
  "todo",
  "in-progress",
  "review",
  "blocked",
  "done",
];

export type DashboardSummary = {
  onlineNodes: NodeItem[];
  totalNodes: number;
  activeRuns: RunItem[];
  recentOutcomes: RunItem[];
  taskCounts: Record<TaskStatus, number>;
};

export function summarizeDashboard(
  nodes: NodeItem[],
  tasks: TaskItem[],
  runs: RunItem[],
): DashboardSummary {
  const onlineNodes = nodes.filter((n) => n.status === "online");

  const taskCounts: Record<TaskStatus, number> = {
    backlog: 0,
    todo: 0,
    "in-progress": 0,
    review: 0,
    blocked: 0,
    done: 0,
  };
  for (const t of tasks) {
    if (t.status in taskCounts) {
      taskCounts[t.status]++;
    }
  }

  const activeRuns: RunItem[] = [];
  const finishedRuns: RunItem[] = [];
  for (const r of runs) {
    const st = r.state ?? "queued";
    if (
      st === "queued" ||
      st === "preparing" ||
      st === "running" ||
      st === "paused"
    ) {
      activeRuns.push(r);
    } else {
      finishedRuns.push(r);
    }
  }

  finishedRuns.sort((a, b) =>
    (b.endedAt ?? b.updatedAt ?? "").localeCompare(
      a.endedAt ?? a.updatedAt ?? "",
    ),
  );

  return {
    onlineNodes,
    totalNodes: nodes.length,
    activeRuns,
    recentOutcomes: finishedRuns.slice(0, 5),
    taskCounts,
  };
}
