# opus

A web UI for LLM provider CLIs. opus drives the real CLIs (headless, as child processes)
and gives you a browser interface on top of them, instead of reimplementing them.

> **Status: early development / proof of concept.** Nothing here is stable yet.

## Scope

**MVP**

- Web interface
- [Antigravity CLI](https://antigravity.google) (`agy`) as the only provider
- Self-hosted, runs locally on one machine

**Planned**

- Claude Code provider
- Telegram channel to send commands
- Docker packaging
- Multi-user

## Stack

| Layer                     | Choice                                              |
| ------------------------- | --------------------------------------------------- |
| Runtime / package manager | [Bun](https://bun.sh)                               |
| Monorepo                  | Bun workspaces + [Turborepo](https://turborepo.dev) |
| Frontend                  | React Router 8                                      |
| Backend                   | ElysiaJS                                            |
| Auth                      | better-auth                                         |

The backend is a modular monolith: one server process with strict module boundaries.
Providers are adapters behind a common interface, so adding a CLI means adding an adapter.

## Repository layout

```
apps/web         React Router UI
apps/api         ElysiaJS backend
packages/shared  types shared by web and api (run events, API contracts)
cli/             installed provider CLIs (gitignored)
data/            local database (gitignored)
workspaces/      directories the agents work in (gitignored)
```

See [AGENTS.md](./AGENTS.md) for the full structure and the architecture rules.

## Getting started

Prerequisites: [Bun](https://bun.sh).

```bash
bun install
```

Install the Antigravity CLI into the project's `cli/` directory:

```bash
curl -fsSL https://antigravity.google/cli/install.sh | bash -s -- --dir "$PWD/cli/antigravity"
```

Headless mode uses cached credentials, so sign in once by running `agy` interactively
before starting opus.

Run everything in development:

```bash
bun run dev
```

Other root scripts: `bun run build`, `bun run typecheck`, `bun run test`.

## Security

opus gives an AI agent the ability to run commands on the machine it is installed on.
In headless mode `agy` was observed running with permission mode `always-proceed`, meaning
tools run without asking for approval.

- Keep opus bound to `127.0.0.1` unless you know what you are doing.
- Do not expose it to the internet without authentication in front of it.
- Agents work in dedicated directories under `workspaces/`, separate from app data.

## Contributing

Read [AGENTS.md](./AGENTS.md) first. It describes the module boundaries and conventions
that humans and coding agents are both expected to follow.
