/**
 * dash/lib/logger.client.ts
 * Browser-side logger implementation.
 */
import type { Logger } from "./logger.types";

const isDev =
  process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";

export const clientLogger: Logger = {
  info: (msg, data) => {
    if (isDev) console.info(`[INFO] ${msg}`, data || "");
  },
  warn: (msg, data) => {
    console.warn(`[WARN] ${msg}`, data || "");
  },
  error: (msg, error, data) => {
    console.error(`[ERROR] ${msg}`, { error, ...(data || {}) });
  },
  debug: (msg, data) => {
    if (isDev) console.debug(`[DEBUG] ${msg}`, data || "");
  },
};
