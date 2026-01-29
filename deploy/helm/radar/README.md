# Radar Helm Chart

Deploy Radar to your Kubernetes cluster for web-based cluster visualization and management.

> **See also:** [In-Cluster Deployment Guide](../../../docs/in-cluster.md) for ingress and DNS setup.

## Prerequisites

- Kubernetes 1.21+
- Helm 3.0+

## Installation

### Quick Start

```bash
helm install radar ./deploy/helm/radar \
  --namespace radar \
  --create-namespace
```

Access via port-forward:
```bash
kubectl port-forward svc/radar 9280:9280 -n radar
open http://localhost:9280
```

### With Ingress

```bash
helm install radar ./deploy/helm/radar \
  --namespace radar \
  --create-namespace \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=radar.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

### With TLS

```bash
helm install radar ./deploy/helm/radar \
  --namespace radar \
  --create-namespace \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=radar.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set ingress.tls[0].secretName=radar-tls \
  --set ingress.tls[0].hosts[0]=radar.example.com
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/skyhook-io/radar` |
| `image.tag` | Image tag | Chart appVersion |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `9280` |
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `timeline.storage` | Timeline storage (memory/sqlite) | `memory` |
| `persistence.enabled` | Enable PVC for SQLite | `false` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.memory` | Memory request | `128Mi` |

See `values.yaml` for all configuration options.

## RBAC

The chart creates a ClusterRole with read-only access to common Kubernetes resources.

**Default permissions (always enabled):**
- Core: pods, services, configmaps, events, namespaces, nodes, pvcs, serviceaccounts, endpoints
- Apps: deployments, daemonsets, statefulsets, replicasets
- Networking: ingresses, networkpolicies
- Batch: jobs, cronjobs
- Autoscaling: horizontalpodautoscalers
- CRDs: Argo, Knative, cert-manager, Gateway API

**Opt-in permissions (disabled by default for security):**

| Feature | Value | Description |
|---------|-------|-------------|
| Secrets | `rbac.secrets: true` | View secrets in resource list |
| Terminal | `rbac.podExec: true` | Shell access to pods |
| Port Forward | `rbac.portForward: true` | Port forwarding to pods |
| Logs | `rbac.podLogs: true` | View pod logs (enabled by default) |

Radar uses its ServiceAccount permissions to access the Kubernetes API. The UI automatically detects which features are available based on RBAC and hides unavailable features.

## Upgrading

```bash
helm upgrade radar ./deploy/helm/radar -n radar
```

## Uninstalling

```bash
helm uninstall radar -n radar
kubectl delete namespace radar
```
