# API Reference

Skyhook Explorer exposes a REST API for programmatic access. All endpoints are available at `http://localhost:9280/api` (or your configured port).

## Authentication

The API uses your local kubeconfig for Kubernetes authentication. No additional authentication is required for the Explorer API itself.

## Response Format

All responses are JSON unless otherwise noted. Errors return:

```json
{
  "error": "Error message description"
}
```

---

## Core Endpoints

### Health Check

```
GET /api/health
```

Returns server health and resource counts.

**Response:**
```json
{
  "status": "ok",
  "resources": {
    "pods": 42,
    "deployments": 12,
    "services": 15,
    "ingresses": 3
  }
}
```

### Cluster Info

```
GET /api/cluster-info
```

Returns cluster platform and version information.

**Response:**
```json
{
  "platform": "gke",
  "version": "v1.28.3-gke.1234",
  "context": "gke_project_zone_cluster",
  "server": "https://10.0.0.1"
}
```

**Platform Values:** `gke`, `eks`, `aks`, `minikube`, `kind`, `k3s`, `rancher`, `openshift`, `unknown`

### List Namespaces

```
GET /api/namespaces
```

Returns all namespaces in the cluster.

**Response:**
```json
{
  "namespaces": [
    {"name": "default", "status": "Active"},
    {"name": "kube-system", "status": "Active"},
    {"name": "production", "status": "Active"}
  ]
}
```

### API Resources

```
GET /api/api-resources
```

Returns available API resources for CRD discovery.

**Response:**
```json
{
  "resources": [
    {
      "name": "pods",
      "kind": "Pod",
      "namespaced": true,
      "group": "",
      "version": "v1"
    },
    {
      "name": "certificates",
      "kind": "Certificate",
      "namespaced": true,
      "group": "cert-manager.io",
      "version": "v1"
    }
  ]
}
```

---

## Topology

### Get Topology

```
GET /api/topology
```

Returns the cluster topology graph.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | Filter by namespace |
| `view` | string | View mode: `traffic` or `resources` |

**Response:**
```json
{
  "nodes": [
    {
      "id": "pod/default/nginx-abc123",
      "kind": "Pod",
      "name": "nginx-abc123",
      "namespace": "default",
      "status": "Running",
      "health": "healthy",
      "labels": {"app": "nginx"},
      "metadata": {
        "creationTimestamp": "2024-01-15T10:30:00Z",
        "ownerReferences": [...]
      }
    }
  ],
  "edges": [
    {
      "source": "service/default/nginx",
      "target": "pod/default/nginx-abc123",
      "type": "selects"
    }
  ]
}
```

---

## Resources

### List Resources

```
GET /api/resources/{kind}
```

List all resources of a specific kind.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `kind` | string | Resource kind (e.g., `pods`, `deployments`, `services`) |

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | Filter by namespace |

**Response:**
```json
{
  "items": [
    {
      "kind": "Pod",
      "metadata": {
        "name": "nginx-abc123",
        "namespace": "default"
      },
      "spec": {...},
      "status": {...}
    }
  ]
}
```

### Get Resource

```
GET /api/resources/{kind}/{namespace}/{name}
```

Get a single resource with its relationships.

**Response:**
```json
{
  "resource": {
    "kind": "Pod",
    "metadata": {...},
    "spec": {...},
    "status": {...}
  },
  "relationships": {
    "parent": {
      "kind": "ReplicaSet",
      "name": "nginx-5d4f6b7c8",
      "namespace": "default"
    },
    "children": [],
    "services": [
      {"kind": "Service", "name": "nginx", "namespace": "default"}
    ],
    "configMaps": [],
    "secrets": []
  }
}
```

### Update Resource

```
PUT /api/resources/{kind}/{namespace}/{name}
```

Update a resource from YAML.

**Request Body:** YAML string of the resource

**Response:**
```json
{
  "message": "Resource updated successfully"
}
```

### Delete Resource

```
DELETE /api/resources/{kind}/{namespace}/{name}
```

Delete a resource.

**Response:**
```json
{
  "message": "Resource deleted successfully"
}
```

---

## Events

### List Events

```
GET /api/events
```

Get recent Kubernetes events.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | Filter by namespace |
| `limit` | integer | Maximum events to return (default: 100) |

