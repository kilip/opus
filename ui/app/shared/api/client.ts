import createClient from "openapi-fetch";

import { resolveErrorMessage } from "./errors";
import type { paths } from "./schema";

/** Typed API client. Paths are relative to the Fiber `/api` base path. */
export const api = createClient<paths>({ baseUrl: "/api" });

export interface ApiMeta {
  total?: number;
  nextCursor?: string;
}

export interface ApiEnvelope<T> {
  data: T;
  meta?: ApiMeta;
  error?: string;
}

/** A non-2xx API response, or a 2xx response without a body. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message?: string) {
    super(message || code);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

function extractErrorCode(error: unknown, status: number): string {
  if (typeof error === "object" && error !== null) {
    if ("error" in error && typeof error.error === "string" && error.error) {
      return error.error;
    }
    if (
      "message" in error &&
      typeof error.message === "string" &&
      error.message
    ) {
      return error.message;
    }
  }
  return status === 404 ? "RESOURCE_NOT_FOUND" : "REQUEST_FAILED";
}

/** Return the data payload of a successful envelope response, or throw an {@link ApiError}. */
export async function unwrap<T>(
  promiseOrResult:
    | Promise<{ data?: unknown; error?: unknown; response: Response }>
    | { data?: unknown; error?: unknown; response: Response },
  featureErrors?: Record<string, string>,
): Promise<T> {
  const { data, error, response } = await promiseOrResult;
  if (response.ok && data !== undefined) {
    if (typeof data === "object" && data !== null && "data" in data) {
      const payload: unknown = data.data;
      return payload as T;
    }
    return data as T;
  }
  const code = extractErrorCode(error, response.status);
  const message = resolveErrorMessage(code, featureErrors);
  throw new ApiError(response.status, code, message);
}

/** Return data and meta of a successful envelope response, or throw an {@link ApiError}. */
export async function unwrapMeta<T>(
  promiseOrResult:
    | Promise<{ data?: unknown; error?: unknown; response: Response }>
    | { data?: unknown; error?: unknown; response: Response },
  featureErrors?: Record<string, string>,
): Promise<{ data: T; meta?: ApiMeta }> {
  const { data, error, response } = await promiseOrResult;
  if (response.ok && data !== undefined) {
    if (typeof data === "object" && data !== null && "data" in data) {
      const payload: unknown = data.data;
      const metaVal: unknown = "meta" in data ? data.meta : undefined;
      return { data: payload as T, meta: metaVal as ApiMeta | undefined };
    }
    return { data: data as T };
  }
  const code = extractErrorCode(error, response.status);
  const message = resolveErrorMessage(code, featureErrors);
  throw new ApiError(response.status, code, message);
}
