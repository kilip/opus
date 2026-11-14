import type { StageDef } from "~/shared/components/Statusbar";
import type { Tone } from "~/shared/components/Tag";
import type { TaskPriority, TaskStatus } from "./api/types";

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
  { id: "spec", name: "Spec", tone: "info" },
  { id: "inProgress", name: "In Progress", tone: "accent" },
  { id: "inReview", name: "In Review", tone: "primary" },
  { id: "done", name: "Done", tone: "success" },
];

export function stageName(status: TaskStatus) {
  return STAGES.find((s) => s.id === status)?.name ?? status;
}

/** Lowest first; the index is the level shown by the 0–3 star widget. */
export const PRIORITIES: TaskPriority[] = ["none", "low", "medium", "high"];

export function priorityLevel(priority: TaskPriority) {
  return PRIORITIES.indexOf(priority);
}

export function priorityFromLevel(level: number): TaskPriority {
  return PRIORITIES[Math.min(Math.max(level, 0), PRIORITIES.length - 1)];
}
