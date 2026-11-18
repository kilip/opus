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
