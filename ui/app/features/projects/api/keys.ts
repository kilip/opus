/** Query key factory: every project query is nested under `["projects"]`. */
export const projectKeys = {
	all: ["projects"] as const,
	list: () => [...projectKeys.all, "list"] as const,
	detail: (id: string) => [...projectKeys.all, "detail", id] as const,
	nodes: (id: string) => [...projectKeys.all, "detail", id, "nodes"] as const,
};
