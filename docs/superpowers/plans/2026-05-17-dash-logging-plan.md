# Dashboard Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a robust, isomorphic logging system for the Opus dashboard using Pino for server/edge and structured console for the browser, ensuring privacy and self-hosting compatibility.

**Architecture:**
- **Unified Interface**: `dash/lib/logger.ts` provides a common `Logger` interface.
- **Server/Edge Handler**: `dash/lib/logger.server.ts` uses Pino with `pino-roll` for file logging and daily rotation.
- **Browser Handler**: `dash/lib/logger.client.ts` routes logs to `console.*` with production filtering (errors/warnings only).
- **Global Observability**: Root `layout.tsx` is wrapped in an `ErrorBoundary` and a client-side `Observability` component to capture unhandled errors and rejections.

**Tech Stack:**
- Pino v9, pino-roll, pino-pretty
- Next.js 16 (App Router)
- TypeScript
- Vitest (for TDD)

---

### Task 1: Environment & Dependencies

**Files:**
- Modify: `dash/package.json`
- Modify: `dash/.env.example`

- [x] **Step 1: Install logging dependencies**

Run: `pnpm add pino pino-roll`
Run: `pnpm add -D pino-pretty`

- [x] **Step 2: Update .env.example**

Add the following to `dash/.env.example`:
```dotenv
# Logging
OPUS_LOG_LEVEL=info
OPUS_LOG_DIR=
```

- [x] **Step 3: Commit**

```bash
git add dash/package.json dash/pnpm-lock.yaml dash/.env.example
git commit -m "chore(dash): add logging dependencies and env variables"
```

---

### Task 2: Isomorphic Logger Interface (TDD)

**Files:**
- Create: `dash/lib/logger.ts`
- Create: `dash/lib/logger.test.ts`
- Test: `dash/lib/logger.test.ts`

- [x] **Step 1: Write the failing test for interface and routing**

