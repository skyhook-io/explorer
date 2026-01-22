# Getting Started

This guide will help you install and run Skyhook Explorer for the first time.

## Installation

### Option 1: kubectl Plugin (Krew)

If you have [Krew](https://krew.sigs.k8s.io/) installed:

```bash
kubectl krew install explorer
kubectl explorer
```

### Option 2: Homebrew (macOS)

```bash
brew install skyhook-io/tap/explorer
skyhook-explorer
```

### Option 3: Direct Download

Download the appropriate binary from [GitHub Releases](https://github.com/skyhook-io/explorer/releases):

- `explorer-darwin-amd64` - macOS Intel
- `explorer-darwin-arm64` - macOS Apple Silicon
- `explorer-linux-amd64` - Linux x86_64
- `explorer-linux-arm64` - Linux ARM64
- `explorer-windows-amd64.exe` - Windows

Make it executable and run:

```bash
chmod +x explorer-*
./explorer-darwin-arm64  # or your platform
```

### Option 4: Docker

```bash
docker run -v ~/.kube:/root/.kube -p 9280:9280 ghcr.io/skyhook-io/explorer
```

Then open http://localhost:9280 in your browser.

### Option 5: Build from Source

```bash
git clone https://github.com/skyhook-io/explorer.git
cd explorer
make build
./explorer
```

## First Run

1. Ensure you have a valid kubeconfig (usually at `~/.kube/config`)
2. Run the explorer:
   ```bash
   skyhook-explorer
   ```
3. Your browser will open automatically to http://localhost:9280
4. You'll see the topology view of your current Kubernetes context

## Basic Usage

### View Different Namespaces

Use the namespace dropdown in the header to filter by namespace, or select "All Namespaces" to see everything.

### Switch View Modes

Toggle between:
- **Traffic View** - Shows network flow: Ingress → Service → Pod
- **Resources View** - Shows ownership hierarchy: Deployment → ReplicaSet → Pod

### Inspect Resources

Click any node in the topology to open the resource detail panel showing:
- Resource metadata
- Status and conditions
- Related resources
- Full YAML (editable)

### Open Pod Terminal

1. Click on a Pod node
2. Click the "Terminal" button in the detail panel
3. Select a container (if multiple)
4. An interactive terminal opens in the bottom dock

### Stream Pod Logs

1. Click on a Pod node
2. Click the "Logs" button
3. Select a container and configure options
4. Logs stream in real-time in the bottom dock

## CLI Options

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `--namespace` | _(all)_ | Initial namespace filter |
| `--port` | `9280` | Server port |
| `--no-browser` | `false` | Don't auto-open browser |
| `--persist-history` | `false` | Persist change history to file |
| `--history-limit` | `1000` | Maximum changes to retain |
| `--version` | | Show version and exit |

## Examples

```bash
# Start with a specific namespace
skyhook-explorer --namespace production

# Use a different kubeconfig
skyhook-explorer --kubeconfig ~/.kube/staging-config

# Run on a different port without opening browser
skyhook-explorer --port 8080 --no-browser

# Enable persistent change history
skyhook-explorer --persist-history --history-limit 5000
```

## Next Steps

- Read the [User Guide](User-Guide) for detailed feature documentation
- Check the [API Reference](API-Reference) for programmatic access
- See [Development](Development) to contribute
