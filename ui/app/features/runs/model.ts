import type { Tone } from "~/shared/components/Tag";
import type { RunItem, RunState } from "./api/types";

export const RUN_STATES: RunState[] = [
  "queued",
  "preparing",
  "running",
  "merged",
  "failed",
  "needs-info",
  "paused",
  "cancelled",
];

export const STATE_TONES: Record<RunState, Tone> = {
  queued: "neutral",
  preparing: "info",
  running: "primary",
  merged: "success",
  failed: "danger",
  "needs-info": "accent",
  paused: "accent",
  cancelled: "neutral",
};

export function isActiveRunState(state: RunState): boolean {
  return (
    state === "queued" ||
    state === "preparing" ||
    state === "running" ||
    state === "paused"
  );
}

export type TimelineStep = {
  state: RunState;
  label: string;
  status: "done" | "current" | "pending" | "failed";
};

const HAPPY_PATH: RunState[] = ["queued", "preparing", "running", "merged"];

/** Build the ordered state timeline for a Run detail view. */
export function buildStateTimeline(
  run: Pick<RunItem, "state">,
): TimelineStep[] {
  const cur: RunState = run.state ?? "queued";
  if (
    cur === "failed" ||
    cur === "needs-info" ||
    cur === "paused" ||
    cur === "cancelled"
  ) {
    return [
      { state: "queued", label: "queued", status: "done" },
      { state: "preparing", label: "preparing", status: "done" },
      { state: "running", label: "running", status: "done" },
      {
        state: cur,
        label: cur,
        status: cur === "paused" ? "current" : "failed",
      },
    ];
  }

  const idx = HAPPY_PATH.indexOf(cur);
  return HAPPY_PATH.map((st, i) => ({
    state: st,
    label: st,
    status: i < idx ? "done" : i === idx ? "current" : "pending",
  }));
}

export type ToolCallPayload = {
  tool: string;
  args?: Record<string, unknown>;
  status: "ok" | "error" | string;
  error?: string;
  duration_ms?: number;
};

/** Parse a `tool_call` session event payload, returning null if not a valid tool call. */
export function parseToolCallPayload(payload: unknown): ToolCallPayload | null {
  let obj = payload;
  if (typeof obj === "string") {
    try {
      obj = JSON.parse(obj);
    } catch {
      return null;
    }
  }
  if (!obj || typeof obj !== "object" || Array.isArray(obj)) {
    return null;
  }
  const rec = obj as Record<string, unknown>;
  if (typeof rec.tool !== "string" || rec.tool.trim() === "") {
    return null;
  }
  return {
    tool: rec.tool,
    args:
      rec.args && typeof rec.args === "object" && !Array.isArray(rec.args)
        ? (rec.args as Record<string, unknown>)
        : undefined,
    status: typeof rec.status === "string" && rec.status ? rec.status : "ok",
    error: typeof rec.error === "string" && rec.error ? rec.error : undefined,
    duration_ms:
      typeof rec.duration_ms === "number" ? rec.duration_ms : undefined,
  };
}

export type RunReportCriterion = {
  id?: string;
  criterion?: string;
  met: boolean;
  evidence?: string;
};

export type RunReport = {
  summary: string;
  pr_url?: string;
  commit_sha?: string;
  assumptions?: string[];
  acceptance_criteria?: RunReportCriterion[];
};

/** Parse `Run.report` JSON into a structured RunReport, or null when empty. */
export function parseRunReport(report: unknown): RunReport | null {
  let obj = report;
  if (typeof obj === "string") {
    if (obj.trim() === "") return null;
    try {
      obj = JSON.parse(obj);
    } catch {
      return null;
    }
  }
  if (!obj || typeof obj !== "object" || Array.isArray(obj)) {
    return null;
  }
  const rec = obj as Record<string, unknown>;
  const summary = typeof rec.summary === "string" ? rec.summary : "";
  const pr_url =
    typeof rec.pr_url === "string" && rec.pr_url ? rec.pr_url : undefined;
  const commit_sha =
    typeof rec.commit_sha === "string" && rec.commit_sha
      ? rec.commit_sha
      : undefined;
  const assumptions = Array.isArray(rec.assumptions)
    ? rec.assumptions.filter((a): a is string => typeof a === "string")
    : undefined;
  const acceptance_criteria = Array.isArray(rec.acceptance_criteria)
    ? rec.acceptance_criteria
        .filter(
          (c): c is Record<string, unknown> =>
            Boolean(c) && typeof c === "object" && !Array.isArray(c),
        )
        .map((c) => ({
          id: typeof c.id === "string" && c.id ? c.id : undefined,
          criterion:
            typeof c.criterion === "string" && c.criterion
              ? c.criterion
              : undefined,
          met: Boolean(c.met),
          evidence:
            typeof c.evidence === "string" && c.evidence
              ? c.evidence
              : undefined,
        }))
    : undefined;

  if (
    !summary &&
    !pr_url &&
    !commit_sha &&
    (!assumptions || assumptions.length === 0) &&
    (!acceptance_criteria || acceptance_criteria.length === 0)
  ) {
    return null;
  }

  return {
    summary,
    pr_url,
    commit_sha,
    assumptions,
    acceptance_criteria,
  };
}
