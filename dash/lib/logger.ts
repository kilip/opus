/**
 * dash/lib/logger.ts
 * Unified logger interface that automatically detects the runtime environment.
 */

import { clientLogger } from "./logger.client";
import { serverLogger } from "./logger.server";
import type { Logger } from "./logger.types";

const isServer = typeof window === "undefined";
const isEdge = process.env.NEXT_RUNTIME === "edge";

/**
 * Isomorphic logger instance.
 */
export const logger: Logger = isServer || isEdge ? serverLogger : clientLogger;

export type { Logger };
