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

The installer downloads the latest release archive, verifies its `SHA-256` checksum against `checksums.txt`, and installs `opus` to `~/.local/bin` (macOS/Linux) or `%LOCALAPPDATA%\opus\bin` (Windows, `amd64` only). Override with `OPUS_VERSION` and `OPUS_INSTALL_DIR`. To uninstall, run `opus uninstall` (add `--purge` to also remove `$APP_HOME` / `~/.opus`), or manually delete the installed `opus` binary. Worker machines also require `git`, `gh`, and `agy` in `PATH`.

**Manual download:** Prebuilt archives (`opus_<version>_<os>_<arch>.tar.gz` / `.zip`) and `checksums.txt` are published on [GitHub Releases](https://github.com/kilip/opus/releases). Verify with `sha256sum -c checksums.txt` before extracting.

### 🔄 Self-Update

Update an installed `opus` binary in place from GitHub Releases (including prereleases) with SHA-256 verification:

```bash
opus self-update
```

- `--check` — Check for updates without installing; exits `0` when up to date or `10` when a newer release is available.
- `--version <v>` — Install a specific release version (e.g. `0.3.0` or `v0.3.0`).
- `--yes` (`-y`) — Skip confirmation prompts and allow downgrades when `--version` is older than the current binary.

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

Opus is configured via environment variables (a `.env` file in the working directory is auto-loaded on startup). See [.env.example](.env.example), [Production Deployment Guide](docs/deploy.md), and [Backup and Restore](docs/backup.md).

### `APP_*`

| Variable | Default | Description | Required in production |
| :--- | :--- | :--- | :--- |
| `APP_NAME` | `opus` | Application name used in HTTP headers and logs | No |
| `APP_HOME` | `~/.opus` | Directory for repos, worktrees, `auth_secret`, and `node.json` | Recommended |
| `APP_HOST` | `0.0.0.0` | TCP bind address for the HTTP server | No |
| `APP_PORT` | `7880` | TCP listen port for the HTTP server | No |
| `APP_LOG_LEVEL` | `info` | Structured log level (`debug`, `info`, `warn`, `error`) | No |
| `APP_READ_TIMEOUT` | `10s` | HTTP server read timeout | No |
| `APP_WRITE_TIMEOUT` | `10s` | HTTP server write timeout (SSE streams clear this per connection) | No |
| `APP_IDLE_TIMEOUT` | `60s` | HTTP keep-alive idle timeout | No |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout | No |
| `APP_BODY_LIMIT` | `4194304` | Maximum HTTP request body size in bytes (4 MiB) | No |
| `APP_DATABASE_URL` | `file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)` | SQLite DSN (WAL mode and `_txlock=immediate` added automatically) | Recommended |
| `APP_BASE_URL` | `http://localhost:7880` | Canonical public URL for OAuth callbacks, HSTS, and origin checks | **Yes** |
| `APP_TRUSTED_ORIGINS` | — | Comma-separated additional allowed CORS / CSRF / WebSocket origins | If multi-origin |
| `APP_TRUSTED_PROXIES` | — | Comma-separated trusted proxy IPs/CIDRs for `X-Forwarded-For` | If behind proxy |
| `APP_VERSION` | — | Optional version string exposed to the UI via `/config.js` | No |
| `APP_ALLOW_DOWNGRADE` | `false` | Allow starting a binary older than the stored schema version | No |
| `APP_SENTRY_DSN` | — | Sentry DSN for backend and UI error reporting | No |
| `APP_SENTRY_ENVIRONMENT` | `development` | Sentry environment tag | No |
| `APP_SENTRY_TRACES_SAMPLE_RATE` | `1.0` | Sentry performance trace sample rate (`0.0`–`1.0`) | No |
| `APP_SMTP_HOST` | — | SMTP server host (falls back to console mailer when unset) | For email OTP |
| `APP_SMTP_PORT` | `587` | SMTP server port | For email OTP |
| `APP_SMTP_USERNAME` | — | SMTP authentication username | If SMTP auth |
| `APP_SMTP_PASSWORD` | — | SMTP authentication password | If SMTP auth |
| `APP_SMTP_FROM` | — | Outgoing sender email address | For email OTP |

### `AUTH_*`

| Variable | Default | Description | Required in production |
| :--- | :--- | :--- | :--- |
| `AUTH_SECRET` | — | Secret key for signing session cookies (min 32 chars; persisted in `APP_HOME/auth_secret` if unset) | **Yes** |
| `AUTH_SESSION_TTL` | `720h` | Session cookie lifetime | No |
| `AUTH_SETUP_TOKEN` | — | First-run token for `/onboarding` (auto-generated on stderr if unset) | No |
| `AUTH_COOKIE_SECURE` | `auto` | Session cookie `Secure` flag (`auto`, `true`, `false`) | No |
| `AUTH_GITHUB_ID` | — | GitHub OAuth client ID | For GitHub OAuth |
| `AUTH_GITHUB_SECRET` | — | GitHub OAuth client secret | For GitHub OAuth |
| `AUTH_GOOGLE_ID` | — | Google OAuth client ID | For Google OAuth |
| `AUTH_GOOGLE_SECRET` | — | Google OAuth client secret | For Google OAuth |
| `AUTH_FACEBOOK_ID` | — | Facebook OAuth app ID | For Facebook OAuth |
| `AUTH_FACEBOOK_SECRET` | — | Facebook OAuth app secret | For Facebook OAuth |

### `TRACKER_*`, `RUN_*`, `WORKER_*`, and `NOTIFY_*`

| Variable | Default | Description | Required in production |
| :--- | :--- | :--- | :--- |
| `TRACKER_POLL_INTERVAL` | `60s` | Interval between GitHub tracker sync polling cycles | No |
| `RUN_POLL_INTERVAL` | `30s` | Interval between run reconciler scans | No |
| `RUN_STAGE_TIMEOUT` | `2h` | Inactivity timeout for an active run stage | No |
| `RUN_NODE_GRACE` | `2m` | Grace period before failing active runs on a disconnected worker | No |
| `WORKER_AGY_BIN` | `agy` | Path or binary name for the `agy` ACP CLI on worker nodes | No |
| `NOTIFY_TELEGRAM_BOT_TOKEN` | — | Telegram bot token for alerts | No |
| `NOTIFY_TELEGRAM_CHAT_ID` | — | Telegram chat ID for alerts | No |

---

## 🛠️ Development

With `mise` installed:

```bash
# Run API and UI dev servers together
mise run dev

# Format, lint, and test everything
mise run qc
```

See [Operations Notes](docs/operations.md), [Production Deployment Guide](docs/deploy.md), [Backup and Restore](docs/backup.md), and [Contributing](CONTRIBUTING.md).

---

## 📄 License

Licensed under the [MIT License](LICENSE).

---

<p align="center">
  Made with ❤️ and ☕ — love and caffeine
</p>
