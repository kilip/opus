import { queryOptions, useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { RUN_ERRORS } from "./errors";
import { runKeys } from "./keys";
import type { RunItem, RunState } from "./types";

export const runsQueryOptions = (state?: RunState) =>
  queryOptions({
    queryKey: runKeys.list(state),
    queryFn: async ({ signal }): Promise<RunItem[]> =>
      unwrap(
        api.GET("/runs", {
          params: state ? { query: { state } } : undefined,
          signal,
        }),
        RUN_ERRORS,
      ),
    refetchInterval: 10_000,
  });

export function useRuns(state?: RunState) {
  return useQuery(runsQueryOptions(state));
}

export const runQueryOptions = (id: string) =>
  queryOptions({
    queryKey: runKeys.detail(id),
    queryFn: async ({ signal }): Promise<RunItem> =>
      unwrap(
        api.GET("/runs/{id}", {
          params: { path: { id } },
          signal,
        }),
        RUN_ERRORS,
      ),
    refetchInterval: 5_000,
  });

export function useRun(id: string) {
  return useQuery(runQueryOptions(id));
}
