/**
 * Standard API response envelope as defined in ADR-004.
 */
export interface ApiEnvelope<T> {
  data: T | null;
  error: ProblemDetail | null;
  meta: PaginationMeta | null;
}

/**
 * RFC 7807 Problem Details for HTTP APIs.
 */
export interface ProblemDetail {
  status: number;
  code: string;
  title: string;
  detail?: string;
  instance?: string;
  request_id?: string;
  validation?: Record<string, string[]>;
}

/**
 * Pagination metadata for collection endpoints.
 */
export interface PaginationMeta {
  next_cursor: string | null;
  total: number | null;
  request_id: string;
}
