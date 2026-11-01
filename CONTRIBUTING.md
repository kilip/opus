# Contributing to opus

Thanks for your interest in contributing to opus! Please review this guide before submitting changes.

## Prerequisites

- [Bun](https://bun.sh) (v1.2+)
- Git

## Getting Started

1. Clone the repository:

   ```bash
   git clone https://github.com/<owner>/opus.git
   cd opus
   ```

2. Install dependencies:

   ```bash
   bun install
   ```

3. (Optional) Install the Antigravity CLI into the local `cli/` directory to test agent runs:

   ```bash
   curl -fsSL https://antigravity.google/cli/install.sh | bash -s -- --dir "$PWD/cli/antigravity"
   ```

## Development Workflow

### Commands

| Command             | Description                                 |
| ------------------- | ------------------------------------------- |
| `bun run dev`       | Start dev servers across apps via Turbo     |
| `bun run build`     | Build all workspaces via Turbo              |
| `bun run typecheck` | Run TypeScript checks across all workspaces |
| `bun run test`      | Run test suites across all workspaces       |
| `bun run lint`      | Check linting and formatting via Biome      |
| `bun run lint:fix`  | Fix auto-fixable lint and format issues     |
| `bun run format`    | Format all files using Biome                |

### Quality Checks

Before opening a pull request or submitting code, ensure all checks pass:

```bash
bun run lint
bun run typecheck
bun run test
```

## Branching & Git Workflow

- Always branch off `main`.
- Never commit directly to `main`.
- Branch naming format: `type/short-description` (e.g., `feat/login-page`, `fix/stream-parser`).

## Commit Guidelines

opus follows **Conventional Commits without scope**: `type: <emoji> subject`.

### Format

```text
type: <emoji> subject

[optional body]

[optional footer(s)]
```

### Emoji Mapping

| Type       | Emoji | Description                              | Example                                    |
| ---------- | ----- | ---------------------------------------- | ------------------------------------------ |
| `feat`     | ✨    | New feature                              | `feat: ✨ add refresh token rotation`      |
| `fix`      | 🐛    | Bug fix                                  | `fix: 🐛 handle empty response body`       |
| `refactor` | ♻️    | Code change that neither fixes nor adds  | `refactor: ♻️ extract query builder`       |
| `perf`     | ⚡    | Performance improvement                  | `perf: ⚡ cache parsed index`              |
| `test`     | ✅    | Adding or correcting tests               | `test: ✅ cover discount edge cases`       |
| `docs`     | 📝    | Documentation changes only               | `docs: 📝 document local setup`            |
| `build`    | 📦    | Build system or external dependencies    | `build: 📦 bump typescript to 5.6`         |
| `ci`       | 👷    | CI configuration files and scripts       | `ci: 👷 cache node_modules`                |
| `chore`    | 🔧    | Maintenance tasks, repo configs, tooling | `chore: 🔧 update gitignore`               |
| `revert`   | ⏪    | Reverting a previous commit              | `revert: ⏪ revert refresh token rotation` |

### Rules

- **No scope**: Use `type: <emoji>`, never `type(scope): <emoji>`.
- **Subject line**: Imperative mood, lowercase after the emoji, no trailing period, max ~72 characters including the emoji.
- **Body**: Explain _why_, not _what_. Wrap lines at ~72 characters.
- **Breaking changes**: Append `!` after type (`type!: <emoji> subject`) and include a `BREAKING CHANGE:` footer.
- **Atomic commits**: One logical change per commit. Do not combine refactors with functional changes.

## Architecture & Code Standards

- **Architecture**: Modular monolith with strict one-way dependencies.
- **Type Safety**: Strict TypeScript. Avoid `any`; use `unknown` and narrow types.
- **Style**: Code formatting and linting rules are enforced via Biome (`biome.json`).
- **Testing**: Deterministic tests using `bun:test`. Add regression tests for bug fixes and coverage for new features.
- **Security**: Never commit secrets or credentials. Treat agent CLI output as untrusted text.
