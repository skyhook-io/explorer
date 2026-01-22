# Skyhook Explorer Wiki

Welcome to the Skyhook Explorer documentation. Explorer is a real-time Kubernetes cluster visualization and management tool.

## Quick Links

- [Getting Started](Getting-Started) - Installation and your first run
- [User Guide](User-Guide) - Complete guide to the web interface
- [API Reference](API-Reference) - REST API documentation
- [Development](Development) - Contributing and local development

## What is Skyhook Explorer?

Skyhook Explorer is an open-source tool that provides:

- **Visual topology** of your Kubernetes cluster resources
- **Real-time updates** via Server-Sent Events
- **Pod terminal access** directly in your browser
- **Log streaming** with container selection
- **Port forwarding** management
- **Helm release** inspection and management
- **Resource editing** with YAML support

## How It Works

Explorer runs as a single binary on your local machine. It:

1. Connects to your Kubernetes cluster using your kubeconfig
2. Uses SharedInformers to efficiently watch cluster resources
3. Serves a React web UI embedded in the binary
4. Pushes real-time updates to the browser via SSE
5. Provides WebSocket connections for terminal sessions

No agents or modifications to your cluster are required.

## Requirements

- Access to a Kubernetes cluster (kubeconfig)
- A modern web browser

## Support

- [GitHub Issues](https://github.com/skyhook-io/explorer/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/skyhook-io/explorer/discussions) - Questions and community
