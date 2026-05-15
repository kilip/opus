# Phase 4 — Integration & Deployment

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Draft  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Phase Goal

Wire all components together and validate end-to-end flows. Build the `npx get-opus` wizard, finalize Docker images, configure system service registration, and complete CI/CD release pipelines. The output is a fully deployable, installable Opus v1.0.1.

---

## Prerequisites

- Phase 1 is complete.
- Phase 2 is complete (API passing all tests, binary builds successfully).
- Phase 3 is complete (Frontend passing all tests, PWA build succeeds).

---

## Context for AI Agent

> Always provide `docs/CONVENTIONS.md`, `docs/PRD.md`, and `docs/ARCHITECTURE.md` alongside this file.

**You are wiring together the Opus API and dash, building the distribution layer, and validating everything end-to-end.** Follow the Architecture document for all deployment specifications. The `npx get-opus` wizard is a separate Node.js package in `installer/`.

---

## P4-T1 — Validate End-to-End Auth Flow

### What to Do

Validate the complete OAuth2 authentication flow end-to-end with both the API and dash running concurrently. This is a manual validation task with a defined checklist.

### Setup

```bash
# Terminal 1 — Start API
cd api/
OPUS_SERVER_ENV=development OPUS_AUTH_SECRET=test-secret-32-chars-minimum task dev

# Terminal 2 — Start Dash
cd dash/
NEXT_PUBLIC_API_URL=http://localhost:8080 task dev
```

### Validation Steps

#### Google OAuth2 Flow

1. Open `http://localhost:3000`.
2. Verify redirect to `/login`.
3. Click "Sign in with Google".
4. Verify redirect to `http://localhost:8080/auth/google`.
5. Complete Google OAuth2 (requires valid `OPUS_AUTH_GOOGLE_CLIENT_ID`).
6. Verify redirect back to dash at `http://localhost:3000`.
7. Verify user name and avatar are displayed.
8. Open browser DevTools → Application → Cookies.
9. Verify `refresh_token` cookie exists with `HttpOnly: true` and `SameSite: Strict`.
10. Verify no access token in cookies (it should be in memory only).

#### Token Refresh Flow

1. Wait for access token to expire (or manually test `POST /auth/refresh`).
2. Verify `AuthGuard` silently refreshes the token.
3. Verify the old `refresh_token` cookie is replaced with a new one.
4. Verify the user remains on the dash.

#### Logout Flow

1. Click the logout button on the dash.
2. Verify `POST /auth/logout` is called.
3. Verify `refresh_token` cookie is cleared.
4. Verify redirect to `/login`.
5. Verify visiting `/` redirects to `/login` (session is fully cleared).

#### Email/Password Flow (Development Only)

1. With `OPUS_SERVER_ENV=development`, submit the email/password form.
2. Verify login succeeds.
3. With `OPUS_SERVER_ENV=production`, verify `POST /auth/login` returns `403 Forbidden`.

### Acceptance Criteria

- [x] Google OAuth2 flow completes without errors.
- [x] `refresh_token` cookie is `HttpOnly`, `SameSite: Strict`.
- [x] Access token is not stored in cookies or `localStorage`.
- [x] Silent token refresh works without user interaction.
- [x] Logout clears session and redirects to login.
- [x] Revisiting `/` after logout redirects to `/login`.
- [x] Email/password login returns 403 in production mode.

---

## P4-T2 — Validate SSE Streaming End-to-End

### What to Do

Validate the SSE streaming connection end-to-end with both services running.

### Validation Steps

1. Log in via OAuth2 or email/password (development mode).
2. Navigate to the dash.
3. Click the "Connect" button.
4. Verify `StreamOutput` component shows `isConnected: true` (green indicator).
5. Verify heartbeat events appear in the output every ~30 seconds.
6. Open browser DevTools → Network tab → Filter by `EventStream`.
7. Verify the `/stream` request shows `text/event-stream` content type.
8. Disconnect from network and verify `error` state appears in `StreamOutput`.
9. Reconnect and click "Connect" again — verify reconnection works.

### Acceptance Criteria

- [x] SSE connection establishes successfully after login.
- [x] `StreamOutput` shows green indicator when connected.
- [x] Heartbeat events appear in output every ~30 seconds.
- [x] Network tab shows `text/event-stream` content type for stream request.
- [x] Error state displays correctly on connection loss.
- [x] Reconnection works after connection loss.

