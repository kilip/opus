# ADR-013: `npx get-opus` Installer

**Status:** Accepted
**Date:** 2026-05-18
**Deciders:** Chief Architect
**Context:** Opus Installer (`get-opus/`)

---

## 1. Context

Opus is a self-hosted, autonomous AI assistant. For a self-hosted tool to achieve broad adoption,
the installation experience must be frictionless. A developer or non-technical user must be able
to go from zero to a running Opus instance in a single command, without reading documentation,
configuring a process manager, or manually downloading binaries.

The `npx get-opus` installer is the canonical entry point for all new Opus installations. It is
the first thing a user runs and therefore the first impression of the project. A poor installation
experience directly reduces adoption and increases support burden.

This ADR establishes the technology stack, installation flow, binary distribution strategy,
interactive setup convention, service management approach, and project structure for the
`get-opus/` package.

> **Note for AI agents and automated tooling:** This ADR covers the installer only. Upgrade,
> uninstall, and other lifecycle operations are out of scope and will be addressed in a future ADR
> covering the `opus` CLI binary. Do not infer or implement lifecycle commands (`opus update`,
> `opus uninstall`) within the installer.

---

## 2. Decision

The `get-opus` installer is a **zero-dependency Node.js CLI script** that downloads the
appropriate pre-built `opus-server` binary from GitHub Releases, runs a minimal interactive
setup wizard, generates a `config.json` with sensible defaults, auto-configures a system service,
and opens the Opus Dash onboarding flow in the user's browser.

---

### 2.1 Technology Stack

| Concern | Choice | Rationale |
|---|---|---|
| Runtime | Node.js >= 24 | LTS; native `fetch`, `fs`, `path`, `os`, `child_process` — no polyfills needed |
| Dependencies | **Zero npm dependencies** | Security, reliability, and auditability; all required functionality is available in Node.js built-ins |
| Language | JavaScript (ESM) | No build step required; `npx` executes directly |
| Archive format | `.tar.gz` (Linux/macOS), `.zip` (Windows) | Standard formats; extractable with built-in Node.js `zlib` and `child_process` |
| Distribution | GitHub Releases | Zero infrastructure cost; standard open-source convention |

#### Rationale — Zero Dependencies

The installer is a privileged operation — it runs with the user's permissions, downloads and
executes a binary, and modifies system configuration. Every additional npm dependency is an
additional supply-chain attack surface. Node.js >= 24 provides all required primitives
(`fetch`, `fs/promises`, `zlib`, `child_process`, `readline`) without external packages.

#### Rationale — Node.js >= 24

Node.js 24 is the current LTS release as of 2026. It provides:

- Native `fetch` API — no `node-fetch` or `axios` required
- `fs/promises` — clean async file operations
- `readline/promises` — async interactive prompts without `inquirer`
- Broad platform availability across Linux, macOS, and Windows

---

### 2.2 Binary Distribution Strategy

Pre-built `opus-server` binaries are attached to each GitHub Release as release assets.

#### 2.2.1 Supported Platforms

| OS | Architecture | Asset Name |
|---|---|---|
| Linux | x64 | `opus-server-linux-amd64.tar.gz` |
| Linux | arm64 | `opus-server-linux-arm64.tar.gz` |
| macOS | x64 (Intel) | `opus-server-darwin-amd64.tar.gz` |
| macOS | arm64 (Apple Silicon) | `opus-server-darwin-arm64.tar.gz` |
| Windows | x64 | `opus-server-windows-amd64.zip` |

#### 2.2.2 Asset Resolution

The installer resolves the correct asset at runtime using `os.platform()` and `os.arch()`:

