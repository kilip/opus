# Phase 3 — Frontend Foundation (Next.js)

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Draft  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Phase Goal

Implement the complete Next.js 16 frontend: project initialization, PWA configuration, API client, TanStack Query hooks, SSE streaming hook, routing, auth UI, dashboard shell, and full test coverage (Vitest + Playwright).

---

## Prerequisites

- Phase 1 is complete.
- Phase 2 API is running locally on `http://localhost:8080`.
- Node.js LTS (22.x) is installed.
- pnpm is installed (`npm install -g pnpm`).
- Task (Taskfile runner) is installed.

---

## Context for AI Agent

> Always provide `docs/CONVENTIONS.md`, `docs/PRD.md`, and `docs/ARCHITECTURE.md` alongside this file.

**You are implementing the Next.js 16 frontend for Opus (`dashboard/`).** Use the App Router exclusively — no Pages Router. All components must be TypeScript strict. Use Tailwind CSS v4 for styling. Shadcn/ui for UI components. TanStack Query for server state. Serwist for PWA/Service Worker. No global state manager (no Zustand, no Redux).

---

## P3-T1 — Initialize Next.js 16 Project

### What to Do

Initialize the Next.js 16 project with TypeScript, Tailwind CSS, and App Router inside `dashboard/`.

### Commands to Run

```bash
cd dashboard/
pnpm create next-app@latest . \
  --typescript \
  --tailwind \
  --app \
  --src-dir=false \
  --import-alias="@/*" \
  --no-git
```

### `tsconfig.json` Strict Mode

Ensure `tsconfig.json` has:

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitReturns": true
  }
}
```

### `.eslintrc.json`

```json
{
  "extends": ["next/core-web-vitals", "next/typescript"],
  "rules": {
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-unused-vars": "error"
  }
}
```

### Constraints

- Use `pnpm` as the package manager — never `npm` or `yarn`.
- App Router only — no `pages/` directory.
- No `src/` directory — files at root level of `dashboard/`.
- Import alias must be `@/*`.

### Acceptance Criteria

- [x] Directory `dashboard/app/` exists with `layout.tsx` and `page.tsx`.
- [x] File `dashboard/tsconfig.json` exists with `strict: true`.
- [x] File `dashboard/tailwind.config.ts` exists.
- [x] File `dashboard/.eslintrc.json` exists with `no-explicit-any: error`.
- [x] File `dashboard/package.json` exists with `next`, `react`, `typescript` as dependencies.
- [x] Running `pnpm dev` starts the development server without errors.

---

## P3-T2 — Install and Configure Tailwind CSS v4 and Shadcn/ui

### What to Do

Upgrade to Tailwind CSS v4 and initialize Shadcn/ui with the default theme.

### Commands to Run

```bash
cd dashboard/
pnpm add tailwindcss@latest @tailwindcss/postcss
pnpm dlx shadcn@latest init
```

### Shadcn/ui Init Prompts

When `shadcn init` asks:
- Style: `Default`
- Base color: `Neutral`
- CSS variables: `Yes`

### Install Base Components

```bash
pnpm dlx shadcn@latest add button input label card avatar separator
```

### Constraints

- All Shadcn/ui components go into `dashboard/components/ui/` — do not edit them manually.
- Use CSS variables for theming — never hardcode color values.
- Tailwind v4 uses `@import "tailwindcss"` in CSS, not the v3 `@tailwind` directives.

### Acceptance Criteria

- [x] `tailwindcss@4.x` is listed in `package.json`.
- [x] `dashboard/components/ui/` contains Shadcn/ui components: `button`, `input`, `label`, `card`, `avatar`, `separator`.
- [x] `dashboard/app/globals.css` uses `@import "tailwindcss"` (Tailwind v4 syntax).
- [x] Running `pnpm build` compiles without Tailwind errors.

---

## P3-T3 — Install and Configure TanStack Query

### What to Do

Install TanStack Query v5 and configure a global `QueryClient` provider in the root layout.

### Commands to Run

```bash
cd dashboard/
pnpm add @tanstack/react-query @tanstack/react-query-devtools
```

### Files to Create

`dashboard/components/shared/QueryProvider.tsx`

```tsx
"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useState } from "react";

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60 * 1000,
            retry: 1,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
```

### Update Root Layout

Wrap `children` in `QueryProvider` inside `dashboard/app/layout.tsx`.

### Constraints

- `QueryClient` must be created inside `useState` to prevent shared state between requests.
- `ReactQueryDevtools` renders only in development (it auto-detects `NODE_ENV`).

### Acceptance Criteria

- [x] `@tanstack/react-query` v5.x is in `package.json`.
- [x] File `dashboard/components/shared/QueryProvider.tsx` exists.
- [x] Root `layout.tsx` wraps children with `QueryProvider`.
- [x] `ReactQueryDevtools` is included.
- [x] Running `pnpm build` compiles without errors.

---

## P3-T4 — Install and Configure Serwist (PWA / Service Worker)

### What to Do

Install Serwist and configure it for Next.js 16 App Router. Set up the Service Worker entry point and caching strategies.

### Commands to Run

```bash
cd dashboard/
pnpm add @serwist/next serwist
```

### Update `next.config.ts`

```ts
import withSerwist from "@serwist/next";

const withSerwistConfig = withSerwist({
  swSrc: "sw.ts",
  swDest: "public/sw.js",
});

export default withSerwistConfig({
  // Next.js config here
});
```

### Create `dashboard/sw.ts`

```ts
import { defaultCache } from "@serwist/next/worker";
import { Serwist } from "serwist";

const serwist = new Serwist({
  precacheEntries: self.__SW_MANIFEST,
  skipWaiting: true,
  clientsClaim: true,
  navigationPreload: true,
  runtimeCaching: [
    {
      matcher: /^\/api\/.*/,
      handler: "NetworkOnly",
    },
    {
      matcher: /\.(?:js|css|woff2)$/,
      handler: "StaleWhileRevalidate",
    },
    {
      matcher: /^\/offline$/,
      handler: "CacheFirst",
    },
    ...defaultCache,
  ],
  fallbacks: {
    document: "/offline",
  },
});