---

## P4-T3 — Implement `npx get-opus` Wizard

### What to Do

Create the Node.js installer package that provides the `npx get-opus` experience. This is a separate package in `installer/` at the monorepo root.

### Directory Structure

```
installer/
├── src/
│   ├── index.ts          # Entry point — calls install()
│   ├── install.ts        # Main install wizard
│   ├── platform.ts       # Platform detection (OS + arch)
│   ├── download.ts       # Binary download from GitHub Releases
│   ├── config.ts         # Generate ~/.opus/config.toml
│   └── service.ts        # Register system service
├── package.json
└── tsconfig.json
```

### `package.json` for Installer

```json
{
  "name": "get-opus",
  "version": "1.0.1",
  "description": "Installer for Opus AI Agent",
  "bin": {
    "opus": "./dist/index.js"
  },
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "dependencies": {
    "inquirer": "^9.0.0",
    "chalk": "^5.0.0",
    "ora": "^7.0.0",
    "node-fetch": "^3.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "@types/node": "^22.0.0",
    "@types/inquirer": "^9.0.0"
  }
}
```

### `install.ts` — Wizard Flow

Implement the interactive wizard with the following steps:

```
[1/5] Detecting platform...
[2/5] Downloading binary from GitHub Releases...
[3/5] Configuring...

  Prompts:
  ? Database driver:           › sqlite / postgres
  ? Database path (sqlite):    › ~/.opus/opus.db
  ? Server port:               › 8080
  ? Google Client ID:          › (leave blank to skip)
  ? Google Client Secret:      › (leave blank to skip)
  ? GitHub Client ID:          › (leave blank to skip)
  ? GitHub Client Secret:      › (leave blank to skip)
  ? JWT Secret:                › (auto-generated if blank)

[4/5] Writing ~/.opus/config.toml...
[5/5] Installing system service...

  ✓ Opus is installed and running!
  ✓ Open http://localhost:<port> to get started.
  Manage with: opus start | stop | restart | status | logs
```

### `platform.ts`

Detect OS and architecture to select the correct binary from GitHub Releases:

| `process.platform` | `process.arch` | Binary Name |
|--------------------|---------------|-------------|
| `linux` | `x64` | `opus-linux-amd64` |
| `linux` | `arm64` | `opus-linux-arm64` |
| `darwin` | `x64` | `opus-darwin-amd64` |
| `darwin` | `arm64` | `opus-darwin-arm64` |
| `win32` | `x64` | `opus-windows-amd64.exe` |

### `download.ts`

Download the binary from:
```
https://github.com/kilip/opus/opus/releases/latest/download/<binary-name>
```

Save to `/usr/local/bin/opus` (Linux/macOS) or `%LOCALAPPDATA%\opus\opus.exe` (Windows).
Make executable (`chmod +x`) on Linux/macOS.

### `config.ts`

Generate `~/.opus/config.toml` from wizard answers. Auto-generate a 32-byte random JWT secret using `crypto.randomBytes(32).toString('hex')` if the user leaves it blank.

### `service.ts`

Register the system service based on platform:

| Platform | Service Manager | Action |
|----------|----------------|--------|
| Linux | systemd | Write `/etc/systemd/system/opus.service`, run `systemctl enable --now opus` |
| macOS | launchd | Write `~/Library/LaunchAgents/com.opus.agent.plist`, run `launchctl load` |
| Windows | Windows Service | Run `sc.exe create Opus binPath= "..."` |

### Constraints

- Use `inquirer` for interactive prompts.
- Use `ora` for spinners.
- Use `chalk` for colored output.
- Auto-generate JWT secret if user leaves blank — never use a weak default.
- Handle unsupported platform gracefully with a clear error message.

### Acceptance Criteria

- [x] Directory `installer/` exists at monorepo root.
- [x] File `installer/package.json` exists with `bin.opus` pointing to `dist/index.js`.
- [x] File `installer/src/install.ts` implements the 5-step wizard.
- [x] File `installer/src/platform.ts` detects OS and arch correctly.
- [x] File `installer/src/download.ts` downloads from GitHub Releases.
- [x] File `installer/src/config.ts` generates `~/.opus/config.toml`.
- [x] File `installer/src/service.ts` registers system service for Linux, macOS, Windows.
- [x] JWT secret is auto-generated if blank (32 bytes, hex-encoded).
- [x] Running `node dist/index.js` starts the interactive wizard.
- [x] Unsupported platforms show a clear error message.

