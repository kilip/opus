# Operations Notes

For production deployment, reverse proxy, and backup procedures, see the [Production Deployment Guide](deploy.md) and [Backup and Restore](backup.md).

## Worker Node Setup

Worker nodes execute `vibe` runs inside isolated git worktrees using `gh` and `agy`.

### 1. Prerequisites (`gh` and `agy`)

Install and manually authenticate both CLIs on each worker machine before joining the cluster:

1. **GitHub CLI (`gh`)**:
   ```bash
   gh auth login
   gh auth status
   ```
2. **Antigravity CLI (`agy`)**:
   Install `agy` (or set `WORKER_AGY_BIN` to its path) and complete interactive login:
   ```bash
   agy auth login
   ```
   The worker reports `agy` and `gh` readiness (`ready`, `unauthenticated`, `missing`) to the master node on connect.

### 2. Join the Cluster (`opus join`)

Generate a join token in the Opus UI (`/nodes`) and run `opus join` on the worker node:

```bash
opus join <join-url> [--name <node-name>]
```

- `--name`: Optional node name (defaults to the machine hostname).
- Saves node credentials to `$APP_HOME/node.json` (or `~/.opus/node.json` when `APP_HOME` is unset).

### 3. Run the Worker Daemon (`opus worker`)

Start the worker process using the saved configuration:

```bash
WORKER_AGY_BIN=agy opus worker
```

- `WORKER_AGY_BIN`: Path or binary name for `agy` (defaults to `agy`).
- `APP_HOME`: Base directory for cloned repos (`<APP_HOME>/repos/<slug>`), worktrees (`<APP_HOME>/wt/<project>/<task-key>`), and `node.json`.
- `GH_TOKEN`: Passed through from the worker environment to `agy` subprocesses so `gh` commands can create and merge pull requests.

## Run as a Service

Manage Opus (`server` or `worker`) as a background service using `opus service`:

```bash
opus service install [--role server|worker] [--system]
opus service start   [--role server|worker] [--system]
opus service stop    [--role server|worker] [--system]
opus service restart [--role server|worker] [--system]
opus service status  [--role server|worker] [--system]
opus service uninstall [--role server|worker] [--system]
```

- **`--role`** (default `server`): `server` runs `opus serve`; `worker` runs `opus worker` (requires `$APP_HOME/node.json` from `opus join`).
- **`--system`**: Install a system-wide unit (`/etc/systemd/system/opus.service` on Linux, `/Library/LaunchDaemons/com.kilip.opus.plist` on macOS) instead of a user-level service.
- **Environment file (`$APP_HOME/opus.env`, mode `0600`)**: Created on `install` if absent (generating `AUTH_SECRET` for `server`), never overwritten if it already exists, and preserved on `uninstall`. Secrets are never embedded in unit, plist, or task XML files.
- **Backends & Logs**:
  - **Linux (`systemd`)**: `~/.config/systemd/user/opus.service` (or `opus-worker.service`), `Restart=on-failure`, `RestartSec=2`. Logs go to `journalctl --user -u opus`.
  - **macOS (`launchd`)**: `~/Library/LaunchAgents/com.kilip.opus.plist`, logs to `~/Library/Logs/opus.log` (or `opus-worker.log`).
  - **Windows (`schtasks`)**: Scheduled Task `Opus` (or `OpusWorker`), logs to `%LOCALAPPDATA%\opus\opus.log`.
- **`opus service status` exit codes**: `0` running, `3` installed but stopped, `4` not installed.

### 4. Run Execution & Worktree Lifecycle

- **`start_run`**: Verifies `agy` is installed and authenticated (`agy auth status`), clones or validates the project repository, creates an isolated git worktree at `<APP_HOME>/wt/<project>/<task-key>` on the task branch, starts `agy acp` with policy enforcement and Opus MCP injection, streams `session/update` events, and reports `session_completed` (or `run_error` / `run_cancelled`).
- **`cleanup_worktree`**: Removes the task worktree directory and deletes the local task branch when a run finishes or is cleaned up.

## Uninstall

Remove the installed `opus` binary while preserving your data directory (`$APP_HOME`, defaulting to `~/.opus`):

```bash
opus uninstall
```

To also remove the data directory, pass `--purge` (you will be prompted to type `purge` unless `--yes` is passed):

```bash
opus uninstall --purge
```

### Manual Removal

- **macOS / Linux**: Delete the `opus` binary (for example `~/.local/bin/opus`), and optionally remove `$APP_HOME` (`~/.opus`).
- **Windows**: Delete `opus.exe` (and any leftover `opus.exe.old`) from `%LOCALAPPDATA%\opus\bin` (or `$env:OPUS_INSTALL_DIR`), remove that directory from your User `Path` environment variable, and optionally remove `$env:APP_HOME` (`%USERPROFILE%\.opus`).

### Docker & Kubernetes

`opus uninstall` refuses to run inside containers. Stop and remove the Docker container (or uninstall the Helm release) instead.

When running the built-in worker inside Docker or Kubernetes:

- Set `GH_TOKEN` (via Secret or `-e GH_TOKEN=...`) with **Contents**, **Issues**, **Pull requests** (read/write) and **Metadata** (read) scopes.
- Complete `agy auth login` interactively inside the container (`docker exec -it <container> agy auth login` or `kubectl exec -it ... -- agy auth login`) so credentials persist in `/data`; automated login is planned.

## Security Model

### Run-Scoped MCP Token

- Every active run receives a unique 32-byte random Bearer token (`mcp_token`) generated in `PrepareAndStartSession`.
- Only the SHA-256 hash (`mcp_token_hash`) is stored on the `Run` record; the raw token is never persisted to SQLite or logged.
- `POST /mcp` requires `X-Opus-Run-ID` for Bearer callers, verifies the token with constant-time comparison, rejects terminal runs (`merged`, `failed`, `cancelled`), and scopes all MCP tools strictly to that run's project.
- Workers inject `mcp_token` (never the node's long-lived `credential`) into `agy` MCP server headers and fail closed if `mcp_token` is absent.

### Node Credential Capabilities

- Long-lived node credentials authenticate `/api/nodes/connect` WebSocket connections and node-level envelopes.
- Node-originated session events (`HandleWorkerEvent`) are verified against the session's assigned `node_id`; events from mismatched nodes are ignored and logged.

### WebSocket Limits & Origin Validation

- Inbound WebSocket frames on `/api/nodes/connect` are capped at **1 MiB** (`1 << 20` bytes).
- Connections use a 60-second read deadline extended by Pong frames, with server Ping keepalives every 20 seconds.
- Non-empty `Origin` headers are restricted to `APP_BASE_URL` and `APP_TRUSTED_ORIGINS` (empty `Origin` remains allowed for non-browser worker clients).
