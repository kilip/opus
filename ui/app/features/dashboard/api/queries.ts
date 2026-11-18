import { useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { dashboardKeys } from "./keys";
import type { NodeItem, RunItem, TaskItem } from "./types";

export function useDashboardNodes() {
  return useQuery({
    queryKey: dashboardKeys.nodes(),
    queryFn: async ({ signal }): Promise<NodeItem[]> =>
      unwrap(api.GET("/nodes", { signal })),
    refetchInterval: 15_000,
  });
}

export function useDashboardTasks() {
  return useQuery({
    queryKey: dashboardKeys.tasks(),
    queryFn: async ({ signal }): Promise<TaskItem[]> =>
      unwrap(api.GET("/tasks", { signal })),
    refetchInterval: 15_000,
  });
}

export function useDashboardRuns() {
  return useQuery({
    queryKey: dashboardKeys.runs(),
    queryFn: async ({ signal }): Promise<RunItem[]> =>
      unwrap(api.GET("/runs", { signal })),
    refetchInterval: 10_000,
  });
}
