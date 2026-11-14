import createClient from "openapi-fetch";

import type { paths } from "./schema";

/** Typed API client. Paths are relative to the Fiber `/api` base path. */
export const api = createClient<paths>({ baseUrl: "/api" });

/** A non-2xx API response, or a 2xx response without a body. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

function messageOf(error: unknown): string | undefined {
  if (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof error.message === "string"
  ) {
    return error.message;
  }
  return undefined;
}

/** Return the body of a successful response, or throw an {@link ApiError}. */
export function unwrap<T>({
  data,
  error,
  response,
}: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (response.ok && data !== undefined) return data;
  throw new ApiError(
    response.status,
    messageOf(error) ?? (response.statusText || "Request failed"),
  );
}