```javascript
// get-opus/src/platform.js
import os from 'os';

const PLATFORM_MAP = {
  'linux-x64':   'opus-server-linux-amd64.tar.gz',
  'linux-arm64': 'opus-server-linux-arm64.tar.gz',
  'darwin-x64':  'opus-server-darwin-amd64.tar.gz',
  'darwin-arm64':'opus-server-darwin-arm64.tar.gz',
  'win32-x64':   'opus-server-windows-amd64.zip',
};

export function resolveAssetName() {
  const key = `${os.platform()}-${os.arch()}`;
  const asset = PLATFORM_MAP[key];
  if (!asset) {
    throw new Error(`Unsupported platform: ${key}. Supported: ${Object.keys(PLATFORM_MAP).join(', ')}`);
  }
  return asset;
}
```

#### 2.2.3 Download URL Convention

```
https://github.com/kilip/opus/releases/download/{version}/{asset}
```

The installer fetches the latest release tag from the GitHub API before constructing the
download URL:

```
GET https://api.github.com/repos/kilip/opus/releases/latest
```

---

### 2.3 Installation Flow

The installer executes the following steps in order. Each step is logged to stdout with a
clear status indicator (`✓`, `✗`, `→`).

```
npx get-opus
  │
  ├─ 1. Preflight Checks
  │     ├─ Node.js >= 24 (hard fail if not met)
  │     ├─ Network connectivity (warn if offline)
  │     └─ Existing installation detection
  │
  ├─ 2. Minimal Interactive Setup (3 questions max)
  │     ├─ Port (default: 8080)
  │     ├─ Data directory (default: ~/.opus)
  │     └─ Auto-setup system service? [Y/n]
  │
  ├─ 3. Binary Download & Extraction
  │     ├─ Fetch latest release version from GitHub API
  │     ├─ Download platform-specific asset
  │     ├─ Verify SHA-256 checksum against release manifest
  │     └─ Extract binary to {dataDir}/bin/opus-server
  │
  ├─ 4. Configuration Generation
  │     └─ Write {dataDir}/config.json with user answers + sensible defaults
  │
  ├─ 5. Service Setup (if confirmed)
  │     ├─ Linux  → systemd user service (~/.config/systemd/user/opus.service)
  │     ├─ macOS  → launchd user agent (~/Library/LaunchAgents/com.opus.server.plist)
  │     └─ Windows → NSSM-based Windows Service (if NSSM available) or Task Scheduler
  │
  ├─ 6. First Start
  │     └─ Start opus-server process (via service or directly)
  │
  └─ 7. Onboarding Handoff
        ├─ Wait for server health check (GET /health, max 30s)
        └─ Open http://localhost:{port} in default browser
```

#### 2.3.1 Preflight — Node.js Version Check

```javascript
// get-opus/src/preflight.js
import { execSync } from 'child_process';

const MINIMUM_NODE_MAJOR = 24;

export function checkNodeVersion() {
  const [major] = process.versions.node.split('.').map(Number);
  if (major < MINIMUM_NODE_MAJOR) {
    console.error(
      `✗ Node.js ${MINIMUM_NODE_MAJOR}+ is required. ` +
      `You are running ${process.version}.\n` +
      `  → Install the latest LTS: https://nodejs.org`
    );
    process.exit(1);
  }
}
```

#### 2.3.2 Existing Installation Detection

If `{dataDir}/bin/opus-server` already exists, the installer prompts the user:

```
→ An existing Opus installation was detected at ~/.opus.
  This installer is for first-time setup only.
  To upgrade, run: opus update
  To reinstall, remove ~/.opus and run npx get-opus again.