---

## P4-T4 — Implement `api/Dockerfile` and `dash/Dockerfile`

### What to Do

Create production Dockerfiles for both components using multi-stage builds.

### `api/Dockerfile`

```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /opus ./cmd/opus

# Stage 2: Runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /opus /usr/local/bin/opus

EXPOSE 8080
CMD ["opus", "start"]
```

### `dash/Dockerfile`

```dockerfile
# Stage 1: Dependencies
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile

# Stage 2: Build
FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm install -g pnpm && pnpm build

# Stage 3: Runtime
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000
CMD ["node", "server.js"]
```

### Next.js Standalone Output

Add to `dash/next.config.ts`:

```ts
export default withSerwistConfig({
  output: "standalone",
  // ...
});
```

### Constraints

- Use multi-stage builds to minimize image size.
- `api/Dockerfile` must include `sqlite-libs` for SQLite support.
- `dash/Dockerfile` must use `output: "standalone"` Next.js configuration.
- `CGO_ENABLED=1` is required for the `go-sqlite3` driver.

### Acceptance Criteria

- [x] File `api/Dockerfile` exists with multi-stage build.
- [x] File `dash/Dockerfile` exists with multi-stage build.
- [x] `api/Dockerfile` includes `sqlite-libs` in runtime stage.
- [x] `dash/next.config.ts` has `output: "standalone"`.
- [x] Running `docker build -t opus-api ./api` succeeds.
- [x] Running `docker build -t opus-dash ./dash` succeeds.
- [x] Running `docker run -p 8080:8080 opus-api` starts the API server.
- [x] Running `docker run -p 3000:3000 opus-dash` starts the dash.

---

## P4-T5 — Finalize `docker-compose.yml` and `docker-compose.dev.yml`

### What to Do

Update both Docker Compose files to reference the local Dockerfiles (for development builds) and add health checks.

### Updates to `docker-compose.yml`

Add a health check for the `api` service:

```yaml
api:
  healthcheck:
    test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
    interval: 30s
    timeout: 10s
    retries: 3
```

Add `dash` depends on `api` with `condition: service_healthy`.

### Updates to `docker-compose.dev.yml`

Ensure both services use `build.context` pointing to their respective directories.

### Validation

```bash
# Validate production compose
docker compose config

# Validate dev compose
docker compose -f docker-compose.yml -f docker-compose.dev.yml config

# Run full stack in development
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

### Acceptance Criteria

- [x] `api` service in `docker-compose.yml` has a health check targeting `/health`.
- [x] `dash` depends on `api` with `condition: service_healthy`.
- [x] Both compose files pass `docker compose config` validation.
- [x] Running the full stack via Docker Compose starts both services successfully.
- [x] `GET http://localhost:8080/health` returns `{"success": true}` when running via Docker.
- [x] `http://localhost:3000` serves the dash when running via Docker.

---

## P4-T6 — Configure systemd / launchd / Windows Service Registration

### What to Do

Create the system service definition files that the `npx get-opus` wizard deploys. These files are templates — the installer substitutes variables at install time.

### Files to Create

**`installer/templates/opus.service`** (Linux systemd)

```ini
[Unit]
Description=Opus AI Agent
Documentation=https://github.com/kilip/opus/opus
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/opus start
Restart=always
RestartSec=5
User={{USER}}
Environment=HOME=/home/{{USER}}
Environment=OPUS_SERVER_ENV=production

[Install]
WantedBy=multi-user.target
```

**`installer/templates/com.opus.agent.plist`** (macOS launchd)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.opus.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/opus</string>
    <string>start</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>{{HOME}}/.opus/logs/opus.log</string>
  <key>StandardErrorPath</key>
  <string>{{HOME}}/.opus/logs/opus.error.log</string>
