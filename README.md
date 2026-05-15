# Opus

Self-hosted AI agent platform for modern workflows.

Opus is a powerful, self-hosted AI agent platform designed to give you full control over your AI interactions. It consists of a robust Go backend and a sleek Next.js 16 dashboard.

## Quick Start

### 1. npx get-opus (Recommended)
The easiest way to get started as an end user.
```bash
npx get-opus
```

### 2. Docker Compose
For technical users who prefer containerized deployments.
```bash
docker compose up -d
```

### 3. Bare Metal
Download the binary for your platform from the [Releases](https://github.com/kilip/opus/releases) page.
```bash
./opus start
```

## Developer Setup

Prerequisites: Go 1.23+, Node.js 22+, pnpm, and Task.

```bash
# Install all dependencies
task setup

# Start development servers (API & Dashboard)
task dev
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `start` | Start the Opus server |
| `stop` | Stop the Opus server |
| `restart`| Restart the Opus server |
| `status` | Check server status |
| `logs`   | View server logs |

## Configuration

Opus can be configured via environment variables or a config file. See [.env.example](.env.example) for a full list of variables.
By default, Opus looks for configuration in `~/.opus/config.toml`.

## Architecture

Opus uses a decoupled architecture with a Go API and a Next.js frontend. For more details, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## License

MIT
