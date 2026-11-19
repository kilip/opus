# Production Deployment Guide

This guide covers single-owner, self-hosted production deployment of **Opus**.

## Architecture Constraints

- **Single replica**: Opus uses embedded SQLite in WAL mode (`APP_DATABASE_URL`). Run **1 replica** only; do not scale horizontally behind a load balancer.
- **Persistent storage**: Mount persistent volumes for both the SQLite database file (and `-wal` / `-shm` sidecars) and `$APP_HOME` (`~/.opus` by default).

## Required Environment Variables

Generate a strong session signing secret (minimum 32 characters):

```bash
openssl rand -base64 48
```

Set these core production variables:

- `APP_BASE_URL`: Public HTTPS origin (for example `https://opus.example.com`). Enables HSTS, sets OAuth callback URLs, and validates WebSocket/CSRF origins.
- `APP_TRUSTED_ORIGINS`: Comma-separated additional allowed origins if accessed from multiple hostnames.
- `APP_TRUSTED_PROXIES`: Comma-separated reverse proxy IPs/CIDRs (for example `127.0.0.1/32,10.0.0.0/8`) so `X-Forwarded-For` and `X-Forwarded-Proto` are honored.
- `AUTH_SECRET`: Output of `openssl rand -base64 48`.
- `AUTH_COOKIE_SECURE`: Leave `auto` (sets `Secure` when `APP_BASE_URL` uses `https://`) or set `true`.
- `APP_SMTP_HOST`, `APP_SMTP_PORT`, `APP_SMTP_USERNAME`, `APP_SMTP_PASSWORD`, `APP_SMTP_FROM`: Required for email OTP delivery. If `APP_SMTP_HOST` is unset, Opus falls back to the console mailer (OTP codes are printed to server logs only).
- `AUTH_SETUP_TOKEN`: Optional pre-set first-run onboarding token. If unset, Opus generates one on first startup and prints `Opus setup token: ...` to stderr.
- `GH_TOKEN`: Fine-grained GitHub PAT with **Contents** (read/write), **Issues** (read/write), **Pull requests** (read/write), and **Metadata** (read) for vibe runs.

## Reverse Proxy & TLS

Your reverse proxy must:

1. Forward `Host`, `X-Forwarded-For`, and `X-Forwarded-Proto`.
2. Support WebSocket upgrades on `/api/nodes/connect`.
3. Disable response buffering on SSE streams (`/api/sessions/*/events`).
4. Forward `/mcp` untouched (no path rewriting or auth stripping).

### Caddy Example

```caddyfile
opus.example.com {
    reverse_proxy 127.0.0.1:7880 {
        flush_interval -1
    }
}
```

### Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name opus.example.com;

    # TLS configuration omitted

    location /api/nodes/connect {
        proxy_pass http://127.0.0.1:7880;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 3600s;
    }

    location ~ ^/api/sessions/[^/]+/events$ {
        proxy_pass http://127.0.0.1:7880;
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_cache off;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 3600s;
    }

    location / {
        proxy_pass http://127.0.0.1:7880;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## `/mcp` CSRF Exemption

Browser API routes under `/api/*` are protected by CSRF origin/header checks. `POST /mcp` is intentionally exempt from CSRF middleware because it is a machine-to-machine Streamable HTTP MCP endpoint authenticated by a run-scoped Bearer token (`mcp_token`) bound to `X-Opus-Run-ID` and that run's single project (and rejected once the run reaches a terminal state).

## Upgrades & Rollbacks

1. **Back up first**: Follow [Backup and Restore](backup.md) before upgrading.
2. **Schema version guard**: Opus stores the running binary's semver in the `meta` table. On startup, if the stored schema version is newer than the binary version, Opus refuses to start to prevent silent schema corruption.
3. **Rollback**: Restore the matching pre-upgrade database backup (recommended), or set `APP_ALLOW_DOWNGRADE=true` only if you have verified the schema is backwards-compatible.

## Pre-Flight Checklist

- [ ] `APP_BASE_URL` set to your public `https://` URL
- [ ] `AUTH_SECRET` set to a 32+ character random secret
- [ ] `APP_TRUSTED_PROXIES` configured if behind a reverse proxy
- [ ] SMTP (`APP_SMTP_HOST`, `APP_SMTP_FROM`) or OAuth (`AUTH_*`) configured for login
- [ ] `GH_TOKEN` configured with Contents, Issues, Pull requests (read/write) and Metadata (read)
- [ ] Reverse proxy configured with WebSocket upgrade (`/api/nodes/connect`) and SSE buffering disabled
- [ ] Single replica enforced and backup job scheduled (see [Backup and Restore](backup.md))
