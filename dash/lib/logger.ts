/**
 * dash/lib/logger.ts
 * Unified logger interface that automatically detects the runtime environment.
 */

import { clientLogger } from "./logger.client";
import type { Logger } from "./logger.types";

/**
 * Isomorphic logger instance.
 * Uses dynamic require for serverLogger to avoid bundling node:fs/node:os into the client.
 */
export const logger: Logger =
  typeof window === "undefined"
    ? // @ts-expect-error - server-side require
      require("./logger.server").serverLogger
    : clientLogger;

export type { Logger };