serwist.addEventListeners();
```

### Constraints

- API routes (`/api/*`) must use `NetworkOnly` strategy — never cache API responses.
- The `/offline` page must use `CacheFirst` strategy.
- The Service Worker source is `sw.ts` at the root of `dashboard/`.

### Acceptance Criteria

- [x] `@serwist/next` and `serwist` are in `package.json`.
- [x] File `dashboard/sw.ts` exists with correct caching strategies.
- [x] `dashboard/next.config.ts` wraps with `withSerwist`.
- [x] `NetworkOnly` is applied to `/api/*` routes.
- [x] `CacheFirst` is applied to `/offline` route.
- [x] Running `pnpm build` generates `public/sw.js`.

---

## P3-T5 — Implement API Base Client and Response Types

### What to Do

Implement the base API client and all shared TypeScript types used across query hooks.

### Files to Create

- `dashboard/lib/api/client.ts`
- `dashboard/lib/api/types.ts`

### `types.ts`

```ts
export interface ApiResponse<T> {
  success: boolean;
  data: T | null;
  error: ApiError | null;
}

export interface ApiError {
  code: string;
  message: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl: string;
  provider: string;
  createdAt: string;
  updatedAt: string;
}

export interface AuthTokens {
  accessToken: string;
}

export interface StreamEvent {
  type: string;
  data: string;
  timestamp: string;
}
```

### `client.ts`

```ts
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const response = await fetch(`${API_BASE_URL}/api/v1${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    credentials: "include",
    ...options,
  });

  if (!response.ok) {
    const error = await response.json();
    return error as ApiResponse<T>;
  }

  return response.json();
}

export const apiClient = {
  get: <T>(path: string, options?: RequestInit) =>
    request<T>(path, { method: "GET", ...options }),
  post: <T>(path: string, body?: unknown, options?: RequestInit) =>
    request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      ...options,
    }),
};
```

### Constraints

- `credentials: "include"` is mandatory — needed for HttpOnly cookie (refresh token).
- Base URL comes from `NEXT_PUBLIC_API_URL` env variable with `localhost:8080` fallback.
- No `any` types.

### Acceptance Criteria

- [x] File `dashboard/lib/api/types.ts` exists with all types.
- [x] File `dashboard/lib/api/client.ts` exists.
- [x] `apiClient.get` and `apiClient.post` are exported.
- [x] `credentials: "include"` is set on all requests.
- [x] No `any` types used.
- [x] Code compiles with TypeScript strict mode.

---

## P3-T6 — Implement Auth Query Hooks

### What to Do

Implement TanStack Query hooks for authentication actions.

### File to Create

`dashboard/lib/api/auth.ts`

### Hooks to Implement

| Hook | Type | Description |
|------|------|-------------|
| `useRefreshToken` | `useMutation` | Calls `POST /auth/refresh`, returns new access token |
| `useLogout` | `useMutation` | Calls `POST /auth/logout`, invalidates all queries |

### Token Storage

The access token is stored in React state (not localStorage — see Conventions Section 4). Implement a simple React context to hold the access token:

**`dashboard/lib/api/AuthContext.tsx`**

```tsx
"use client";
import { createContext, useContext, useState } from "react";

interface AuthContextValue {
  accessToken: string | null;
  setAccessToken: (token: string | null) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [accessToken, setAccessToken] = useState<string | null>(null);
  return (
    <AuthContext.Provider value={{ accessToken, setAccessToken }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuthContext() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuthContext must be used within AuthProvider");
  return ctx;
}
```

Update root layout to wrap with `AuthProvider`.

### Constraints

- Access token is stored in React state only — never `localStorage` or `sessionStorage`.
- `useLogout` must call `queryClient.invalidateQueries()` on success to clear all cached data.

### Acceptance Criteria

- [x] File `dashboard/lib/api/auth.ts` exists with `useRefreshToken` and `useLogout`.
- [x] File `dashboard/lib/api/AuthContext.tsx` exists.
- [x] Root layout wraps with `AuthProvider`.
- [x] `useLogout` invalidates all queries on success.
- [x] Access token is never stored in `localStorage`.
- [x] Code compiles with TypeScript strict mode.

---

## P3-T7 — Implement User Query Hooks

### What to Do

Implement TanStack Query hook to fetch the current authenticated user.

### File to Create

`dashboard/lib/api/user.ts`

### Hook to Implement

```ts
export function useCurrentUser(accessToken: string | null) {
  return useQuery({
    queryKey: ["user", "me"],
    queryFn: () =>
      apiClient.get<User>("/user/me", {
        headers: { Authorization: `Bearer ${accessToken}` },
      }),
    enabled: !!accessToken,
  });
}
```

### Acceptance Criteria

- [x] File `dashboard/lib/api/user.ts` exists with `useCurrentUser` hook.
- [x] Query is disabled when `accessToken` is null.
- [x] Query sends `Authorization: Bearer <token>` header.
- [x] Code compiles with TypeScript strict mode.

---

## P3-T8 — Implement `useStream` Hook

### What to Do

Implement the SSE streaming hook using the browser-native `EventSource` API.

### File to Create

`dashboard/lib/api/useStream.ts`

### Implementation

```ts
"use client";
import { useEffect, useState, useCallback } from "react";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function useStream(accessToken: string | null) {
  const [output, setOutput] = useState<string>("");
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearOutput = useCallback(() => setOutput(""), []);

  useEffect(() => {
    if (!accessToken) return;

    // EventSource does not support custom headers
    // Pass token as query param (acceptable for SSE — token is short-lived)
    const url = `${API_BASE_URL}/api/v1/stream?token=${encodeURIComponent(accessToken)}`;
    const es = new EventSource(url, { withCredentials: true });

    es.onopen = () => setIsConnected(true);
    es.onmessage = (e) => setOutput((prev) => prev + e.data);
    es.onerror = () => {
      setError("Stream connection lost");
      setIsConnected(false);
      es.close();
    };

    return () => {
      es.close();
      setIsConnected(false);
    };
  }, [accessToken]);

  return { output, isConnected, error, clearOutput };
}
```

### Constraints

- `EventSource` does not support custom headers — pass the access token as a query parameter.
- Close `EventSource` on component unmount (cleanup function in `useEffect`).
- Set `isConnected: false` on error and close the connection.

### Acceptance Criteria

- [x] File `dashboard/lib/api/useStream.ts` exists.
- [x] Hook accepts `accessToken: string | null`.
- [x] Hook returns `{ output, isConnected, error, clearOutput }`.
- [x] `EventSource` is closed on unmount.
- [x] Hook does nothing when `accessToken` is null.
- [x] Code compiles with TypeScript strict mode.

---

## P3-T9 — Implement Root Layout and Global Styles

### What to Do

Implement `dashboard/app/layout.tsx` (root layout) and `dashboard/app/globals.css`.

### `app/layout.tsx`

```tsx
import type { Metadata } from "next";
import "./globals.css";
import { QueryProvider } from "@/components/shared/QueryProvider";
import { AuthProvider } from "@/lib/api/AuthContext";

export const metadata: Metadata = {
  title: "Opus",
  description: "Your 24/7 autonomous AI assistant",
  manifest: "/manifest.webmanifest",
  themeColor: "#000000",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <AuthProvider>
          <QueryProvider>{children}</QueryProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
```

### `app/globals.css`

Use Tailwind v4 syntax:

```css
@import "tailwindcss";
@import "tw-animate-css";

@layer base {
  :root {
    /* Shadcn/ui CSS variables go here — generated by shadcn init */
  }
}
```

### Acceptance Criteria

- [x] File `dashboard/app/layout.tsx` exists.
- [x] Root layout wraps with `AuthProvider` and `QueryProvider`.
- [x] `metadata.manifest` points to `/manifest.webmanifest`.
- [x] `metadata.themeColor` is set.
- [x] File `dashboard/app/globals.css` uses `@import "tailwindcss"` (v4 syntax).
- [x] Code compiles without errors.

---

## P3-T10 — Implement Auth Layout and Login Page

### What to Do

Implement the auth route group layout and the login page.

### Files to Create

- `dashboard/app/(auth)/layout.tsx`
- `dashboard/app/(auth)/login/page.tsx`

### `(auth)/layout.tsx`

Centered layout for auth pages. No navigation. Redirects to dashboard if user is already authenticated.

### `(auth)/login/page.tsx`

Login page with:
- Opus logo/wordmark
- "Sign in with Google" button → redirects to `GET /auth/google`
- "Sign in with GitHub" button → redirects to `GET /auth/github`
- Development-only: "Sign in with Email" form (renders only when `NEXT_PUBLIC_DEV_MODE=true`)

### Component Rules

- Buttons use Shadcn/ui `Button` component.
- No inline styles — Tailwind only.
- The page is a Server Component (no `"use client"` directive) unless interactivity requires it.

### Acceptance Criteria

- [x] File `dashboard/app/(auth)/layout.tsx` exists.
- [x] File `dashboard/app/(auth)/login/page.tsx` exists.
- [x] Login page has "Sign in with Google" button linking to `/auth/google`.
- [x] Login page has "Sign in with GitHub" button linking to `/auth/github`.
- [x] Dev-only email form is conditionally rendered.
- [x] Uses Shadcn/ui `Button` component.
- [x] No inline styles.
- [x] Code compiles without errors.

---

## P3-T11 — Implement `AuthGuard` Component

### What to Do

Implement a client-side route protection component that redirects unauthenticated users to the login page.

### File to Create

`dashboard/components/shared/AuthGuard.tsx`

### Implementation

```tsx
"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthContext } from "@/lib/api/AuthContext";
import { useRefreshToken } from "@/lib/api/auth";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { accessToken, setAccessToken } = useAuthContext();
  const { mutate: refresh, isPending } = useRefreshToken();

  useEffect(() => {
    if (!accessToken) {
      // Attempt silent refresh via HttpOnly cookie
      refresh(undefined, {
        onSuccess: (data) => {
          if (data.success && data.data) {
            setAccessToken(data.data.accessToken);
          } else {
            router.push("/login");
          }
        },
        onError: () => router.push("/login"),
      });
    }
  }, [accessToken]);

  if (isPending || !accessToken) {
    return <div className="flex h-screen items-center justify-center">Loading...</div>;
  }

  return <>{children}</>;
}
```

### Constraints

- `AuthGuard` must attempt a silent token refresh via the HttpOnly cookie before redirecting.
- Show a loading state while the refresh is pending.
- Only redirect to `/login` if the refresh fails.

### Acceptance Criteria

- [x] File `dashboard/components/shared/AuthGuard.tsx` exists.
- [x] Component is a Client Component (`"use client"`).
- [x] Attempts silent refresh on mount if no access token.
- [x] Redirects to `/login` on refresh failure.
- [x] Shows loading state while refresh is pending.
- [x] Code compiles without errors.

---

## P3-T12 — Implement Dashboard Layout and Main Dashboard Page

### What to Do

Implement the protected dashboard route group.

### Files to Create

- `dashboard/app/(dashboard)/layout.tsx`
- `dashboard/app/(dashboard)/page.tsx`

### `(dashboard)/layout.tsx`

Wrap all dashboard pages with `AuthGuard`. Include a minimal navigation bar with the Opus logo and a logout button.

### `(dashboard)/page.tsx`

Main dashboard page:
- Display current user name and avatar (from `useCurrentUser` hook).
- Include a `StreamOutput` component.
- Include a "Connect" button that starts the SSE stream.

### Acceptance Criteria

- [x] File `dashboard/app/(dashboard)/layout.tsx` exists with `AuthGuard`.
- [x] File `dashboard/app/(dashboard)/page.tsx` exists.
- [x] Dashboard page displays user name and avatar.
- [x] Dashboard page includes `StreamOutput` component.
- [x] Dashboard layout wraps with `AuthGuard`.
- [x] Code compiles without errors.

---

## P3-T13 — Implement `StreamOutput` Component

### What to Do

Implement the component that displays SSE stream output.

### File to Create

`dashboard/components/shared/StreamOutput.tsx`

### Props Interface

```tsx
interface StreamOutputProps {
  output: string;
  isConnected: boolean;
  error: string | null;
}
```

### Implementation Requirements

- Display output in a monospace font.
- Show a green dot indicator when `isConnected: true`, red when `false`.
- Show error message when `error` is not null.
- Auto-scroll to the bottom when new content arrives (`useEffect` + `ref`).
- Empty state: "No output yet. Click Connect to start streaming."

### Acceptance Criteria

- [x] File `dashboard/components/shared/StreamOutput.tsx` exists.
- [x] Component accepts `{ output, isConnected, error }` props.
- [x] Displays output in monospace font.
- [x] Shows connection status indicator.
- [x] Auto-scrolls to bottom on new content.
- [x] Shows empty state message when output is empty.
- [x] No inline styles — Tailwind only.
- [x] Code compiles without errors.

---

## P3-T14 — Implement Offline Fallback Page

### What to Do

Implement the PWA offline fallback page shown when the server is unreachable.

### File to Create

`dashboard/app/offline/page.tsx`

### Requirements

- Display a clear "You are offline" message.
- Display the Opus wordmark/logo.
- Include a "Try again" button that reloads the page (`window.location.reload()`).
- This page is a Client Component (needs `window.location`).
- Styled with Tailwind CSS — centered, full-screen.

### Acceptance Criteria

- [x] File `dashboard/app/offline/page.tsx` exists.
- [x] Page is a Client Component (`"use client"`).
- [x] Displays "You are offline" message.
- [x] Includes a "Try again" button that calls `window.location.reload()`.
- [x] Page is centered and full-screen.
- [x] No inline styles.
- [x] Code compiles without errors.

---

## P3-T15 — Configure PWA Manifest

### What to Do

Create the PWA web app manifest file.

### File to Create

`dashboard/public/manifest.webmanifest`

### Content

```json
{
  "name": "Opus",
  "short_name": "Opus",
  "description": "Your 24/7 autonomous AI assistant",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#000000",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ]
}
```

### Icon Placeholders

Create two placeholder PNG files (1x1 pixel black PNG) at:
- `dashboard/public/icons/icon-192.png`
- `dashboard/public/icons/icon-512.png`

These will be replaced with real icons before production release.

### Acceptance Criteria

- [x] File `dashboard/public/manifest.webmanifest` exists.
- [x] Contains `name`, `short_name`, `description`, `start_url`, `display`.
- [x] Contains `background_color` and `theme_color`.
- [x] Contains two icon entries: 192x192 and 512x512.
- [x] Both icon files exist in `dashboard/public/icons/`.
- [x] Root layout `metadata.manifest` points to `/manifest.webmanifest`.

---

## P3-T16 — Configure `dashboard/Taskfile.yml`

### What to Do

Create the final `dashboard/Taskfile.yml` with all tasks.

### File to Create

`dashboard/Taskfile.yml`

### Content

```yaml
version: "3"

