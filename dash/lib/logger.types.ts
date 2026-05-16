/**
 * dash/lib/logger.types.ts
 * Shared types for the isomorphic logger.
 */

export interface Logger {
  info(msg: string, data?: Record<string, unknown>): void;
  warn(msg: string, data?: Record<string, unknown>): void;
  error(
    msg: string,
    error?: Error | unknown,
    data?: Record<string, unknown>,
  ): void;
  debug(msg: string, data?: Record<string, unknown>): void;
}
