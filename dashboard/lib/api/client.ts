import { ApiResponse } from "./types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const response = await fetch(`${API_BASE_URL}/api/v1${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    credentials: "include",
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      success: false,
      data: null,
      error: { code: "UNKNOWN_ERROR", message: "An unexpected error occurred" },
    }));
    return error as ApiResponse<T>;
  }

  return response.json();
}

export const apiClient = {
  get: <T>(path: string, options?: RequestInit) =>
    request<T>(path, { method: "GET", ...options }),
  post: <T>(path: string, body?: unknown, options?: RequestInit) =>
    request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      ...options,
    }),
};