tasks:
  setup:
    desc: Install Node.js dependencies
    cmds:
      - pnpm install
      - pnpm dlx playwright install --with-deps

  dev:
    desc: Start Next.js in development mode
    cmds:
      - pnpm dev

  build:
    desc: Build production bundle
    cmds:
      - pnpm build

  test:
    desc: Run unit and component tests (Vitest)
    cmds:
      - pnpm test

  test:coverage:
    desc: Run unit tests with coverage report
    cmds:
      - pnpm test:coverage

  test:e2e:
    desc: Run end-to-end tests (Playwright)
    cmds:
      - pnpm test:e2e

  test:e2e:ui:
    desc: Run end-to-end tests with Playwright UI
    cmds:
      - pnpm test:e2e:ui

  lint:
    desc: Run ESLint and TypeScript type checks
    cmds:
      - pnpm lint
      - pnpm tsc --noEmit
```

### Acceptance Criteria

- [x] File `dashboard/Taskfile.yml` exists.
- [x] `task setup` installs dependencies and Playwright browsers.
- [x] All 7 tasks are defined.
- [x] Running `task --list` from `dashboard/` lists all tasks without errors.

---

## P3-T17 — Write Vitest Unit Tests (Hooks and Utilities)

### What to Do

Install Vitest and write unit tests for custom hooks and utility functions.

### Commands to Run

```bash
cd dashboard/
pnpm add -D vitest @vitejs/plugin-react @testing-library/react @testing-library/user-event jsdom
```

### `vitest.config.ts`

```ts
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
  },
  resolve: {
    alias: {
      "@": new URL(".", import.meta.url).pathname,
    },
  },
});
```

### `vitest.setup.ts`

```ts
import "@testing-library/jest-dom";
```

### Test Cases to Implement

**`lib/utils/cn.test.ts`** — Test the Tailwind class merge utility.

| Test | Description |
|------|-------------|
| `merges classes correctly` | `cn("a", "b")` returns `"a b"` |
| `handles conditional classes` | `cn("a", false && "b")` returns `"a"` |
| `handles Tailwind conflicts` | `cn("p-2", "p-4")` returns `"p-4"` |

**`lib/api/useStream.test.ts`** — Test the SSE hook.

| Test | Description |
|------|-------------|
| `does not connect when accessToken is null` | No EventSource created |
| `returns empty output initially` | `output` is `""` on mount |
| `clearOutput resets output` | `clearOutput()` sets output to `""` |

### `package.json` Scripts

Add to `dashboard/package.json`:

```json
{
  "scripts": {
    "test": "vitest",
    "test:coverage": "vitest --coverage"
  }
}
```

### Acceptance Criteria

- [x] `vitest`, `@testing-library/react` are in `devDependencies`.
- [x] File `dashboard/vitest.config.ts` exists.
- [x] File `dashboard/vitest.setup.ts` exists.
- [x] `lib/utils/cn.test.ts` exists with 3 test cases.
- [x] `lib/api/useStream.test.ts` exists with 3 test cases.
- [x] Running `task test` passes all tests.

---

## P3-T18 — Write Playwright E2E Tests

### What to Do

Install Playwright and write E2E tests for the four critical flows.

### Commands to Run

```bash
cd dashboard/
pnpm add -D @playwright/test
```

### `playwright.config.ts`

```ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "html",
  use: {
    baseURL: "http://localhost:3000",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: "pnpm dev",
    url: "http://localhost:3000",
    reuseExistingServer: !process.env.CI,
  },
});
```

### E2E Test Files

**`e2e/auth.spec.ts`**

| Test | Description |
|------|-------------|
| `redirects unauthenticated users to /login` | Visit `/` → redirected to `/login` |
| `login page shows Google and GitHub buttons` | Both OAuth buttons visible |
| `Google login button redirects to /auth/google` | Click → navigates to `/auth/google` |

**`e2e/dashboard.spec.ts`**

| Test | Description |
|------|-------------|
| `authenticated users see dashboard` | Mock auth → visit `/` → dashboard visible |
| `logout clears session and redirects to login` | Click logout → redirected to `/login` |

**`e2e/stream.spec.ts`**

| Test | Description |
|------|-------------|
| `StreamOutput shows empty state` | No output → empty state message visible |
| `Connect button is visible` | "Connect" button renders on dashboard |

**`e2e/pwa.spec.ts`**

| Test | Description |
|------|-------------|
| `manifest.webmanifest is accessible` | `GET /manifest.webmanifest` returns 200 |
| `offline page is accessible` | `GET /offline` returns 200 |
| `service worker is registered` | `sw.js` is served |

### `package.json` Scripts

```json
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui"
  }
}
```

### Acceptance Criteria

- [x] `@playwright/test` is in `devDependencies`.
- [x] File `dashboard/playwright.config.ts` exists.
- [x] File `e2e/auth.spec.ts` exists with 3 test cases.
- [x] File `e2e/dashboard.spec.ts` exists with 2 test cases.
- [x] File `e2e/stream.spec.ts` exists with 2 test cases.
- [x] File `e2e/pwa.spec.ts` exists with 3 test cases.
- [x] Running `task test:e2e` executes all tests.

---

## Phase 3 Completion Checklist

Before proceeding to Phase 4, verify:

- [x] All 18 tasks (P3-T1 through P3-T18) are marked complete.
- [x] Running `pnpm build` produces a production build without errors.
- [x] Running `task test` passes all Vitest unit tests.
- [x] Running `task test:e2e` passes all Playwright E2E tests.
- [x] Running `task lint` passes ESLint and TypeScript checks.
- [x] `/manifest.webmanifest` is accessible at runtime.
- [x] `/offline` page renders correctly.
- [x] Service Worker (`/sw.js`) is served.
- [x] Login page shows both OAuth buttons.
- [x] Unauthenticated visits to `/` redirect to `/login`.