</dict>
</plist>
```

### `service.ts` Template Substitution

The `service.ts` installer module must:
1. Read the template file.
2. Substitute `{{USER}}` and `{{HOME}}` with actual values from `process.env`.
3. Write the substituted file to the correct system path.
4. Run the appropriate enable/start command.

### Acceptance Criteria

- [x] File `installer/templates/opus.service` exists.
- [x] File `installer/templates/com.opus.agent.plist` exists.
- [x] `service.ts` reads templates, substitutes variables, and writes to system path.
- [x] Linux: service is registered with `systemctl enable --now opus`.
- [x] macOS: plist is loaded with `launchctl load`.
- [x] Windows: service is created with `sc.exe create`.

---

## P4-T7 — Finalize GitHub Actions `ci.yml`

### What to Do

Update the CI workflow to cache Go modules and Node.js dependencies for faster runs.

### Updates

Add caching steps:

```yaml
- name: Cache Go modules
  uses: actions/cache@v4
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('api/go.sum') }}

- name: Cache pnpm store
  uses: actions/cache@v4
  with:
    path: ~/.pnpm-store
    key: ${{ runner.os }}-pnpm-${{ hashFiles('dash/pnpm-lock.yaml') }}
```

### Acceptance Criteria

- [x] `ci.yml` includes Go module cache step.
- [x] `ci.yml` includes pnpm store cache step.
- [x] CI run time improves by at least 30% after caching.
- [x] All CI steps still pass with caching enabled.

---

## P4-T8 — Finalize GitHub Actions `build.yml`

### What to Do

Update the build workflow to use Docker Buildx for multi-platform image builds (linux/amd64 and linux/arm64).

### Updates

```yaml
- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and Push API Image
  uses: docker/build-push-action@v5
  with:
    context: ./api
    platforms: linux/amd64,linux/arm64
    push: true
    tags: |
      ghcr.io/opus/opus-api:latest
      ghcr.io/opus/opus-api:${{ github.sha }}
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

Apply the same multi-platform configuration to the `dash` image build.

### Acceptance Criteria

- [x] `build.yml` uses `docker/setup-buildx-action@v3`.
- [x] API image is built for `linux/amd64` and `linux/arm64`.
- [x] Dash image is built for `linux/amd64` and `linux/arm64`.
- [x] Images are tagged with both `latest` and `${{ github.sha }}`.
- [x] Docker layer caching is configured with `type=gha`.

---

## P4-T9 — Finalize GitHub Actions `release.yml`

### What to Do

Finalize the release workflow to include the installer npm package publication and validate the cross-compile step.

### Validation of Cross-Compile Step

Ensure the binary names match exactly what the installer's `download.ts` expects:

| OS | Arch | Binary Name |
|----|------|-------------|
| linux | amd64 | `opus-linux-amd64` |
| linux | arm64 | `opus-linux-arm64` |
| darwin | amd64 | `opus-darwin-amd64` |
| darwin | arm64 | `opus-darwin-arm64` |
| windows | amd64 | `opus-windows-amd64.exe` |

### npm Package Publication Step

```yaml
- name: Build Installer
  run: |
    cd installer
    npm install
    npm run build

- name: Publish npm Package
  run: npm publish
  working-directory: ./installer
  env:
    NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### Acceptance Criteria

- [x] `release.yml` cross-compiles for all 5 target platforms.
- [x] Binary names exactly match the installer's `download.ts` expectations.
- [x] `release.yml` builds the installer package before publishing.
- [x] `release.yml` publishes `installer/` to npm with the tag version.
- [x] GitHub Release is created with all 5 binaries attached.
- [x] `NPM_TOKEN` is referenced from GitHub Secrets — not hardcoded.

---

## P4-T10 — Lighthouse PWA Audit

### What to Do

Run a Lighthouse audit against the production build and verify the PWA score meets the target of ≥ 90.

### Setup

```bash
# Build production dash
cd dash/
pnpm build
pnpm start &

# Install Lighthouse CLI
npm install -g lighthouse

# Run audit
lighthouse http://localhost:3000 \
  --output=json \
  --output=html \
  --output-path=./lighthouse-report \
  --only-categories=performance,accessibility,best-practices,seo,pwa
