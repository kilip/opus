/** Query key factory: every task query is nested under `["tasks"]`. */
export const taskKeys = {
  all: ["tasks"] as const,
  list: (params?: { projectId?: string }) =>
    params?.projectId
      ? ([...taskKeys.all, "list", { projectId: params.projectId }] as const)
      : ([...taskKeys.all, "list"] as const),
  projects: () => [...taskKeys.all, "projects"] as const,
  detail: (id: string) => [...taskKeys.all, "detail", id] as const,
  activity: (id: string) =>
    [...taskKeys.all, "detail", id, "activity"] as const,
  prs: (id: string) => [...taskKeys.all, "detail", id, "prs"] as const,
};
