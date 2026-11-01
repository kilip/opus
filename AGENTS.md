# AGENTS.md

## Project

`opus`: a web UI for LLM provider CLIs (Claude Code, Antigravity CLI, ...).
The UI drives the real CLIs as child processes; it does not reimplement them.

**MVP / proof of concept scope**
- Web only
- Antigravity CLI (`agy`) only
- Self-hosted, local, single machine

**Later (not in MVP, do not build yet)**
- Claude Code provider
- Telegram channel (send commands)
- Docker packaging
- Multi-user

## Stack

- Runtime / package manager: Bun
- Frontend: React Router 8
- Backend: ElysiaJS
- Auth: better-auth

## Directory structure

Target layout. Create directories only when the code that belongs there exists;
do not scaffold empty modules ahead of time (no `telegram/`, no `claude-code/` in the MVP).

```
opus/
├── AGENTS.md
├── package.json              # Bun workspaces + turbo scripts
├── turbo.json
├── apps/
│   ├── web/                  # @opus/web: React Router 8 UI (layout in "apps/web layout" below)
│   └── api/                  # @opus/api: ElysiaJS backend
│       └── src/
│           ├── index.ts      # composition root: wires modules, starts Elysia
│           ├── config.ts     # env/config (CLI binPath, homeDir, port, data dirs)
│           └── modules/
│               ├── providers/    # AgentProvider interface + one folder per provider
│               │   └── antigravity/   # spawn agy, stream-parser.ts, __fixtures__/
│               ├── runs/         # run lifecycle: start, cancel, queue, timeouts
│               ├── workspaces/   # working directories agents run in
│               └── channels/
│                   └── web/      # HTTP routes + SSE for the browser
├── packages/
│   ├── shared/               # @opus/shared: RunEvent types, API contracts (src/run-events.ts)
│   ├── db/                   # @opus/db: Drizzle ORM, bun:sqlite client, schema, migrations
│   └── auth/                 # @opus/auth: better-auth instance, plugins (OTP, OAuth), env validation
├── cli/                      # gitignored: installed CLI binaries (cli/antigravity/agy)
├── data/                     # gitignored: SQLite (app + better-auth)
└── workspaces/               # gitignored: agent working directories, separate from data/
```

Workspace package names: `@opus/web`, `@opus/api`, `@opus/shared`, `@opus/db`, `@opus/auth`.

### Module dependencies (proposed)

Dependencies point one way. A module may import only from the modules listed.

| Module / Package | May depend on                             |
| ---------------- | ----------------------------------------- |
| `packages/db`    | nothing                                   |
| `packages/auth`  | `@opus/db`, `@opus/shared`                |
| `providers/*`    | `@opus/shared`                            |
| `workspaces`     | nothing                                   |
| `runs`           | `providers`, `workspaces`, `@opus/shared` |
| `channels/*`     | `runs`, `@opus/auth`, `@opus/shared`      |

### Where things go

- Database schema, migrations, and Drizzle connection: `packages/db`.
- Better-auth configuration, plugins (OTP, OAuth), and auth middleware/helpers: `packages/auth`.
- New provider: `modules/providers/<name>/`, implementing `AgentProvider`. Nothing outside
  `providers/` may know that provider's raw output format.
- New event type or API shape used by more than one app: `packages/shared`.
- New inbound channel (e.g. Telegram): `modules/channels/<name>/`, calling `runs` and `@opus/auth`.
- Test fixtures: real captured CLI output only, in the adapter's `__fixtures__/` folder.
- Anything under `cli/`, `data/`, `workspaces/` is runtime state and must stay gitignored.

### apps/web layout (proposed)

React Router 8 in framework mode, SPA (`ssr: false`): the app sits behind login and the
backend is Elysia, so there is no server rendering to maintain.

```
apps/web/
├── package.json
├── react-router.config.ts      # ssr: false
├── vite.config.ts
├── tsconfig.json
├── public/
└── app/
    ├── root.tsx                # document shell, global providers
    ├── routes.ts               # explicit route map: single source of truth for URLs
    ├── routes/                 # thin route modules: compose features, no business logic
    │   ├── login.tsx
    │   ├── app-layout.tsx      # authenticated shell (sidebar + outlet); redirects to /login
    │   ├── home.tsx            # empty state / start a new session
    │   ├── session.tsx         # /sessions/:sessionId
    │   └── settings.tsx        # provider status
    ├── features/               # one folder per feature: components + hooks + state
    │   ├── conversation/       # message list, tool-call cards, composer, run header, useRunStream
    │   ├── sessions/           # session list in the sidebar
    │   └── provider-status/    # CLI binary found? version? logged in?
    ├── components/             # shared presentational components, no feature logic
    ├── lib/
    │   ├── api.ts              # typed Elysia client (Eden Treaty)
    │   └── auth-client.ts      # better-auth client
    └── styles/
```

**Screens (MVP)**

| Route                  | Purpose                                                                                                                              |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `/login`               | better-auth sign-in                                                                                                                  |
| `/`                    | inside `app-layout`: empty state, start a new session                                                                                |
| `/sessions/:sessionId` | conversation view: message list, collapsible tool-call cards, composer at the bottom, run header (cwd, permission mode, token usage) |
| `/settings`            | provider status: binary path, version, login state                                                                                   |

**Rules**

- Routes stay thin. Logic lives in `features/<name>/`, which may import from `components/`
  and `lib/`, never from another feature's internals.
- The UI only knows `RunEvent` from `@opus/shared`. Conversation state is a reducer that
  folds `RunEvent`s; it must never parse provider-specific output.
