# Development Guide

This guide covers setting up a development environment and contributing to Skyhook Explorer.

## Prerequisites

- **Go 1.22+** - [Download](https://go.dev/dl/)
- **Node.js 20+** - [Download](https://nodejs.org/)
- **npm** - Comes with Node.js
- **Make** - Usually pre-installed on macOS/Linux
- **Access to a Kubernetes cluster** - For testing

Recommended tools:
- [Air](https://github.com/cosmtrek/air) - Go hot reload (installed via `make watch-backend`)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) - Kubernetes CLI
- [kind](https://kind.sigs.k8s.io/) or [minikube](https://minikube.sigs.k8s.io/) - Local clusters

## Project Structure

```
explorer/
├── cmd/explorer/              # CLI entry point
│   └── main.go               # Main function, CLI flags
├── internal/
│   ├── helm/                 # Helm SDK integration
│   │   ├── client.go         # Helm operations wrapper
│   │   ├── handlers.go       # HTTP handlers
│   │   └── types.go          # Type definitions
│   ├── k8s/                  # Kubernetes client
│   │   ├── cache.go          # SharedInformer caching
│   │   ├── client.go         # Client initialization
│   │   ├── cluster_detection.go # Platform detection
│   │   ├── discovery.go      # API resource discovery
│   │   ├── dynamic_cache.go  # CRD support
│   │   ├── history.go        # Change history
│   │   └── update.go         # Resource mutations
│   ├── server/               # HTTP server
│   │   ├── server.go         # Chi router, REST handlers
│   │   ├── sse.go            # SSE broadcaster
│   │   ├── exec.go           # Pod exec WebSocket
│   │   ├── logs.go           # Pod logs handlers
│   │   └── portforward.go    # Port forward sessions
│   ├── static/               # Embedded frontend
│   │   └── embed.go
│   └── topology/             # Graph construction
│       ├── builder.go        # Node/edge creation
│       ├── relationships.go  # Resource relationships
│       └── types.go          # Type definitions
├── web/                      # React frontend
│   ├── src/
│   │   ├── api/              # API client, hooks
│   │   ├── components/       # React components
│   │   │   ├── dock/         # Bottom dock
│   │   │   ├── events/       # Events timeline
│   │   │   ├── helm/         # Helm UI
│   │   │   ├── logs/         # Logs viewer
│   │   │   ├── portforward/  # Port forward UI
│   │   │   ├── resources/    # Resource panels
│   │   │   ├── topology/     # Graph visualization
│   │   │   └── ui/           # Base components
│   │   ├── types.ts          # TypeScript types
│   │   └── utils/            # Utility functions
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── deploy/                   # Deployment configs
├── Makefile
└── go.mod
```

## Development Setup

### Clone and Install

```bash
git clone https://github.com/skyhook-io/explorer.git
cd explorer

# Install frontend dependencies
cd web && npm install && cd ..

# Verify Go modules
go mod download
```

### Running in Development Mode

Development mode runs the backend and frontend separately with hot reload:

**Terminal 1 - Backend (port 9280):**
```bash
make watch-backend
```

**Terminal 2 - Frontend (port 9273):**
```bash
make watch-frontend
```

Open http://localhost:9273 in your browser. The Vite dev server proxies `/api` requests to the backend.

### Alternative: Direct Go Run

```bash
# Build frontend first
cd web && npm run build && cd ..

# Run backend in dev mode (serves from web/dist)
go run ./cmd/explorer --dev --no-browser
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build frontend + embedded binary |
| `make frontend` | Build frontend only |
| `make backend` | Build backend only |
| `make watch-frontend` | Vite dev server with HMR |
| `make watch-backend` | Go with Air hot reload |
| `make test` | Run all tests |
| `make lint` | Run linters |
| `make docker` | Build Docker image |
| `make clean` | Remove build artifacts |

## Architecture Overview

### Backend Flow

1. **Startup** (`cmd/explorer/main.go`)
   - Parse CLI flags
   - Initialize Kubernetes client
   - Start informer caches
   - Initialize HTTP server
   - Open browser (optional)

2. **Kubernetes Caching** (`internal/k8s/cache.go`)
   - SharedInformers for typed resources
   - Dynamic informers for CRDs
   - Change notifications via channels
   - Field stripping for memory efficiency

3. **HTTP Server** (`internal/server/server.go`)
   - Chi router with middleware
   - REST handlers for resources
   - SSE broadcaster for real-time updates
   - WebSocket handler for pod exec

4. **Topology Building** (`internal/topology/builder.go`)
   - Constructs graph from cached resources
   - Determines edges via owner refs and selectors
   - Supports traffic and resources view modes

### Frontend Flow

1. **Entry** (`web/src/main.tsx`)
   - React app initialization
   - TanStack Query provider
   - React Router setup

2. **API Layer** (`web/src/api/`)
   - Typed API client
   - SSE connection management
   - React Query hooks

3. **Topology View** (`web/src/components/topology/`)
   - ReactFlow canvas
   - ELK.js layout
   - Custom node renderers

4. **Bottom Dock** (`web/src/components/dock/`)
   - Terminal sessions (xterm.js)
   - Logs viewer
   - Port forward management

## Adding New Features

### Adding a New API Endpoint

1. Add handler in `internal/server/server.go`:
```go
r.Get("/api/example", s.handleExample)
```

2. Implement handler:
```go
func (s *Server) handleExample(w http.ResponseWriter, r *http.Request) {
    // Implementation
    json.NewEncoder(w).Encode(response)
}
```

3. Add TypeScript types in `web/src/types.ts`

4. Add API client function in `web/src/api/client.ts`

5. Create React Query hook if needed

### Adding a New Resource Type

1. Add informer in `internal/k8s/cache.go`:
```go
case "newresources":
    informer = informerFactory.Apps().V1().NewResources().Informer()
```

2. Add to topology builder in `internal/topology/builder.go`

3. Add node type in `web/src/components/topology/`

4. Update TypeScript types

### Adding a New Frontend Component

1. Create component in `web/src/components/`
2. Use existing UI components from `components/ui/`
3. Follow existing patterns for API calls
4. Add to routing if it's a page

## Testing

### Backend Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/k8s/...

# Run with coverage
go test -cover ./...
```

### Frontend Tests

```bash
cd web

# Type checking
npm run tsc

# Lint
npm run lint
```

## Code Style

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Keep functions focused and small
- Handle errors explicitly

### TypeScript/React

- Use TypeScript strict mode
- Prefer functional components with hooks
- Use TanStack Query for server state
- Follow existing component patterns

## Pull Request Guidelines

1. **Fork and branch** from `main`
2. **Write tests** for new functionality
3. **Update documentation** if needed
4. **Run tests and linting** before submitting
5. **Keep PRs focused** - one feature/fix per PR
6. **Write clear commit messages**

### Commit Message Format

```
type: short description

Longer description if needed.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Debugging

### Backend

```bash
# Verbose logging
go run ./cmd/explorer --dev 2>&1 | tee debug.log

# With delve debugger
dlv debug ./cmd/explorer -- --dev --no-browser
```

### Frontend

- Use browser DevTools
- React Developer Tools extension
- TanStack Query Devtools (included in dev mode)

### Kubernetes

```bash
# Check what Explorer sees
curl http://localhost:9280/api/health
curl http://localhost:9280/api/topology | jq .

# Compare with kubectl
kubectl get pods -A
kubectl get events -A
```

## Building Releases

Releases are built via GitHub Actions using GoReleaser:

```bash
# Local build (for testing)
goreleaser build --snapshot --clean

# Full release (tags only)
git tag v1.2.3
git push origin v1.2.3
```

The CI pipeline builds binaries for:
- darwin/amd64, darwin/arm64
- linux/amd64, linux/arm64
- windows/amd64

## Getting Help

- [GitHub Issues](https://github.com/skyhook-io/explorer/issues) - Bug reports
- [GitHub Discussions](https://github.com/skyhook-io/explorer/discussions) - Questions
- [Contributing Guide](../CONTRIBUTING.md) - Contribution process
