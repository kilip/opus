"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "./client";
import type { AuthTokens } from "./types";

export function useRefreshToken() {
  return useMutation({
    mutationFn: () => apiClient.post<AuthTokens>("/auth/refresh"),
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => apiClient.post("/auth/logout"),
    onSuccess: () => {
      queryClient.invalidateQueries();
    },
  });
}
