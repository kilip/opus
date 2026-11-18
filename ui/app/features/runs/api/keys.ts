import type { RunState } from "./types";

/** Query key factory: every run query is nested under `["runs"]`. */
export const runKeys = {
	all: ["runs"] as const,
	list: (state?: RunState) =>
		state
			? ([...runKeys.all, "list", state] as const)
			: ([...runKeys.all, "list"] as const),
	detail: (id: string) => [...runKeys.all, "detail", id] as const,
};