**Response:**
```json
{
  "events": [
    {
      "type": "Warning",
      "reason": "BackOff",
      "message": "Back-off restarting failed container",
      "involvedObject": {
        "kind": "Pod",
        "name": "nginx-abc123",
        "namespace": "default"
      },
      "count": 5,
      "firstTimestamp": "2024-01-15T10:00:00Z",
      "lastTimestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Event Stream (SSE)

```
GET /api/events/stream
```

Server-Sent Events stream for real-time updates.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | Filter by namespace |
| `view` | string | View mode for topology updates |

**Event Types:**
- `topology` - Topology graph update
- `event` - Kubernetes event
- `heartbeat` - Connection keep-alive

**Example:**
```
event: topology
data: {"nodes":[...],"edges":[...]}

event: event
data: {"type":"Normal","reason":"Scheduled",...}

event: heartbeat
data: {"timestamp":"2024-01-15T10:30:00Z"}
```

---

## Changes (History)

### List Changes

```
GET /api/changes
```

Get resource change history.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | Filter by namespace |
| `kind` | string | Filter by resource kind |
| `since` | string | RFC3339 timestamp |
| `limit` | integer | Maximum changes (default: 100) |
| `include_k8s_events` | boolean | Include K8s events |
| `include_managed` | boolean | Include managed resources |

**Response:**
```json
{
  "changes": [
    {
      "type": "updated",
      "kind": "Pod",
      "name": "nginx-abc123",
      "namespace": "default",
      "timestamp": "2024-01-15T10:30:00Z",
      "health": "healthy",
      "owner": {
        "kind": "ReplicaSet",
        "name": "nginx-5d4f6b7c8"
      }
    }
  ]
}
```

### Get Child Changes

```
GET /api/changes/{kind}/{namespace}/{name}/children
```

Get changes for child resources of a workload.

---

## Pod Operations

### Get Pod Logs

```
GET /api/pods/{namespace}/{name}/logs
```

Fetch pod logs (non-streaming).

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `container` | string | Container name |
| `tail` | integer | Lines from end (default: 100) |
| `previous` | boolean | Previous container logs |
| `timestamps` | boolean | Include timestamps |

**Response:**
```json
{
  "logs": "2024-01-15T10:30:00Z Starting server...\n2024-01-15T10:30:01Z Listening on port 8080\n"
}
```

### Stream Pod Logs (SSE)

```
GET /api/pods/{namespace}/{name}/logs/stream
```

Stream pod logs in real-time via Server-Sent Events.

**Query Parameters:** Same as above

**Event Format:**
```
data: 2024-01-15T10:30:00Z Log line here

