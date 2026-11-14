import { queryOptions, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { TASK_ERRORS } from "./errors";
import { taskKeys } from "./keys";
import type { Activity, Task } from "./types";

export const tasksQueryOptions = () =>
  queryOptions({
    queryKey: taskKeys.list(),
    queryFn: async ({ signal }): Promise<Task[]> =>
      unwrap(api.GET("/tasks" as never, { signal } as never), TASK_ERRORS),
  });

export const taskQueryOptions = (id: string) =>
  queryOptions({
    queryKey: taskKeys.detail(id),
    queryFn: async ({ signal }): Promise<Task> =>
      unwrap(
        api.GET(
          "/tasks/{id}" as never,
          {
            params: { path: { id } },
            signal,
          } as never,
        ),
        TASK_ERRORS,
      ),
  });

export const taskActivityQueryOptions = (id: string) =>
  queryOptions({
    queryKey: taskKeys.activity(id),
    queryFn: async ({ signal }): Promise<Activity[]> =>
      unwrap(
        api.GET(
          "/tasks/{id}/activity" as never,
          {
            params: { path: { id } },
            signal,
          } as never,
        ),
        TASK_ERRORS,
      ),
  });

export const taskListQuery = tasksQueryOptions;
export const taskQuery = taskQueryOptions;
export const taskActivityQuery = taskActivityQueryOptions;

export function useTasks() {
  return useQuery(tasksQueryOptions());
}

/** Fetch a single task. Pass `undefined` to disable the query (e.g. for a new task). */
export function useTask(id: string | undefined) {
  const queryClient = useQueryClient();
  return useQuery({
    ...taskQueryOptions(id ?? ""),
    enabled: Boolean(id),
    // Opening a task from the board needs no extra request.
    initialData: () =>
      queryClient
        .getQueryData<Task[]>(taskKeys.list())
        ?.find((task) => task.id === id),
  });
}

export function useTaskActivity(id: string | undefined) {
  return useQuery({
    ...taskActivityQueryOptions(id ?? ""),
    enabled: Boolean(id),
  });
}
