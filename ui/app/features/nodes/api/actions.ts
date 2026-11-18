import { useMutation, useQueryClient } from "@tanstack/react-query";

import { ApiError, api, unwrap } from "~/shared/api/client";
import { resolveErrorMessage } from "~/shared/api/errors";
import { NODE_ERRORS } from "./errors";
import { nodeKeys } from "./keys";
import type { JoinTokenResponse } from "./types";

export function useCreateJoinToken() {
  return useMutation({
    mutationFn: async (): Promise<JoinTokenResponse> =>
      unwrap(api.POST("/nodes/join-tokens"), NODE_ERRORS),
  });
}

export function useDeleteNode() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { error, response } = await api.DELETE("/nodes/{id}", {
        params: { path: { id } },
      });
      if (!response.ok) {
        const code =
          error &&
          typeof error === "object" &&
          "error" in error &&
          typeof error.error === "string"
            ? error.error
            : response.status === 404
              ? "RESOURCE_NOT_FOUND"
              : "REQUEST_FAILED";
        throw new ApiError(
          response.status,
          code,
          resolveErrorMessage(code, NODE_ERRORS),
        );
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: nodeKeys.all });
    },
  });
}
