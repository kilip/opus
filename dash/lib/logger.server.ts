/**
 * dash/lib/logger.server.ts
 * Pino implementation for Node.js/Edge environments.
 */

import { existsSync, mkdirSync } from "node:fs";
import { homedir } from "node:os";
import { join, resolve } from "node:path";
import pino from "pino";
import type { Logger } from "./logger.types";

const getOpusDir = () => {
  if (process.env.OPUS_HOME) return process.env.OPUS_HOME;
  const env = process.env.OPUS_SERVER_ENV || "production";

  if (env === "development") {
    // Check for .opus in current dir or parent (root)
    const localOpus = resolve(process.cwd(), ".opus");
    if (existsSync(localOpus)) return localOpus;

    const parentOpus = resolve(process.cwd(), "..", ".opus");
    if (existsSync(parentOpus)) return parentOpus;

    return localOpus; // fallback
  }

  return join(homedir(), ".opus");
};

const opusDir = getOpusDir();
const logDir = process.env.OPUS_LOG_DIR || join(opusDir, "logs");
const logLevel = process.env.OPUS_LOG_LEVEL || "info";

// Ensure log directory exists
if (logDir && !existsSync(logDir)) {
  try {
    mkdirSync(logDir, { recursive: true });
  } catch (_err) {
    // Fallback handled by pino transport
  }
}

const targets: pino.TransportTargetOptions[] = [
  {
    target:
      process.env.NODE_ENV === "development" ? "pino-pretty" : "pino/file",
    options:
      process.env.NODE_ENV === "development"
        ? { colorize: true }
        : { destination: 1 },
    level: logLevel as pino.Level,
  },
];

if (logDir) {
  targets.push({
    target: "pino-roll",
    options: {
      file: join(logDir, "dash"),
      extension: ".log",
      dateFormat: "yyyy-MM-dd",
      size: "50m",
      limit: { count: 30 },
    },
    level: logLevel as pino.Level,
  });
}

const pinoInstance = pino({
  level: logLevel,
  redact: {
    paths: [
      "password",
      "token",
      "secret",
      "authorization",
      "cookie",
      "accessToken",
      "refreshToken",
      "*.password",
      "*.token",
      "*.secret",
      "*.authorization",
      "*.cookie",
    ],
    censor: "[REDACTED]",
  },
  transport: { targets },
});

export const serverLogger: Logger = {
  info: (msg, data) => pinoInstance.info(data || {}, msg),
  warn: (msg, data) => pinoInstance.warn(data || {}, msg),
  error: (msg, error, data) => {
    const payload: Record<string, unknown> = { ...data };
    if (error instanceof Error) {
      payload.err = {
        message: error.message,
        stack: error.stack,
        name: error.name,
      };
    } else if (error) {
      payload.err = error;
    }
    pinoInstance.error(payload, msg);
  },
  debug: (msg, data) => pinoInstance.debug(data || {}, msg),
};
