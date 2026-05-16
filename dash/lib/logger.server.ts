/**
 * dash/lib/logger.server.ts
 * Pino implementation for Node.js/Edge environments.
 */
import { join } from "node:path";
import pino from "pino";
import type { Logger } from "./logger.types";

const logDir = process.env.OPUS_LOG_DIR;
const logLevel = process.env.OPUS_LOG_LEVEL || "info";

const targets: pino.TransportTargetOptions[] = [
  {
    target: "pino/file",
    options: { destination: 1 }, // stdout
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
  transport:
    process.env.NODE_ENV === "development"
      ? {
          target: "pino-pretty",
          options: { colorize: true },
        }
      : targets.length > 1
        ? { targets }
        : { target: "pino/file", options: { destination: 1 } },
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
