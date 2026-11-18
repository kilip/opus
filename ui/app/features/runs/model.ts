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