```typescript
// dash/lib/logger.test.ts
import { describe, expect, it, vi } from "vitest";

describe("isomorphic logger routing", () => {
  it("should export a logger instance", async () => {
    const { logger } = await import("./logger");
    expect(logger).toBeDefined();
    expect(logger.info).toBeTypeOf("function");
  });
});
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test dash/lib/logger.test.ts`
Expected: FAIL (file doesn't exist)

- [x] **Step 3: Implement the shared interface and isomorphic routing**

```typescript
/**
 * dash/lib/logger.ts
 * Unified logger interface that automatically detects the runtime environment.
 */

export interface Logger {
  info(msg: string, data?: Record<string, unknown>): void;
  warn(msg: string, data?: Record<string, unknown>): void;
  error(msg: string, error?: Error | unknown, data?: Record<string, unknown>): void;
  debug(msg: string, data?: Record<string, unknown>): void;
}

const isServer = typeof window === "undefined";
const isEdge = process.env.NEXT_RUNTIME === "edge";

/**
 * Isomorphic logger instance.
 * Automatically switches between Pino (server/edge) and Console (client).
 */
export const logger: Logger = (isServer || isEdge)
  ? require("./logger.server").serverLogger 
  : require("./logger.client").clientLogger;
```

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test dash/lib/logger.test.ts`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add dash/lib/logger.ts dash/lib/logger.test.ts
git commit -m "feat(dash): add isomorphic logger interface with tests"
```

---

### Task 3: Server-Side Pino Implementation (TDD)

**Files:**
- Create: `dash/lib/logger.server.ts`
- Create: `dash/lib/logger.server.test.ts`
- Test: `dash/lib/logger.server.test.ts`

- [x] **Step 1: Write failing test for server logging**

```typescript
// dash/lib/logger.server.test.ts
import { describe, expect, it, vi } from "vitest";
import { serverLogger } from "./logger.server";

describe("serverLogger", () => {
  it("should have all logger methods defined", () => {
    expect(serverLogger.info).toBeDefined();
    expect(serverLogger.error).toBeDefined();
  });
});
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test dash/lib/logger.server.test.ts`
Expected: FAIL

- [x] **Step 3: Implement the Pino server logger**

```typescript
/**
 * dash/lib/logger.server.ts
 * Pino implementation for Node.js/Edge environments.
 */
import pino from "pino";
import { join } from "path";
import { Logger } from "./logger";

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
          options: { colorize: true }
        }
      : targets.length > 1 ? { targets } : { target: "pino/file", options: { destination: 1 } },
});

export const serverLogger: Logger = {
  info: (msg, data) => pinoInstance.info(data || {}, msg),
  warn: (msg, data) => pinoInstance.warn(data || {}, msg),
  error: (msg, error, data) => {
    const payload = { ...data };
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
```

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test dash/lib/logger.server.test.ts`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add dash/lib/logger.server.ts dash/lib/logger.server.test.ts
git commit -m "feat(dash): implement server-side pino logger with tests"
```

---

### Task 4: Client-Side Console Implementation (TDD)

**Files:**
- Create: `dash/lib/logger.client.ts`
- Create: `dash/lib/logger.client.test.ts`
- Test: `dash/lib/logger.client.test.ts`

- [x] **Step 1: Write failing test for browser logging**

```typescript
// dash/lib/logger.client.test.ts
import { describe, expect, it, vi, beforeEach } from "vitest";
import { clientLogger } from "./logger.client";

describe("clientLogger", () => {
  beforeEach(() => {
    vi.spyOn(console, "info").mockImplementation(() => {});
  });

  it("should log info to console", () => {
    clientLogger.info("test message");
    expect(console.info).toHaveBeenCalled();
  });
});
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test dash/lib/logger.client.test.ts`
Expected: FAIL

- [x] **Step 3: Implement the browser console logger**

```typescript
/**
 * dash/lib/logger.client.ts
 * Browser-side logger implementation.
 */
import { Logger } from "./logger";

const isDev = process.env.NODE_ENV === "development";

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
```

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test dash/lib/logger.client.test.ts`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add dash/lib/logger.client.ts dash/lib/logger.client.test.ts
git commit -m "feat(dash): implement client-side console logger with tests"
```

---

### Task 5: Global Observability Components (TDD)

**Files:**
- Create: `dash/components/shared/ErrorBoundary.tsx`
- [x] **Step 1: Write failing test for ErrorBoundary**

```tsx
// dash/components/shared/ErrorBoundary.test.tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ErrorBoundary } from "./ErrorBoundary";

const ThrowError = () => {
  throw new Error("Test Error");
};

describe("ErrorBoundary", () => {
  it("should catch errors and show fallback UI", () => {
    // Suppress console.error for this test
    vi.spyOn(console, "error").mockImplementation(() => {});
    
    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>
    );
    expect(screen.getByText(/Something went wrong/i)).toBeDefined();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm test dash/components/shared/ErrorBoundary.test.tsx`
Expected: FAIL

- [ ] **Step 3: Implement ErrorBoundary component**

```tsx
/**
 * dash/components/shared/ErrorBoundary.tsx
 * Captures React render errors and logs them.
 */
"use client";

import React, { Component, ErrorInfo, ReactNode } from "react";
import { logger } from "@/lib/logger";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
  };

  public static getDerivedStateFromError(_: Error): State {
    return { hasError: true };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    logger.error("Unhandled React Error", error, {
      componentStack: errorInfo.componentStack,
    });
  }

  public render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div className="flex min-h-screen items-center justify-center bg-background p-4 text-foreground">
          <div className="text-center">
            <h2 className="mb-2 text-2xl font-bold">Something went wrong</h2>
            <p className="text-muted-foreground">Please try refreshing the page.</p>
            <button 
              onClick={() => window.location.reload()}
              className="mt-4 rounded-md bg-primary px-4 py-2 text-primary-foreground hover:bg-primary/90"
            >
              Refresh Page
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm test dash/components/shared/ErrorBoundary.test.tsx`
Expected: PASS

- [x] **Step 5: Implement Observability component**

```tsx
/**
 * dash/components/shared/Observability.tsx
 * Client-side component for global event listeners.
 */
"use client";

import { useEffect } from "react";
import { logger } from "@/lib/logger";

export function Observability({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    const handleRejection = (event: PromiseRejectionEvent) => {
      logger.error("Unhandled Promise Rejection", event.reason);
    };

    window.addEventListener("unhandledrejection", handleRejection);
    return () => window.removeEventListener("unhandledrejection", handleRejection);
  }, []);

  return <>{children}</>;
}
```

- [ ] **Step 6: Update root layout**

```tsx
/**
 * dash/app/layout.tsx
 * Updated with ErrorBoundary and Observability wrapper.
 */
import type { Metadata } from "next";
import { Lora, Poppins } from "next/font/google";
import "./globals.css";
import { QueryProvider } from "@/components/shared/QueryProvider";
import { AuthProvider } from "@/lib/api/AuthContext";
import { ErrorBoundary } from "@/components/shared/ErrorBoundary";
import { Observability } from "@/components/shared/Observability";

const poppins = Poppins({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-poppins",
  weight: ["400", "500", "600", "700"],
});

const lora = Lora({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-lora",
  weight: ["400", "500"],
});

export const metadata: Metadata = {
  title: "Opus",
  description: "Your 24/7 autonomous AI assistant",
  manifest: "/manifest.webmanifest",
  appleWebApp: {
    capable: true,
    statusBarStyle: "default",
    title: "Opus",
  },
  icons: {
    apple: "/icons/apple-touch-icon.png",
  },
};

export const viewport = {
  themeColor: "#FFFCF5",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${poppins.variable} ${lora.variable} font-body antialiased`}
      >
        <ErrorBoundary>
          <Observability>
            <AuthProvider>
              <QueryProvider>{children}</QueryProvider>
            </AuthProvider>
          </Observability>
        </ErrorBoundary>
      </body>
    </html>
  );
}
```

- [ ] **Step 7: Commit**

```bash
git add dash/components/shared/ErrorBoundary.tsx dash/components/shared/ErrorBoundary.test.tsx dash/components/shared/Observability.tsx dash/app/layout.tsx
git commit -m "feat(dash): add global error boundary and observability with tests"
```

---

### Task 6: Security & Linting

**Files:**
- Modify: `dash/.eslintrc.json`

- [x] **Step 1: Enforce no-console rule**

```json
{
  "extends": ["next/core-web-vitals", "next/typescript"],
  "rules": {
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-unused-vars": "error",
    "no-console": "error"
  }
}
```

- [x] **Step 2: Commit**

```bash
git add dash/.eslintrc.json
git commit -m "chore(dash): enforce no-console lint rule"
```

---

### Task 7: Infrastructure & Automation

**Files:**
- Modify: `docker-compose.yml`
- Modify: `dash/Taskfile.yml`

- [x] **Step 1: Update docker-compose**

```yaml
# docker-compose.yml
# ... lines 22-34
  dash:
    build:
      context: ./dash
      dockerfile: Dockerfile
      args:
        - NEXT_PUBLIC_API_URL=http://localhost:8080
    ports:
      - "3000:3000"
    environment:
      - OPUS_LOG_LEVEL=info
      - OPUS_LOG_DIR=/var/log/opus
    volumes:
      - ./logs:/var/log/opus
    depends_on:
      api:
        condition: service_healthy
    restart: always
```

- [x] **Step 2: Add Taskfile commands**

```yaml
# dash/Taskfile.yml
# ... existing tasks

  logs:clear:
    desc: Delete all log files in OPUS_LOG_DIR
    cmds:
      - 'find ${OPUS_LOG_DIR:-./logs} -name "dash-*.log" -delete'

  logs:tail:
    desc: Tail the latest dash log file
    cmds:
      - 'tail -f $(ls -t ${OPUS_LOG_DIR:-./logs}/dash-*.log | head -1)'
```

- [x] **Step 3: Commit**

```bash
git add docker-compose.yml dash/Taskfile.yml
git commit -m "chore: update infrastructure for logging support"
```

---

### Task 8: Verification

- [x] **Step 1: Final verification run**

Run: `pnpm test`
Run: `pnpm lint`
Check: `dash/logs/` content after `task dev`

- [x] **Step 2: Final commit**

```bash
git commit --allow-empty -m "docs: finalized dashboard logging implementation"
```
