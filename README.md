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

### 🐳 Docker

```bash
docker build -t opus .
docker run -d -p 7880:7880 -v opus-data:/data opus
```

The dashboard will be available at `http://localhost:7880`.

### ☸️ Kubernetes (Helm)

A Helm chart is included in `charts/`:

```bash
helm install opus ./charts
```

---

## 🚀 Quick Start

### 1. Start the Server

```bash
./opus serve
```

Running `./opus` with no subcommand also starts the server on port `7880`.

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
| `AUTH_SECRET` | — | Secret key for signing session cookies |
| `NOTIFY_TELEGRAM_BOT_TOKEN` | — | Telegram bot token for alerts |
| `NOTIFY_TELEGRAM_CHAT_ID` | — | Telegram chat ID for alerts |

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
