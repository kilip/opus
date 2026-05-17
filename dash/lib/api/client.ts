import type { ApiResponse } from "./types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

let inMemoryToken: string | null = null;
// Use a callback to trigger navigation or auth state reset
let onUnauthorized: () => void = () => {};

export const setAuthToken = (token: string | null) => {
  inMemoryToken = token;
};

export const setOnUnauthorized = (callback: () => void) => {
  onUnauthorized = callback;
};

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<ApiResponse<T>> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  if (inMemoryToken) {
    headers.Authorization = `Bearer ${inMemoryToken}`;
  }

  const url = `${API_BASE_URL}${path}`;

  const response = await fetch(url, {
    ...options,
    headers,
    credentials: "include",
  });

  if (response.status === 401) {
    onUnauthorized();
    return {
      success: false,
      data: null,
      error: { code: "UNAUTHORIZED", message: "Session expired" },
    };
  }

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
