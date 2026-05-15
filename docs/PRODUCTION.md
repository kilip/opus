# Opus Production Deployment Guide

This guide covers the various ways to deploy **Opus v1.0.1** in a production environment.

## 🚀 Recommended: The Quick Installer

The easiest way to get Opus running on your machine (Linux, macOS, or Windows) is using our interactive installer.

```bash
npx opus install
```

### What the installer does:
1. **Detects your platform**: Automatically identifies your OS and architecture.
2. **Downloads the binary**: Fetches the latest production-ready Go binary from GitHub.
3. **Interactive Configuration**: Guides you through setting up your database, port, and OAuth credentials.
4. **Service Registration**: Installs Opus as a background service (systemd, launchd, or Windows Service).
5. **Security**: Auto-generates a secure 32-byte JWT secret if you don't provide one.

---

## 🐳 Docker Deployment

For containerized environments, we provide official Docker images and a `docker-compose.yml` for orchestration.

### Prerequisites
- Docker and Docker Compose installed.
- A `.env` file based on `.env.example`.

### Fast Track
```bash
# Clone the repository
git clone https://github.com/kilip/opus.git
cd opus

# Configure your environment
cp .env.example .env
# Edit .env with your credentials (Google Client ID, etc.)

# Start the stack
docker compose up -d
```

### Services
- **API**: Runs on port `8080` (Internal endpoint: `http://localhost:8080`).
- **Dash**: Runs on port `3000` (Access at: `http://localhost:3000`).

---

## 🏗️ Bare Metal (Manual Binary)

If you prefer to manage the binary yourself:

1. **Download**: Visit the [Latest Release](https://github.com/kilip/opus/releases/latest) and download the binary for your platform.
2. **Setup**:
   ```bash
   chmod +x opus-linux-amd64
   sudo mv opus-linux-amd64 /usr/local/bin/opus
   ```
3. **Initialize**:
   ```bash
   opus init
   ```
   This creates a default config at `~/.opus/config.toml`.
4. **Start**:
   ```bash
   opus start
   ```

---

## ⚙️ Configuration Reference

Opus uses a `config.toml` file located at `~/.opus/config.toml` by default.

| Key | Description | Default |
|-----|-------------|---------|
| `server.port` | The port the API will listen on. | `8080` |
| `database.driver` | `sqlite` or `postgres`. | `sqlite` |
| `database.path` | Path to the SQLite database file. | `~/.opus/opus.db` |
| `auth.secret` | 32-byte secret for JWT signing. | (Generated) |
| `auth.google_client_id` | Google OAuth2 Client ID. | (Required for OAuth) |

---

## 📱 Progressive Web App (PWA)

Opus is a fully compliant PWA. Once installed and running:
1. Open the dash in Chrome or Safari.
2. Click the **Install** icon in the address bar (Chrome) or **Add to Home Screen** (Safari iOS).
3. Opus will now appear as a native app on your desktop or mobile home screen.

---

## 🛠️ Management Commands

If installed via the installer or binary:
- `opus status`: Check if the agent is running.
- `opus logs`: View real-time logs.
- `opus restart`: Restart the agent.
- `opus stop`: Stop the agent.

---

> [!TIP]
> For production use, we highly recommend using a reverse proxy like **Nginx** or **Caddy** with HTTPS enabled to secure your SSE streams and OAuth callbacks.
