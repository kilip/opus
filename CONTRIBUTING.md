# 🤝 Contributing to Opus

Thanks for wanting to contribute to **Opus**! Here's how to get started and keep changes clean.

---

## 🛠️ Development Setup

We use [mise](https://mise.jdx.dev/) to manage Go, Bun, Node, and lint/build tools.

```bash
# Start both the Go API and React Router UI dev servers
mise run dev
```

If you modify backend API handlers or DTOs, regenerate the OpenAPI spec and TypeScript types:

```bash
mise run api
```

---

## 🌿 Branching & Commits

- Create a feature branch from `main` (e.g. `feat/my-feature`, `fix/bug-name`, `docs/update-readme`).
- Keep git history linear (rebase / fast-forward).
- Commit messages are checked by `commitlint` and must follow:

```text
<type>(<scope>): <subject>
```

- **Allowed scopes**: `api`, `ui`, `deps`, `release`, `tools`, `deploy`, `docs`
- **Line length**: Header and body lines must be ≤ 80 characters.
- Keep commits atomic — don't bundle unrelated changes into one commit.

---

## ✅ Quality Checks

Before opening a pull request, run the full quality check:

```bash
mise run qc
```

Or run individual checks:

- `mise run format` — Format Go and UI files
- `mise run lint` — Run Go vet, `golangci-lint`, Biome, and TypeScript checks
- `mise run test` — Run Go tests (`-race`) and UI tests

---

## 📬 Pull Requests

1. Keep PRs focused on a single topic or issue.
2. Make sure `mise run qc` passes cleanly.
3. Describe what changed and why in the PR description.
