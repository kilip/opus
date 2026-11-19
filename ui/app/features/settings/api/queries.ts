import { queryOptions, useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { settingsKeys } from "./keys";
import type { GitHubTokenStatus } from "./types";

export const githubTokenQueryOptions = () =>
  queryOptions({
    queryKey: settingsKeys.githubToken(),
    queryFn: async ({ signal }): Promise<GitHubTokenStatus> =>
      unwrap(api.GET("/settings/github-token", { signal })),
  });

export function useGitHubTokenStatus() {
  return useQuery(githubTokenQueryOptions());
}
