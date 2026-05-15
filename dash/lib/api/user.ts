"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "./client";
import type { User } from "./types";

export function useCurrentUser(accessToken: string | null) {
  return useQuery({
    queryKey: ["user", "me"],
    queryFn: () =>
      apiClient.get<User>("/user/me", {
        headers: { Authorization: `Bearer ${accessToken}` },
      }),
    enabled: !!accessToken,
  });
}
