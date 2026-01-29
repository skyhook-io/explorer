# Development Guide

Guide for developers contributing to Radar or building custom versions.

## Prerequisites

- **Go 1.22+**
- **Node.js 20+**
- **npm**
- **kubectl** with cluster access

## Quick Start

```bash
git clone https://github.com/skyhook-io/radar.git
cd radar

# Install dependencies
make deps

# Start development (two terminals)

# Terminal 1: Frontend with hot reload (port 9273)
make watch-frontend

# Terminal 2: Backend with hot reload (port 9280)
make watch-backend
```

Open http://localhost:9273 — the Vite dev server proxies `/api` requests to the Go backend.

## Make Commands

```bash
make build            # Build everything (frontend + embedded binary)
make frontend         # Build frontend only
make backend          # Build backend only
make test             # Run Go tests
make tsc              # TypeScript type check
make lint             # Run linter
make clean            # Clean build artifacts
make docker           # Build Docker image
```

## Project Structure

```
radar/
├── cmd/explorer/           # CLI entry point (main.go)
├── internal/
│   ├── k8s/               # Kubernetes client, informers, caching
│   ├── server/            # HTTP server, REST API, SSE, WebSocket
│   ├── topology/          # Graph construction and relationships
│   ├── helm/              # Helm SDK client and handlers
│   ├── traffic/           # Traffic visualization (Caretta, Hubble)
│   └── static/            # Embedded frontend (built from web/)
├── web/                    # React frontend
│   ├── src/
│   │   ├── api/           # API client, React Query hooks, SSE
│   │   ├── components/    # React components (topology, resources, helm, etc.)
│   │   ├── contexts/      # React contexts (capabilities, namespace, dock)
│   │   ├── types.ts       # TypeScript type definitions
│   │   └── utils/         # Topology layout and helpers
│   └── package.json
├── deploy/                 # Helm chart, Dockerfile
├── docs/                   # User documentation
└── scripts/                # Release scripts
```

## Architecture

### Backend (Go)

```
┌─────────────────────────────────────────────────────────────────┐
│                         Go Backend                              │
│                                                                 │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐   │
│   │   chi       │    │  Informers  │    │  SSE            │   │
│   │   Router    │───►│  (cached)   │───►│  Broadcaster    │   │
│   └─────────────┘    └─────────────┘    └─────────────────┘   │
│         │                   │                    │             │
│         ▼                   ▼                    ▼             │
│   REST API            K8s Watches         Real-time push      │
│   WebSocket (exec)    Resource cache      to browser          │
└─────────────────────────────────────────────────────────────────┘
```

**Key patterns:**
- **SharedInformers** — Watch-based caching, no polling. Resource changes arrive in milliseconds.
- **SSE Broadcaster** — Central hub for pushing real-time updates to all connected browsers.
- **Topology Builder** — Constructs a directed graph from cached resources on demand. Two modes: resources (hierarchy) and traffic (network flow).
- **Capabilities** — SelfSubjectAccessReview checks at startup to detect RBAC permissions. Resources that aren't accessible (e.g., secrets) are gracefully skipped.

### Frontend (React + TypeScript)

```
┌─────────────────────────────────────────────────────────────────┐
│                      React Frontend                             │
│                                                                 │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐   │
│   │  React      │    │  TanStack   │    │  @xyflow/react  │   │
│   │  Router     │───►│  Query      │───►│  + ELK.js       │   │
│   └─────────────┘    └─────────────┘    └─────────────────┘   │
│                             │                    │             │
│                             ▼                    ▼             │
│                      API + SSE hooks      Graph visualization  │
└─────────────────────────────────────────────────────────────────┘
```

**Key patterns:**
- **useEventSource** — SSE connection with automatic reconnection
- **React Query** — Server state management with caching and background refetching
- **CapabilitiesContext** — Fetches RBAC capabilities from `/api/capabilities` and hides unavailable features

### Tech Stack

**Backend:** Go 1.22+, client-go, chi router, gorilla/websocket, Helm SDK, `go:embed`

**Frontend:** React 18, TypeScript, Vite, @xyflow/react + ELK.js, @xterm/xterm, Monaco Editor, TanStack React Query v5, Tailwind CSS + shadcn/ui

## API Reference

### Core

| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | Health check with resource counts |
| `GET /api/cluster-info` | Cluster platform and version info |
| `GET /api/capabilities` | RBAC capability detection (exec, logs, port-forward, secrets) |
| `GET /api/topology` | Current topology graph (filterable by `?namespace=` and `?view=`) |
| `GET /api/namespaces` | List of namespaces |
| `GET /api/api-resources` | Available API resources (for CRD discovery) |

### Resources

| Endpoint | Description |
|----------|-------------|
| `GET /api/resources/{kind}` | List resources by kind |
| `GET /api/resources/{kind}/{ns}/{name}` | Get single resource with relationships |
| `PUT /api/resources/{kind}/{ns}/{name}` | Update resource from YAML |
| `DELETE /api/resources/{kind}/{ns}/{name}` | Delete resource |

### Events & History

| Endpoint | Description |
|----------|-------------|
| `GET /api/events` | Recent Kubernetes events |
| `GET /api/events/stream` | SSE stream for real-time events |
| `GET /api/changes` | Resource change history (`?namespace=`, `?kind=`, `?limit=`) |

### Pod Operations

| Endpoint | Description |
|----------|-------------|
| `GET /api/pods/{ns}/{name}/logs` | Fetch pod logs |
| `GET /api/pods/{ns}/{name}/logs/stream` | Stream logs via SSE |
| `GET /api/pods/{ns}/{name}/exec` | WebSocket terminal session |

### Port Forwarding

| Endpoint | Description |
|----------|-------------|
| `GET /api/portforwards` | List active sessions |
| `POST /api/portforwards` | Start port forward |
| `DELETE /api/portforwards/{id}` | Stop port forward |
| `GET /api/portforwards/available/{type}/{ns}/{name}` | Get available ports |

### Helm

| Endpoint | Description |
|----------|-------------|
| `GET /api/helm/releases` | List all releases |
| `GET /api/helm/releases/{ns}/{name}` | Release details |
| `GET /api/helm/releases/{ns}/{name}/values` | Release values |
| `GET /api/helm/releases/{ns}/{name}/manifest` | Rendered manifest |
| `GET /api/helm/releases/{ns}/{name}/diff` | Diff between revisions |
| `POST /api/helm/releases/{ns}/{name}/rollback` | Rollback release |
| `POST /api/helm/releases/{ns}/{name}/upgrade` | Upgrade release |
| `DELETE /api/helm/releases/{ns}/{name}` | Uninstall release |

## Adding Features

### New API Endpoint

1. Add route in `internal/server/server.go`:
   ```go
   r.Get("/api/my-endpoint", s.handleMyEndpoint)
   ```

2. Implement handler:
   ```go
   func (s *Server) handleMyEndpoint(w http.ResponseWriter, r *http.Request) {
       // ...
   }
   ```

### New Resource Type

1. Add informer in `internal/k8s/cache.go`
2. Add to topology builder in `internal/topology/builder.go`
3. Add TypeScript type in `web/src/types.ts`

### New UI Component

1. Create component in `web/src/components/`
2. Add route if needed in `web/src/App.tsx`
3. Add API hooks if needed in `web/src/api/`

## Testing

```bash
# Go tests
make test

# TypeScript type check
make tsc

# Manual testing (two terminals)
make watch-backend   # Terminal 1
make watch-frontend  # Terminal 2
```

## Releasing

```bash
# Interactive release (prompts for version and targets)
make release

# Or release specific components
make release-binaries     # CLI via goreleaser → GitHub Releases + Homebrew
make release-docker       # Docker image → GHCR
```

| Target | Command | Output |
|--------|---------|--------|
| CLI binaries | `make release-binaries` | GitHub Releases + Homebrew tap |
| Docker | `make release-docker` | `ghcr.io/skyhook-io/radar:VERSION` |
| All | `make release` | Interactive, choose targets |

### Prerequisites for Releasing

| Target | Requirements |
|--------|--------------|
| CLI binaries | `goreleaser`, `GITHUB_TOKEN` or `gh auth login` |
| Docker | Docker running, GHCR auth (`docker login ghcr.io`) |

### Release Checklist

1. Ensure tests pass: `make test`
2. Tag the release: `git tag v0.X.Y && git push origin v0.X.Y`
3. Run release: `make release`
4. Update Helm chart `appVersion` in `deploy/helm/radar/Chart.yaml`

## Code Style

- **Go:** `gofmt`, `golint`
- **TypeScript:** Prettier (`npm run format:write` in `web/`)
- **Commits:** Conventional commits preferred (`feat:`, `fix:`, `docs:`)
