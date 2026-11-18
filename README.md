# 🎼 Opus

**Command Center for AI-Driven Software Factory**

---

## ✨ Features

- **📋 Tasks & Projects** — Organize work across repositories, track progress, and manage task labels and timelines from one dashboard.
- **🤖 Automated Vibe Runs** — Dispatch tasks to run autonomously in isolated git worktrees with real-time stage tracking.
- **🖥️ Worker Nodes** — Connect multiple machines to your cluster using join tokens and monitor their live status.
- **📡 Live Agent Sessions** — Watch agent activity and streamed output as tasks are executed.
- **🔄 GitHub Sync** — Sync issues, pull requests, labels, and statuses with your GitHub repositories.
- **🔔 Notifications** — Get Telegram alerts when runs finish or worker nodes go offline.
- **🔐 Passwordless Auth** — Sign in securely with email OTP or OAuth (GitHub, Google, Facebook).

---

## 📦 Installation

### ⚡ Quick Install

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/kilip/opus/main/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/kilip/opus/main/scripts/install.ps1 | iex
```

The installer downloads the latest release archive, verifies its `SHA-256` checksum against `checksums.txt`, and installs `opus` to `~/.local/bin` (macOS/Linux) or `%LOCALAPPDATA%\opus\bin` (Windows, `amd64` only). Override with `OPUS_VERSION` and `OPUS_INSTALL_DIR`. To uninstall, delete the installed `opus` binary. Worker machines also require `git`, `gh`, and `agy` in `PATH`.

**Manual download:** Prebuilt archives (`opus_<version>_<os>_<arch>.tar.gz` / `.zip`) and `checksums.txt` are published on [GitHub Releases](https://github.com/kilip/opus/releases). Verify with `sha256sum -c checksums.txt` before extracting.

### 🐳 Docker

```bash
docker build -t opus .
docker run -d -p 7880:7880 -v opus-data:/data opus
```

Or pull the prebuilt multi-arch image (`linux/amd64`, `linux/arm64`):

```bash
docker run -d -p 7880:7880 -v opus-data:/data ghcr.io/kilip/opus:latest
```

To enable `gh` CLI access for vibe runs, pass `-e GH_TOKEN=<token>` (minimum fine-grained PAT scopes: **Contents** read/write, **Issues** read/write, **Pull requests** read/write, **Metadata** read):

```bash
docker run -d -p 7880:7880 -v opus-data:/data -e GH_TOKEN=<token> opus
```

To authenticate `agy` inside the running container (persisted in `/data`; automated login is planned):

```bash
docker exec -it <container> agy auth login
```

The dashboard will be available at `http://localhost:7880`.

### ☸️ Kubernetes (Helm)

A Helm chart is included in `charts/`:

```bash
helm install opus ./charts
```

Or install directly from GHCR OCI:

```bash
helm install opus oci://ghcr.io/kilip/charts/opus --version <version>
```

- **Secrets (`auth.existingSecret`)**: By default the chart auto-generates `<release>-opus-auth` with `AUTH_SECRET` (reused on `helm upgrade` via `lookup`). Argo CD users (which render via `helm template`) should set `auth.existingSecret` to an existing Secret.
- **GitHub token (`github.existingSecret`)**: Create `kubectl create secret generic opus-github --from-literal=GH_TOKEN=<token>` and pass `--set github.existingSecret=opus-github` for vibe runs.
- **Ingress (`config.baseUrl`)**: When enabling `ingress.enabled=true`, set `config.baseUrl` (e.g. `https://opus.example.com`) for OAuth callbacks.

---

## 🚀 Quick Start

### 1. Start the Server

```bash
./opus serve
```

Running `./opus` with no subcommand also starts the server on port `7880`. On first run, Opus prints a one-time `Opus setup token` to stderr for completing owner onboarding at `/onboarding`.

### 2. Connect a Worker Node (Optional)

On each worker machine, make sure `gh` and `agy` are installed and logged in (`gh auth login` and `agy auth login`).

1. Generate a join token in the Opus UI under **Nodes**.
2. Join the cluster:
   ```bash
   opus join <join-url> --name my-worker
   ```
3. Start the worker daemon:
   ```bash
   opus worker
   ```

---

## ⚙️ Configuration

Opus is configured via environment variables. Common settings:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `APP_HOME` | `~/.opus` | Directory for repos, worktrees, and node config |
| `APP_BASE_URL` | `http://localhost:7880` | Public URL for OAuth callbacks |
| `APP_DATABASE_URL` | `file:opus.db?...` | SQLite / LibSQL database connection string |
| `AUTH_SECRET` | — | Secret key for signing session cookies (min 32 chars; persisted in `APP_HOME/auth_secret` if unset) |
| `AUTH_SETUP_TOKEN` | — | First-run token for `/onboarding` (auto-generated on stderr if unset) |
| `AUTH_COOKIE_SECURE` | `auto` | Session cookie `Secure` flag (`auto`, `true`, `false`) |
| `APP_TRUSTED_PROXIES` | — | Comma-separated trusted proxy IPs/CIDRs for `X-Forwarded-For` |
| `NOTIFY_TELEGRAM_BOT_TOKEN` | — | Telegram bot token for alerts |
| `NOTIFY_TELEGRAM_CHAT_ID` | — | Telegram chat ID for alerts |
| `RUN_POLL_INTERVAL` | `30s` | Interval between run reconciler scans |
| `RUN_STAGE_TIMEOUT` | `2h` | Timeout for an active run stage |

---

## 🛠️ Development

With `mise` installed:

```bash
# Run API and UI dev servers together
mise run dev

# Format, lint, and test everything
mise run qc
```

---

<p align="center">
  Made with ❤️ and ☕ — love and caffeine
</p>
