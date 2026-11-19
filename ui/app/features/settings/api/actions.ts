import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { settingsKeys } from "./keys";
import type { GitHubTokenStatus, SetGitHubTokenRequest } from "./types";

export function useSetGitHubToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (
      body: SetGitHubTokenRequest,
    ): Promise<GitHubTokenStatus> =>
      unwrap(api.PUT("/settings/github-token", { body })),
    onSuccess: (data) => {
      queryClient.setQueryData(settingsKeys.githubToken(), data);
    },
  });
}

export function useDeleteGitHubToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<void> => {
      const res = await api.DELETE("/settings/github-token");
      if (res.error) {
        await unwrap(Promise.resolve(res));
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: settingsKeys.githubToken(),
      });
    },
  });
}
