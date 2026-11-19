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

### Testing Helm Charts & Install Scripts

```bash
# Lint and render Helm chart with CI values
helm lint charts --strict
helm template opus charts -f charts/ci/default-values.yaml > /dev/null

# Run Unix install script against a local fixture or in CI
shellcheck scripts/install.sh
```

For full Kubernetes smoke testing, load a local image into `kind` and install with `helm install opus ./charts`.

---

## 🚀 Releases

Releases are automated by `release-please`:

1. Conventional commits merged to `main` update the open `release-please` PR.
2. Merging the release PR creates a version tag (`vX.Y.Z`) and GitHub Release.
3. Tag workflows build and publish cross-platform binaries (`checksums.txt`), multi-arch Docker images (`ghcr.io/kilip/opus`), and the OCI Helm chart (`oci://ghcr.io/kilip/charts/opus`).

---

## 📬 Pull Requests

1. Keep PRs focused on a single topic or issue.
2. Make sure `mise run qc` passes cleanly.
3. Describe what changed and why in the PR description.