```

### PWA Checklist to Verify

| Criterion | Expected |
|-----------|---------|
| Has a `<meta name="viewport">` tag | ✓ |
| Has a valid Web App Manifest | ✓ |
| Has a registered Service Worker | ✓ |
| Responds with 200 when offline | ✓ |
| Has appropriate `theme-color` | ✓ |
| Icons are 192x192 and 512x512 | ✓ |
| Is served over HTTPS (production) | ✓ |

### Remediation

If the PWA score is below 90, address the failing criteria in this order:
1. Service Worker registration issues → check `sw.ts` configuration.
2. Manifest issues → check `manifest.webmanifest`.
3. Offline support → verify `/offline` page is cached by Serwist.
4. Performance issues → check image optimization, bundle size.

### Acceptance Criteria

- [x] Lighthouse audit runs against production build without errors.
- [x] PWA score is ≥ 90.
- [x] All 7 PWA checklist items pass.
- [x] `lighthouse-report.html` is generated and saved.
- [x] Any failures below 90 are documented with remediation steps applied.

---

## P4-T11 — Final Cross-Platform Binary Build Validation

### What to Do

Validate that the cross-compiled binaries for all 5 target platforms are functional.

### Validation Script

```bash
#!/bin/bash
# Run from api/ directory

TARGETS=(
  "linux/amd64/opus-linux-amd64"
  "linux/arm64/opus-linux-arm64"
  "darwin/amd64/opus-darwin-amd64"
  "darwin/arm64/opus-darwin-arm64"
  "windows/amd64/opus-windows-amd64.exe"
)

mkdir -p dist

for TARGET in "${TARGETS[@]}"; do
  IFS='/' read -r OS ARCH BINARY <<< "$TARGET"
  echo "Building $BINARY..."
  CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -o dist/$BINARY ./cmd/opus
  if [ $? -eq 0 ]; then
    echo "✓ $BINARY"
    ls -lh dist/$BINARY
  else
    echo "✗ $BINARY FAILED"
    exit 1
  fi
done

echo ""
echo "All binaries built successfully."
ls -lh dist/
```

### Note on CGO and SQLite

For cross-compilation, `CGO_ENABLED=0` disables CGO which is required for `go-sqlite3`. For distribution binaries that need SQLite support, use a pure-Go SQLite implementation:

```bash
go get modernc.org/sqlite
```

Update `internal/config/database.go` to use `modernc.org/sqlite` instead of `github.com/mattn/go-sqlite3` for the distributed binary. The `go-sqlite3` driver (CGO) can still be used for local development.

### Acceptance Criteria

- [x] All 5 platform binaries build without errors.
- [x] Binary sizes are reasonable (under 50MB each).
- [x] `opus-linux-amd64` can be executed on a Linux amd64 machine.
- [x] `opus-darwin-arm64` can be executed on an Apple Silicon Mac.
- [x] `opus --help` outputs the Cobra command list on supported platforms.
- [x] Binary names exactly match the installer's `download.ts` expectations.

---

## Phase 4 Completion Checklist

This is the final phase. Before marking the project as complete, verify:

- [x] All 11 tasks (P4-T1 through P4-T11) are marked complete.
- [x] End-to-end Google OAuth2 flow works without errors.
- [x] End-to-end SSE streaming works without errors.
- [x] `npx get-opus` wizard runs and produces a working installation.
- [x] `docker compose up` starts both services and health checks pass.
- [x] Lighthouse PWA score is ≥ 90.
- [x] All 5 platform binaries build successfully.
- [x] GitHub Actions CI passes on a test pull request.
- [x] GitHub Actions Build passes on a merge to `main`.
- [x] GitHub Release is created with all binaries and npm package on a test tag.
- [x] No secrets are hardcoded anywhere in the codebase.
- [x] No `.env` files are committed (only `.env.example`).

---

## Final Project Validation

After all phases are complete, perform this final validation:

### Installation Path 1: `npx get-opus`

```bash
npx get-opus
# Follow interactive prompts
# Verify service starts
opus status
# Open browser to http://localhost:8080
```

### Installation Path 2: Docker Compose

```bash
cp .env.example .env
# Fill in required values in .env
docker compose up -d
# Verify health
curl http://localhost:8080/health
# Open browser to http://localhost:3000
```

### Installation Path 3: Bare Metal Binary

```bash
curl -L https://github.com/kilip/opus/opus/releases/latest/download/opus-linux-amd64 -o opus
chmod +x opus
sudo mv opus /usr/local/bin/
opus init
opus start
```

### Developer Setup

```bash
git clone https://github.com/kilip/opus/opus
cd opus
cp .env.example .env
task setup
task dev
# API: http://localhost:8080
# Dash: http://localhost:3000
```

All three installation paths must succeed for the project to be considered complete.
