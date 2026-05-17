// dash/src/shared/utils/api.ts
const BASE_URL = import.meta.env?.VITE_API_URL ?? 'http://localhost:8080';

/**
 * Shared API client for making HTTP requests to the backend server.
 * Handles automatic base URL configuration, headers, and standard error handling.
 */
export const api = {
  /**
   * Performs a GET request to the specified API path.
   * @template T The expected response data type.
   * @param path The relative API path (e.g., '/api/agents').
   * @returns A promise that resolves to the response data.
   */
  get: async <T>(path: string): Promise<T> => {
    const res = await fetch(`${BASE_URL}${path}`);
    if (!res.ok) throw new Error(res.statusText);
    return res.json() as Promise<T>;
  },

  /**
   * Performs a POST request with a JSON payload to the specified API path.
   * @template T The expected response data type.
   * @param path The relative API path.
   * @param body The payload to be JSON-serialized and sent.
   * @returns A promise that resolves to the response data.
   */
  post: async <T>(path: string, body: unknown): Promise<T> => {
    const res = await fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(res.statusText);
    return res.json() as Promise<T>;
  },
};
