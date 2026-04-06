<p align="center">
  <h1 align="center">Kernus Agent</h1>
  <p align="center">A lightweight, terminal-native infrastructure monitoring agent written in Go.</p>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#features">Features</a> ·
  <a href="#commands">Commands</a> ·
  <a href="#configuration">Configuration</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#building-from-source">Building</a>
</p>

---

Kernus is a command-line tool that sits on your servers and quietly watches over your Docker containers and host machines. It collects real-time metrics — CPU, memory, disk, container health, restart counts — and streams them to the [Kernus](https://kernus.app) platform, where you can visualize everything from a single dashboard.

It also ships with an interactive **TUI** (terminal user interface) so you can inspect containers, read logs, and manage workloads without ever leaving the terminal.

## How It Works

```
  Your Server                          Kernus Cloud
  ─────────────────────────────        ─────────────────────────
  │                           │        │                       │
  │  Docker daemon            │        │  API Server           │
  │    └─ containers ─┐       │        │    └─ ingest/metrics  │
  │                   ▼       │        │    └─ alerts engine   │
  │  kernus agent ────────────┼──────▶ │    └─ notification    │
  │    └─ host metrics        │  HTTPS │                       │
  │    └─ container stats     │        │  Dashboard            │
  │    └─ health checks       │        │    └─ real-time view  │
  │                           │        │    └─ alert rules     │
  └───────────────────────────┘        └───────────────────────┘
```

The agent runs as a **background process** and pushes metrics on a configurable interval. No inbound ports required — the server never needs to reach back into your infrastructure.

## Quick Start

```bash
# Install (Linux / macOS)
curl -sSL https://kernus.app/install | sh

# Windows (PowerShell)
irm https://kernus.app/install.ps1 | iex
```

Then connect it to your account:

```bash
# Authenticate
kernus login

# Create an agent token for this host
kernus token create "prod-server-01"

# Save the token and point it to the API
kernus token kn_live_a1b2c3... --server https://api.kernus.app --host prod-server-01

# Start collecting
kernus agent start
```

That's it. The agent will begin streaming Docker container metrics to your Kernus dashboard at a plan-appropriate interval.

## Features

- **Docker container monitoring** — tracks CPU usage, memory consumption, restart count, health status, network I/O, and storage mounts for every running container.
- **Host machine metrics** — collects CPU, RAM, and disk utilization from the underlying server via native OS APIs (Linux `/proc`, Windows `kernel32.dll`).
- **Interactive TUI** — a full terminal interface built with [tview](https://github.com/rivo/tview) and [tcell](https://github.com/gdamore/tcell). Browse containers, inspect stats, tail logs, and control workloads with keyboard shortcuts.
- **Resilient agent loop** — exponential backoff on failures, automatic Docker daemon reconnection, graceful shutdown on `SIGINT`/`SIGTERM`.
- **Plan-aware intervals** — the agent fetches its collection interval from the server, so plan upgrades take effect without restarting anything.
- **Flexible authentication** — JWT-based login with session persistence. Supports both interactive TUI login and headless flag-based login for CI environments.
- **Mock mode** — run the TUI with synthetic data for development and demos, no Docker daemon required.
- **Cross-platform** — works on Linux, macOS, and Windows.

## Commands

### `kernus agent start`

Start the background metrics agent. Reads configuration from `agent.conf` and environment variables, connects to the Docker daemon, and begins streaming container metrics to the server.

**Container scope:** by default the agent lists **only running** containers (same idea as `docker ps`). Stopped or exited containers from `docker ps -a` are **not** counted for plan preflight or sent as metrics. On hosts with many old stopped containers, this keeps counts aligned with what is actually running.

```bash
kernus agent start
```

Include stopped containers (previous behavior, counts everything in `docker ps -a`):

```bash
kernus agent start --all-containers
```

Only monitor stacks whose container names start with a prefix (useful when several projects share one Docker host):

```bash
kernus agent start --name-prefix wikig-main --name-prefix wikig-client
```

Environment equivalents: `KERNUS_AGENT_ALL_CONTAINERS=1`, `KERNUS_AGENT_NAME_PREFIX=wikig-main,wikig-client` (comma-separated).

| Flag | Description |
|------|-------------|
| `--all-containers` | List stopped/exited containers too (`docker ps -a`); they count toward plan limits. |
| `--name-prefix` | Repeatable; only containers whose name starts with one of these strings (after `/`) are included. |

### `kernus token create <label>`

Create a new agent token on the server. Requires an active login session.

```bash
kernus token create "prod-server-01"
```

### `kernus token <token> [flags]`

Save an agent token to local configuration along with server URL and host identity.

```bash
kernus token kn_live_a1b2c3... --server https://api.kernus.app --host prod-server-01 --interval 30
```

| Flag | Description | Default |
|------|-------------|---------|
| `--server` | Kernus API server URL | `https://api.kernus.app` |
| `--host` | Hostname reported to the server | System hostname |
| `--interval` | Collection interval in seconds | `30` |

### `kernus login`

Authenticate with the Kernus platform. Without flags, opens an interactive TUI login form. With flags, performs headless authentication suitable for scripts.

```bash
kernus login
kernus login --email user@example.com --password secret
```

### `kernus logout`

End the current session and remove stored credentials.

```bash
kernus logout
```

### `kernus profile`

Display information about the currently authenticated user.

```bash
kernus profile
kernus profile --json
```

### `kernus see`

Launch the interactive monitoring TUI. If no valid session exists, prompts for login first.

```bash
kernus see
kernus see --machines        # Switch to the machines panel
kernus see --refresh 5       # Set refresh interval to 5 seconds
kernus see --group backend   # Filter by group
kernus see --mock            # Use synthetic data (no Docker needed)
kernus see --docker-host tcp://192.168.1.10:2375
```

### `kernus send`

Continuously collect and send host-level metrics (CPU, RAM, disk) to the server.

```bash
kernus send --name "my-server" --group "backend" --interval 10
```

### `kernus config`

Persist local settings such as credentials and server URL.

```bash
kernus config --username user@example.com --password secret
kernus config --server https://api.kernus.app
```

## Configuration

Kernus stores its configuration in the user's config directory:

| Platform | Path |
|----------|------|
| Linux / macOS | `~/.config/kernus/` |
| Windows | `%APPDATA%\kernus\` |

Two configuration files are used:

- **`config.json`** — user credentials and server URL.
- **`agent.conf`** — agent token, server URL, hostname, and collection interval.

Session data is stored in `session.json` within the same directory.

### Environment Variables

Environment variables take precedence over file-based configuration:

| Variable | Description |
|----------|-------------|
| `KERNUS_SERVER_URL` | Override the API server URL |
| `KERNUS_AGENT_TOKEN` | Override the agent token |
| `KERNUS_HOST_NAME` | Override the reported hostname |
| `KERNUS_INTERVAL` | Override the collection interval (seconds) |

### Server URL Resolution

The server URL is resolved in the following order of priority:

1. CLI flag (`--server`)
2. Environment variable (`KERNUS_SERVER_URL`)
3. Agent config (`agent.conf`)
4. User config (`config.json`)
5. Default: `https://api.kernus.app`

## Architecture

### Agent (this repo)

```
  CLI Entry Points
  ─────────────────────────────────────────────────────────────────
  kernus login      kernus agent start    kernus see    kernus token
       │                   │                   │              │
       ▼                   ▼                   ▼              ▼
  ┌─────────────────────────────────────────────────────────────┐
  │                     cmd/  (Cobra)                           │
  │  login.go  agent.go  see.go  token.go  send.go  config.go  │
  └──────┬──────────┬──────────┬───────────────────────────────┘
         │          │          │
         ▼          ▼          ▼
  ┌──────────┐ ┌─────────────────────┐ ┌──────────────────────┐
  │  auth/   │ │     agent/          │ │       tui/           │
  │  client  │ │  collector.go  ─────┼─┤  app.go              │
  │  session │ │  sender.go     ─────┼─┤  login_app.go        │
  │  jwt     │ │  types.go           │ │  components/         │
  └──────────┘ │  mock_collector.go  │ └──────────────────────┘
               └──────┬──────────────┘
                      │
            ┌─────────┴──────────┐
            ▼                    ▼
  ┌──────────────────┐  ┌─────────────────┐
  │   docker/        │  │   metrics/      │
  │  client.go       │  │  collector.go   │
  │  (Docker SDK)    │  │  unix / windows │
  │  mock.go         │  └─────────────────┘
  └──────────────────┘

  config/  ──  config.go · agent.go · server.go  (layered resolution)
  models/  ──  container · machine · user
```

```
kernus/
├── main.go                     # Entry point
├── cmd/                        # CLI commands (Cobra)
│   ├── root.go                 # Root command and global config loading
│   ├── agent.go                # `agent start` — resilient metrics loop
│   ├── token.go                # `token create` / `token <value>` management
│   ├── login.go                # Interactive and headless authentication
│   ├── logout.go               # Session teardown
│   ├── profile.go              # User profile display
│   ├── config.go               # Local configuration persistence
│   ├── see.go                  # TUI launcher
│   └── send.go                 # Host metrics collection loop
├── internal/
│   ├── agent/                  # Core agent logic
│   │   ├── collector.go        # Docker metrics collection via Docker SDK
│   │   ├── sender.go           # HTTP client for /v1/ingest and /v1/agent/config
│   │   ├── types.go            # MetricCollector interface, IngestRequest, ContainerMetric
│   │   └── mock_collector.go   # Synthetic metrics for testing
│   ├── auth/                   # Authentication layer
│   │   ├── client.go           # AuthClient interface and local implementation
│   │   ├── http_client.go      # HTTP-based auth client for remote API
│   │   ├── jwt.go              # JWT token parsing
│   │   ├── session.go          # Session persistence and validation
│   │   └── mock.go             # Mock auth for development
│   ├── config/                 # Configuration management
│   │   ├── config.go           # User config (config.json)
│   │   ├── agent.go            # Agent config (agent.conf) with env overrides
│   │   └── server.go           # Server URL resolution chain
│   ├── docker/                 # Docker daemon interaction
│   │   ├── client.go           # Full Docker client: list, start, stop, stats, logs
│   │   └── mock.go             # Mock Docker client for offline use
│   ├── metrics/                # Host-level system metrics
│   │   ├── collector.go        # Platform-agnostic collector interface
│   │   ├── collector_unix.go   # Linux/macOS implementation (/proc)
│   │   └── collector_windows.go # Windows implementation (kernel32.dll)
│   ├── models/                 # Domain models
│   │   ├── container.go        # Container, stats, ports, networks, mounts
│   │   ├── machine.go          # Machine, groups, snapshots
│   │   └── user.go             # User and session models
│   └── tui/                    # Terminal user interface (tview + tcell)
│       ├── app.go              # Main TUI application
│       ├── login_app.go        # Interactive login form
│       └── components/         # Reusable TUI components
```

### Key Design Decisions

- **Interface-driven Docker access** — the `MetricCollector` interface abstracts away the Docker SDK, making it straightforward to swap in mock collectors for testing or offline development.
- **Layered configuration** — flags, environment variables, agent config, and user config form a clear resolution chain, so the same binary works seamlessly across development machines and production servers.
- **Resilient collection loop** — the agent tolerates transient Docker and network failures through exponential backoff and automatic reconnection, avoiding noisy restarts in production.

## Building from Source

```bash
git clone https://github.com/kqnd/kernus.git
cd kernus

# Build
make build

# Run the TUI
make run

# Run tests
make test
```

On Windows without `make`:

```bash
go build -o kernus.exe ./...
go run . see
go test ./... -v -race
```

### Requirements

- Go 1.25 or later
- Docker daemon (for container monitoring; not required for `--mock` mode)

## License

See [LICENSE](LICENSE) for details.