```

The installer then exits with code `0`. It does not attempt to upgrade or overwrite an existing
installation.

#### 2.3.3 Checksum Verification

Each GitHub Release includes a `checksums.sha256` manifest file listing SHA-256 hashes for all
release assets. The installer downloads this file alongside the binary and verifies the hash
before extraction using Node.js built-in `crypto.createHash('sha256')`.

If verification fails, the downloaded file is deleted and the installer exits with a clear error
message and a non-zero exit code.

---

### 2.4 Minimal Interactive Setup

The installer asks **no more than three questions**. All other configuration is deferred to
Opus Dash after the server is running.

| # | Question | Default | Notes |
|---|---|---|---|
| 1 | `HTTP port` | `8080` | Validated: must be 1024–65535 and not in use |
| 2 | `Data directory` | `~/.opus` | Created if it does not exist |
| 3 | `Set up system service for auto-start?` | `Y` | Skip with `--no-service` flag |

**Silent mode:** All questions can be bypassed with CLI flags for scripted/CI installation:

```bash
npx get-opus --port 8080 --data-dir ~/.opus --no-service
```

**Rationale for minimal questions:** Opus Dash provides a full setup wizard for LLM API keys,
workspace configuration, user accounts, and all other settings. The installer's only
responsibility is to get the server running so the user can reach the Dash UI.

---

### 2.5 Generated `config.json`

The installer generates `{dataDir}/config.json` with the following defaults. All values are
overridable via Opus Dash after first run.

```json
{
  "$schema": "https://github.com/kilip/opus/releases/latest/download/config.schema.json",
  "server": {
    "address": ":{port}",
    "debug": false
  },
  "database": {
    "driver": "sqlite3",
    "dsn": "{dataDir}/opus.db"
  },
  "log": {
    "level": "info",
    "format": "json"
  },
  "agent": {
    "tick_interval": "60s",
    "max_retries": 3
  },
  "queue": {
    "driver": "database",
    "concurrency": 10
  }
}
```

**Notes:**

- `llm.api_key` is intentionally absent — it must be set via Opus Dash or the
  `OPUS_LLM_APIKEY` environment variable. It must never be written to `config.json`.
- The `$schema` field enables IDE autocompletion when the user edits the config manually.

---

### 2.6 Service Management

System service setup is **automatic by default** and **opt-out via `--no-service`**.

**Rationale:** Opus is a 24/7 autonomous assistant. If the service does not survive a system
reboot, the core value proposition — "always on" — is broken. Users who are comfortable managing
processes themselves (Docker, supervisord, custom scripts) can opt out explicitly.

#### 2.6.1 Linux — systemd User Service

The installer creates a systemd **user** service (not a system service) to avoid requiring
`sudo`. User services start automatically on user login.

```ini
# ~/.config/systemd/user/opus.service
[Unit]
Description=Opus Autonomous AI Assistant
After=network.target

[Service]
Type=simple
ExecStart={dataDir}/bin/opus-server
WorkingDirectory={dataDir}
Restart=on-failure
RestartSec=5s
Environment=OPUS_HOME={dataDir}

[Install]
WantedBy=default.target
```

The installer runs:

```bash
systemctl --user enable opus.service
systemctl --user start opus.service
```

**Limitation:** `systemctl --user` services only auto-start after user login, not at system
boot, unless `loginctl enable-linger {username}` is also run. The installer runs this command
automatically and notes it in the completion output.

#### 2.6.2 macOS — launchd User Agent

```xml
<!-- ~/Library/LaunchAgents/com.opus.server.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.opus.server</string>
  <key>ProgramArguments</key>
  <array>
    <string>{dataDir}/bin/opus-server</string>
  </array>
  <key>WorkingDirectory</key>
  <string>{dataDir}</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>OPUS_HOME</key>
    <string>{dataDir}</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>{dataDir}/logs/opus.log</string>
  <key>StandardErrorPath</key>
  <string>{dataDir}/logs/opus-error.log</string>
</dict>
</plist>
```

The installer runs:

```bash
launchctl load ~/Library/LaunchAgents/com.opus.server.plist
```

#### 2.6.3 Windows — Task Scheduler

On Windows, the installer creates a Task Scheduler task that runs at user logon using
`schtasks.exe`, which is available on all Windows versions without additional tools.

```javascript
// get-opus/src/service/windows.js
import { execSync } from 'child_process';

