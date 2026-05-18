/**
 * Shared API client for all network requests.
 * Centralizes base URL, envelope parsing, and error handling.
 * Consistent with ADR-004 and ADR-011.
 */

import type { ApiEnvelope } from '@/shared/types/api';

const BASE_URL = (import.meta.env.VITE_API_URL as string) ?? '';

/**
 * Custom error class for API errors.
 */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    public title: string,
    public detail?: string,
  ) {
    super(detail || title);
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${BASE_URL}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  const envelope: ApiEnvelope<T> = await response.json();

  if (!response.ok || envelope.error) {
    const error = envelope.error || {
      status: response.status,
      code: 'unknown',
      title: response.statusText,
    };
    throw new ApiError(error.status, error.code, error.title, error.detail);
  }

  return envelope.data as T;
}

export const api = {
  get: <T>(path: string, options?: RequestInit) =>
    request<T>(path, { ...options, method: 'GET' }),
  post: <T>(path: string, body?: unknown, options?: RequestInit) =>
    request<T>(path, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    }),
  put: <T>(path: string, body?: unknown, options?: RequestInit) =>
    request<T>(path, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(path: string, options?: RequestInit) =>
    request<T>(path, { ...options, method: 'DELETE' }),
};
