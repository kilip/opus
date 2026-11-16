/** Query key factory: every node query is nested under `["nodes"]`. */
export const nodeKeys = {
	all: ["nodes"] as const,
	list: () => [...nodeKeys.all, "list"] as const,
};
