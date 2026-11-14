import { QueryClient } from "@tanstack/react-query";

import { ApiError } from "./client";

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        // A 4xx will not fix itself, so only retry network and 5xx failures.
        retry: (failureCount, error) =>
          failureCount < 2 &&
          !(
            error instanceof ApiError &&
            error.status >= 400 &&
            error.status < 500
          ),
      },
    },
  });
}
