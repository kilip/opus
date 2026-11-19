# Operations Notes

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