data: 2024-01-15T10:30:01Z Another log line
```

### Pod Exec (WebSocket)

```
GET /api/pods/{namespace}/{name}/exec
```

WebSocket endpoint for interactive terminal sessions.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `container` | string | Container name |
| `shell` | string | Shell path (default: `/bin/sh`) |

**WebSocket Messages:**

Client → Server:
```json
{"type": "input", "data": "ls -la\n"}
{"type": "resize", "cols": 80, "rows": 24}
```

Server → Client:
```json
{"type": "output", "data": "total 48\ndrwxr-xr-x..."}
{"type": "error", "data": "command not found"}
```

---

## Port Forwarding

### List Port Forwards

```
GET /api/portforwards
```

List active port forward sessions.

**Response:**
```json
{
  "portforwards": [
    {
      "id": "pf-abc123",
      "type": "pod",
      "namespace": "default",
      "name": "nginx-abc123",
      "targetPort": 80,
      "localPort": 8080,
      "status": "active",
      "createdAt": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Start Port Forward

```
POST /api/portforwards
```

Start a new port forward session.

**Request Body:**
```json
{
  "type": "pod",
  "namespace": "default",
  "name": "nginx-abc123",
  "targetPort": 80,
  "localPort": 8080
}
```

If `localPort` is 0 or omitted, a random available port is assigned.

**Response:**
```json
{
  "id": "pf-abc123",
  "localPort": 8080,
  "url": "http://localhost:8080"
}
```

### Stop Port Forward

```
DELETE /api/portforwards/{id}
```

Stop a port forward session.

**Response:**
```json
{
  "message": "Port forward stopped"
}
```

### Get Available Ports

```
GET /api/portforwards/available/{type}/{namespace}/{name}
```

Get available ports for a pod or service.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | `pod` or `service` |
| `namespace` | string | Resource namespace |
| `name` | string | Resource name |

**Response:**
```json
{
  "ports": [
    {"port": 80, "name": "http", "protocol": "TCP"},
    {"port": 443, "name": "https", "protocol": "TCP"}
  ]
}
```

---

## Helm Management

### List Releases

```
GET /api/helm/releases
```

List all Helm releases.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | Filter by namespace |

**Response:**
```json
{
  "releases": [
    {
      "name": "nginx",
      "namespace": "default",
      "chart": "nginx",
      "chartVersion": "15.0.0",
      "appVersion": "1.25.0",
      "status": "deployed",
      "revision": 3,
      "updated": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Get Release Details

```
GET /api/helm/releases/{namespace}/{name}
```

Get detailed information about a release.

**Response:**
```json
{
  "release": {
    "name": "nginx",
    "namespace": "default",
    "info": {
      "status": "deployed",
      "description": "Upgrade complete",
      "firstDeployed": "2024-01-01T00:00:00Z",
      "lastDeployed": "2024-01-15T10:30:00Z"
    },
    "chart": {
      "name": "nginx",
      "version": "15.0.0",
      "appVersion": "1.25.0"
    },
    "config": {...},
    "version": 3
  }
}
```

### Get Release Manifest

```
GET /api/helm/releases/{namespace}/{name}/manifest
```

Get rendered Kubernetes manifests for a release.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `revision` | integer | Specific revision (default: latest) |

**Response:**
```json
{
  "manifest": "---\napiVersion: v1\nkind: Service\n..."
}
```

### Get Release Values

```
GET /api/helm/releases/{namespace}/{name}/values
```

Get computed values for a release.

**Response:**
```json
{
  "values": {
    "replicaCount": 3,
    "image": {
      "repository": "nginx",
      "tag": "1.25.0"
    }
  }
}
```

### Get Release Diff

```
GET /api/helm/releases/{namespace}/{name}/diff
```

Get diff between two revisions.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `revision1` | integer | First revision |
| `revision2` | integer | Second revision |

**Response:**
```json
{
  "diff": "--- revision 2\n+++ revision 3\n@@ -10,7 +10,7 @@\n   replicas: 2\n+  replicas: 3\n"
}
```

### Check Upgrade Availability

```
GET /api/helm/releases/{namespace}/{name}/upgrade-info
```

Check if chart upgrades are available.

**Response:**
```json
{
  "currentVersion": "15.0.0",
  "latestVersion": "15.1.0",
  "upgradeAvailable": true
}
```

### Batch Upgrade Check

```
GET /api/helm/upgrade-check
```

Check upgrade availability for all releases.

**Response:**
```json
{
  "releases": [
    {
      "name": "nginx",
      "namespace": "default",
      "currentVersion": "15.0.0",
      "latestVersion": "15.1.0",
      "upgradeAvailable": true
    }
  ]
}
```

### Rollback Release

```
POST /api/helm/releases/{namespace}/{name}/rollback
```

Rollback to a previous revision.

**Request Body:**
```json
{
  "revision": 2
}
```

**Response:**
```json
{
  "message": "Rollback to revision 2 successful"
}
```

### Upgrade Release

```
POST /api/helm/releases/{namespace}/{name}/upgrade
```

Upgrade release to a new chart version.

**Request Body:**
```json
{
  "version": "15.1.0",
  "values": {}
}
```

**Response:**
```json
{
  "message": "Upgrade successful",
  "revision": 4
}
```

### Uninstall Release

```
DELETE /api/helm/releases/{namespace}/{name}
```

Uninstall a Helm release.

**Response:**
```json
{
  "message": "Release uninstalled successfully"
}
```

---

## Error Codes

| Status | Description |
|--------|-------------|
| 200 | Success |
| 400 | Bad request (invalid parameters) |
| 404 | Resource not found |
| 500 | Internal server error |

## Rate Limiting

The API does not implement rate limiting. All requests go directly to the Kubernetes API server, which may have its own rate limits.
