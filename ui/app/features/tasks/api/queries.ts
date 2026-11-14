import { queryOptions, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { taskKeys } from "./keys";
import type { TaskList } from "./types";

/*
 * Interim: the API is read-only and serves static data, and writes are applied
 * to this cache locally (see actions.ts). Cached tasks therefore never go stale
 * on their own. Once write endpoints exist, remove this and invalidate
 * `taskKeys` after each mutation instead.
 */
const staleTime = Number.POSITIVE_INFINITY;

export const taskListQuery = () =>
  queryOptions({
    queryKey: taskKeys.list(),
    queryFn: async ({ signal }) => unwrap(await api.GET("/tasks", { signal })),
    staleTime,
  });

export const taskQuery = (id: string) =>
  queryOptions({
    queryKey: taskKeys.detail(id),
    queryFn: async ({ signal }) =>
      unwrap(
        await api.GET("/tasks/{id}", { params: { path: { id } }, signal }),
      ),
    staleTime,
  });

export const taskActivityQuery = (id: string) =>
  queryOptions({
    queryKey: taskKeys.activity(id),
    queryFn: async ({ signal }) =>
      unwrap(
        await api.GET("/tasks/{id}/activity", {
          params: { path: { id } },
          signal,
        }),
      ),
    staleTime,
  });

export function useTasks() {
  return useQuery(taskListQuery());
}

/** A single task. Pass `undefined` to disable the query (e.g. a new task). */
export function useTask(id: string | undefined) {
  const queryClient = useQueryClient();
  return useQuery({
    ...taskQuery(id ?? ""),
    enabled: Boolean(id),
    // Opening a task from the board needs no extra request.
    initialData: () =>
      queryClient
        .getQueryData<TaskList>(taskKeys.list())
        ?.items.find((task) => task.id === id),
  });
}

export function useTaskActivity(id: string | undefined) {
  return useQuery({ ...taskActivityQuery(id ?? ""), enabled: Boolean(id) });
}
