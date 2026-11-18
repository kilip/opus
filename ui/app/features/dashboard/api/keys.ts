/** Query key factory: every dashboard query is nested under `["dashboard"]`. */
export const dashboardKeys = {
	all: ["dashboard"] as const,
	nodes: () => [...dashboardKeys.all, "nodes"] as const,
	tasks: () => [...dashboardKeys.all, "tasks"] as const,
	runs: () => [...dashboardKeys.all, "runs"] as const,
};