export function installWindowsService({ binaryPath, dataDir }) {
  const taskName = 'OpusServer';
  execSync(
    `schtasks /create /tn "${taskName}" /tr "${binaryPath}" ` +
    `/sc onlogon /rl limited /f`,
    { stdio: 'pipe' }
  );
}
```

**Note:** NSSM (Non-Sucking Service Manager) provides a more robust Windows service experience
but requires a separate download. Task Scheduler is used as the zero-dependency default.
NSSM support may be introduced in a future ADR.

---

### 2.7 Directory Structure

```
get-opus/
├── package.json          # name: "get-opus"; bin: { "get-opus": "./src/index.js" }
├── README.md
└── src/
    ├── index.js          # Entry point; orchestrates all steps
    ├── preflight.js      # Node.js version check, network check, existing install detection
    ├── platform.js       # OS/arch detection, asset name resolution
    ├── prompt.js         # Minimal interactive wizard (readline/promises)
    ├── download.js       # GitHub API, binary download, checksum verification
    ├── extract.js        # .tar.gz (zlib) and .zip (child_process unzip) extraction
    ├── config.js         # config.json generation
    ├── health.js         # Server health check + browser open
    └── service/
        ├── index.js      # Service setup dispatcher (detects platform)
        ├── linux.js      # systemd user service
        ├── macos.js      # launchd user agent
        └── windows.js    # Task Scheduler
```

---

### 2.8 `package.json` Convention

```json
{
  "name": "get-opus",
  "version": "0.1.0",
  "description": "Installer for Opus — self-hosted autonomous AI assistant",
  "type": "module",
  "bin": {
    "get-opus": "./src/index.js"
  },
  "engines": {
    "node": ">=24"
  },
  "dependencies": {},
  "devDependencies": {}
}
```

**Rules:**

- `dependencies` must remain empty at all times. Any proposal to add a runtime dependency
  requires a new ADR amendment with an explicit security justification.
- `devDependencies` may include test utilities (e.g. `vitest`) for the installer's own test suite.
- The `engines.node` field enforces the Node.js >= 24 requirement at `npx` execution time.

---

### 2.9 Error Handling and Exit Codes

| Exit Code | Meaning |
|---|---|
| `0` | Success or existing installation detected |
| `1` | Preflight failure (unsupported Node.js, unsupported platform) |
| `2` | Download or checksum verification failure |
| `3` | File system error (cannot create data directory, cannot write config) |
| `4` | Service setup failure |
| `5` | Server failed to start within health check timeout |

All errors are printed to `stderr` with a clear human-readable message and a suggested
remediation action where possible.

---

### 2.10 Completion Output

On successful installation, the installer prints a clear summary:

```
✓ Opus installed successfully!

  Version:      v0.3.1
  Data dir:     ~/.opus
  Config:       ~/.opus/config.json
  Binary:       ~/.opus/bin/opus-server
  Service:      systemd user service (opus.service) — enabled and running

  Opening Opus Dash at http://localhost:8080 …

  Complete your setup in the browser:
    • Create your admin account
    • Configure your LLM provider (Anthropic, OpenAI, or Ollama)
    • Connect integrations (Gmail, Google Drive, Telegram, etc.)

  Tip: To manage your Opus service:
    systemctl --user status opus.service
    systemctl --user stop opus.service
    systemctl --user start opus.service
