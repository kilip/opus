/** Query key factory: every task query is nested under `["tasks"]`. */
export const taskKeys = {
  all: ["tasks"] as const,
  list: () => [...taskKeys.all, "list"] as const,
  detail: (id: string) => [...taskKeys.all, "detail", id] as const,
  activity: (id: string) =>
    [...taskKeys.all, "detail", id, "activity"] as const,
};