- `useRunStream` consumes the SSE stream from the server and feeds the reducer.
- Agent output (messages, tool output) is untrusted text. Render tool output as plain text
  or code; sanitize any markdown. Never inject it as raw HTML.
- `apps/web` may `import type` from `@opus/api` for the Eden client types. Never import
  API runtime code.

### UI rendering (proposed)

How agent output is rendered in the browser.

| Output                                | Approach                                                                                                           |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Assistant message (streamed markdown) | Streamdown, used standalone. Do not add the Vercel AI SDK: our data source is `RunEvent`, not the AI SDK protocol. |
| Code blocks                           | Shiki, via Streamdown's code plugin                                                                                |
| Tool calls                            | Own components: collapsible card with tool name, input as code, output as plain text in `<pre>`                    |
| ANSI-colored output                   | Not needed yet. Add a small ANSI-to-HTML converter only when real output contains escape codes                     |
| Terminal emulator (xterm.js)          | Not used. We run the CLI headless, there is no PTY                                                                 |
| File diffs                            | Not in the MVP. Show tool input as code                                                                            |

**Styling**

- Tailwind CSS (Streamdown ships Tailwind-styled output and needs its files registered in the Tailwind setup).
- **No shadcn/ui.** Build the few components we need ourselves. If an accessible primitive is
  needed (dialog, menu), prefer a headless library over a styled component kit.

**Security: agent output is untrusted**

- Streamdown's defaults are permissive (raw HTML enabled; link and image prefixes allow `*`).
  Override them: disable raw HTML, and restrict `allowedLinkPrefixes` and
  `allowedImagePrefixes` (images from external URLs are a data-exfiltration channel).
- Tool output is always rendered as text, never as HTML.

## Architecture

Modular monolith: one Bun server process, strict module boundaries.

- A module exposes its public API only through its `index.ts`.
- Never import another module's internals.
- Modules talk through exported functions/interfaces or an in-process event bus.
- Channels (web now, Telegram later) are thin inbound adapters that call the same
  application services. A channel never talks to the CLI directly.
- Providers are adapters behind one `AgentProvider` interface. UI and channels
  must not know provider-specific output formats.

## CLI integration

- Run the CLI headless, as a child process (`Bun.spawn`), never by wrapping its TUI.
- Antigravity: `agy -p <prompt> --output-format stream-json` (NDJSON on stdout).
- The CLI is installed inside the app's install directory (e.g. `<root>/cli/antigravity/`)
  using the installer's `--dir` option. Resolve the binary from config (`binPath`),
  never from `PATH`.
- Headless mode uses cached credentials: the user must log in once interactively.
- Each adapter translates raw CLI output into a provider-neutral run event model.
  Build and test parsers against real captured output (fixtures), not guesses.

## Security

- Headless `agy` was observed running with `permission_mode: always-proceed`, so the
  agent executes tools and commands without asking. Treat the app as remote code execution.
- Auth is mandatory even for local use. Bind to `127.0.0.1` by default.
- Run agents inside dedicated workspace directories, separate from app data
  (database, auth).
- Do not run agents with access to anything the app itself must protect.

## Proposed, not yet confirmed

- Storage: SQLite (`bun:sqlite`) with Drizzle; append-only event log per run
- Realtime to browser: SSE
- Web: SPA mode, explicit `routes.ts` (not file-system routes), Eden Treaty client with type-only import from `@opus/api`
- Per-user `HOME` for CLI processes, to isolate credentials and user-level skills/config

## Open questions

- Headless flags for resuming a conversation and for selecting the workspace
  (check `agy --help`)
- Whether a stricter permission mode exists for headless runs
- How CLI login works when the app is later moved into Docker

## Git & commits

- Use **Conventional Commits** without scope: `type: <emoji> subject`
  - Do not use scopes (e.g. use `feat: ✨ ...`, not `feat(auth): ✨ ...`).
  - The emoji goes right after the colon.
  - Emoji mapping:

    | Type       | Emoji | Example                                    |
    | ---------- | ----- | ------------------------------------------ |
    | `feat`     | ✨    | `feat: ✨ add refresh token rotation`      |
    | `fix`      | 🐛    | `fix: 🐛 handle empty response body`       |
    | `refactor` | ♻️    | `refactor: ♻️ extract query builder`       |
    | `perf`     | ⚡    | `perf: ⚡ cache parsed index`              |
    | `test`     | ✅    | `test: ✅ cover discount edge cases`       |
    | `docs`     | 📝    | `docs: 📝 document local setup`            |
    | `build`    | 📦    | `build: 📦 bump typescript to 5.6`         |
    | `ci`       | 👷    | `ci: 👷 cache node_modules`                |
    | `chore`    | 🔧    | `chore: 🔧 update gitignore`               |
    | `revert`   | ⏪    | `revert: ⏪ revert refresh token rotation` |

  - Subject: imperative mood, lowercase (after the emoji), no trailing period, max ~72 chars including the emoji.
  - Body (when needed): explain _why_, not _what_; wrap at ~72 chars.
  - Breaking changes: add `!` after type (`type!: <emoji> subject`) and a `BREAKING CHANGE:` footer.
  - One logical change per commit. Don't mix refactors with behavior changes.
  - Never commit directly to `main`/`master`. Work on a branch named `type/short-description`.
  - Never commit or push unless explicitly asked.

## Commands

- `bun run dev` - start dev servers via Turbo
- `bun run build` - build all apps via Turbo
- `bun run typecheck` - run typecheck across all workspaces via Turbo
- `bun run test` - run test suites via Turbo
- `bun run lint` - check linting and formatting via Biome
- `bun run lint:fix` - fix linting and formatting issues via Biome
- `bun run format` - format all files via Biome
