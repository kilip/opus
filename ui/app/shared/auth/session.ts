import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

import { api, unwrap } from "~/shared/api/client";
import type { components } from "~/shared/api/schema";

export type User = components["schemas"]["domain.UserResponse"];
export type SetupStatus = components["schemas"]["domain.SetupStatusResponse"];

export const sessionKeys = {
  all: ["session"] as const,
  me: () => [...sessionKeys.all, "me"] as const,
  setupStatus: () => [...sessionKeys.all, "setup-status"] as const,
};

export function useSession() {
  return useQuery({
    queryKey: sessionKeys.me(),
    queryFn: async ({ signal }): Promise<User> =>
      unwrap(api.GET("/auth/me", { signal })),
    retry: false,
  });
}

export function useSetupStatus() {
  return useQuery({
    queryKey: sessionKeys.setupStatus(),
    queryFn: async ({ signal }): Promise<SetupStatus> =>
      unwrap(api.GET("/setup/status", { signal })),
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: async () => {
      const { response } = await api.POST("/auth/logout");
      if (!response.ok) {
        throw new Error("Logout failed");
      }
    },
    onSettled: () => {
      queryClient.removeQueries({ queryKey: sessionKeys.all });
      navigate("/login");
    },
  });
}
