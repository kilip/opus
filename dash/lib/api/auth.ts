"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthContext } from "./AuthContext";
import { apiClient } from "./client";
import type { AuthTokens } from "./types";

export function useRefreshToken() {
  return useMutation({
    mutationFn: () => apiClient.post<AuthTokens>("/auth/refresh"),
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  const { setAccessToken } = useAuthContext();

  return useMutation({
    mutationFn: () => apiClient.post("/auth/logout"),
    onSuccess: () => {
      setAccessToken(null);
      queryClient.invalidateQueries();
    },
  });
}