```

---

## 3. Alternatives Considered

### 3.1 Shell Script (`curl | bash`)

A single bash/sh script hosted at `get.opus.dev`. Rejected because:

- Does not work on Windows without WSL
- Harder to test across platforms in CI
- `curl | bash` is widely considered a security anti-pattern
- `npx` provides sandboxed, versioned execution with no permanent global install

### 3.2 Standalone Binary Installer (Go or Rust)

A pre-compiled installer binary distributed separately. Rejected because:

- Creates a bootstrapping problem: how does the user get the installer binary?
- Requires maintaining a separate build pipeline for the installer itself
- `npx` is universally available on developer machines and is the established convention for
  Node.js-adjacent tooling

### 3.3 npm Package with Dependencies (e.g. `inquirer`, `axios`, `tar`)

Using popular npm packages for prompts, HTTP, and archive extraction. Rejected because:

- Every runtime dependency is a supply-chain attack surface in an installer that runs with
  user permissions
- Node.js >= 24 provides all required primitives (`readline/promises`, `fetch`, `zlib`,
  `child_process`) without external packages
- Zero-dependency constraint is auditable and verifiable by any contributor reading the source

### 3.4 NSSM for Windows Service

Using NSSM (Non-Sucking Service Manager) as the Windows service manager. Deferred (not rejected)
because:

- NSSM provides a more robust Windows service experience with proper service lifecycle management
- However, NSSM requires a separate download, violating the zero-external-dependency constraint
  for the installer's service setup step
- Task Scheduler covers the MVP requirement; NSSM support will be addressed in a future ADR
  if Windows service reliability becomes a reported issue

### 3.5 Opt-in Service Setup (`--service` flag)

Making service setup opt-in rather than opt-out. Rejected because:

- Opus is marketed as a "24/7 autonomous assistant" — an installation that does not survive
  reboot contradicts this value proposition
- The majority of users want auto-start; power users who manage their own process supervisors
  are a minority and are well-served by `--no-service`
- Opt-out with a clear prompt (`Set up system service? [Y/n]`) provides transparency without
  burdening the majority

---

## 4. Consequences

### 4.1 Positive

- **Single-command installation** — `npx get-opus` takes a new user from zero to a running
  Opus instance with browser onboarding in under two minutes
- **Zero Node.js dependencies** — the installer's supply-chain attack surface is limited to
  Node.js itself; no third-party packages to audit or pin
- **Cross-platform** — Linux (x64/arm64), macOS (Intel/Apple Silicon), and Windows (x64)
  supported from day one
- **Auto-start by default** — systemd/launchd/Task Scheduler setup ensures Opus survives
  reboots without user intervention
- **Frictionless onboarding** — minimal questions + Dash wizard means users are not blocked
  by configuration complexity at install time
- **Idempotent detection** — existing installations are detected and the user is directed to
  the correct lifecycle command (`opus update`), preventing accidental overwrites
- **Checksum verification** — SHA-256 verification of all downloaded binaries provides
  integrity assurance before execution

### 4.2 Negative / Trade-offs

- **Node.js >= 24 requirement** — users on older Node.js versions cannot run `npx get-opus`
  without upgrading; this is a deliberate, documented hard requirement
- **No upgrade path in installer** — upgrade is delegated to `opus update` (future ADR);
  users who try to re-run `npx get-opus` on an existing installation are redirected with a
  clear message
- **Task Scheduler on Windows** — less robust than a true Windows Service (NSSM); an
  installer-level process crash does not auto-restart until next logon. Mitigated by
  `opus-server`'s own crash recovery; NSSM support deferred to future ADR
- **systemd linger requirement** — Linux users on multi-user systems may need
  `loginctl enable-linger` for boot-time auto-start; the installer runs this automatically
  but it requires the user to be logged in at least once post-install
- **GitHub API rate limiting** — fetching the latest release tag from the GitHub API is subject
  to unauthenticated rate limits (60 requests/hour per IP); for automated environments, users
  should pass `--version` explicitly to bypass the API call

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-003: Opus Dash Frontend Architecture](./ADR-003-dash-frontend-architecture.md)
- [ADR-012: Module System and Dependency Injection](./ADR-012-module-system-and-dependency-injection.md)
- [Node.js v24 Release Notes](https://nodejs.org/en/blog/release/v24.0.0)
- [GitHub Releases API](https://docs.github.com/en/rest/releases/releases)
- [systemd User Services](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [launchd Reference — Apple Developer](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html)
- [schtasks — Microsoft Docs](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/schtasks)
