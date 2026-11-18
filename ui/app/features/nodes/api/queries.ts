import { queryOptions, useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { NODE_ERRORS } from "./errors";
import { nodeKeys } from "./keys";
import type { NodeItem } from "./types";

export const nodesQueryOptions = () =>
  queryOptions({
    queryKey: nodeKeys.list(),
    queryFn: async ({ signal }): Promise<NodeItem[]> =>
      unwrap(api.GET("/nodes", { signal }), NODE_ERRORS),
    refetchInterval: 15_000,
  });

export function useNodes() {
  return useQuery(nodesQueryOptions());
}
