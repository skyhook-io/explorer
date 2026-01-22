# User Guide

This guide covers all features of the Skyhook Explorer web interface.

## Interface Overview

The Explorer interface consists of:

1. **Header** - Cluster info, namespace selector, view mode toggle
2. **Main Canvas** - Interactive topology graph
3. **Side Panel** - Resource details (opens on selection)
4. **Bottom Dock** - Terminal sessions, logs, port forwards

## Topology View

### Navigation

- **Pan**: Click and drag on empty space
- **Zoom**: Mouse wheel or pinch gesture
- **Select**: Click on a node
- **Multi-select**: Shift+click or drag a selection box
- **Fit view**: Double-click on empty space

### View Modes

**Traffic View**
Shows network flow through your cluster:
```
Ingress → Service → Pod
```
Best for understanding how external traffic reaches your workloads.

**Resources View**
Shows ownership hierarchy:
```
Deployment → ReplicaSet → Pod
DaemonSet → Pod
StatefulSet → Pod
Job → Pod
CronJob → Job → Pod
```
Best for understanding resource relationships and debugging.

### Node Types

| Icon | Resource | Description |
|------|----------|-------------|
| Cube | Pod | Running container(s) |
| Boxes | Deployment | Manages ReplicaSets |
| Boxes (dashed) | ReplicaSet | Manages Pod replicas |
| Server | DaemonSet | Pod on each node |
| Database | StatefulSet | Stateful workloads |
| Play | Job | One-time task |
| Clock | CronJob | Scheduled jobs |
| Globe | Service | Network endpoint |
| Arrow | Ingress | External access |
| File | ConfigMap | Configuration data |
| Lock | Secret | Sensitive data |
| Chart | HPA | Auto-scaling |
| Disk | PVC | Storage claim |

### Health Indicators

Node colors indicate health status:
- **Green**: Healthy/Running
- **Yellow**: Warning/Pending
- **Red**: Error/Failed
- **Gray**: Unknown/Terminating

## Resource Details Panel

Click any node to open the details panel showing:

### Overview Tab
- Resource name and namespace
- Labels and annotations
- Creation timestamp
- Owner references

### Status Tab
- Current status and phase
- Conditions with timestamps
- Container statuses (for Pods)
- Replica counts (for Deployments)

### Related Tab
- Parent resources (owner)
- Child resources (owned)
- Network resources (Services, Ingresses)
- Config resources (ConfigMaps, Secrets)

### YAML Tab
- Full resource YAML
- Syntax highlighted
- Editable with save button

## Pod Operations

### Terminal Access

1. Select a Pod in the topology
2. Click **Terminal** in the detail panel
3. Choose a container (if multiple)
4. Select shell (`/bin/sh`, `/bin/bash`, or custom)
5. Terminal opens in the bottom dock

Terminal features:
- Full TTY support with colors
- Copy/paste (Ctrl+Shift+C/V or Cmd+C/V)
- Resize with dock handle
- Multiple concurrent sessions

### Log Streaming

1. Select a Pod in the topology
2. Click **Logs** in the detail panel
3. Configure options:
   - Container selection
   - Tail lines (default: 100)
   - Previous container logs
   - Timestamps
4. Logs stream in real-time

Log features:
- Auto-scroll to bottom
- Pause/resume streaming
- Search/filter (Ctrl+F)
- Download logs
- Clear display

## Port Forwarding

### Starting a Port Forward

1. Select a Pod or Service
2. Click **Port Forward** in the detail panel
3. Select the target port from available ports
4. Optionally specify local port (auto-assigned if empty)
5. Click **Start**

### Managing Port Forwards

The Port Forward panel shows all active sessions:
- Target resource and port
- Local port and URL
- Status indicator
- Stop button

Click the local URL to open in a new tab.

### Auto-Discovery

Explorer automatically discovers available ports from:
- Pod container port definitions
- Service port configurations

## Helm Management

Access via the **Helm** tab in the navigation.

### Release List

Shows all Helm releases across namespaces:
- Release name and namespace
- Chart name and version
- App version
- Status (deployed, failed, etc.)
- Revision number
- Last updated timestamp

### Release Details

Click a release to see:

**Overview**
- Chart information
- Release notes
- Resource count

**Values**
- Computed values YAML
- User-supplied vs default values

**Manifest**
- Rendered Kubernetes manifests
- All resources created by the release

**History**
- All revisions with timestamps
- Diff between revisions
- Rollback to any revision

### Release Actions

**Rollback**
1. Open release details
2. Go to History tab
3. Select a revision
4. Click **Rollback**
5. Confirm the operation

**Uninstall**
1. Open release details
2. Click **Uninstall**
3. Confirm deletion

## Events Timeline

Access via the **Events** tab in the navigation.

Shows recent Kubernetes events:
- Resource involved
- Event type (Normal/Warning)
- Reason and message
- Timestamp
- Count (if repeated)

Filter events by:
- Namespace
- Resource type
- Event type
- Time range

## Change History

The **Changes** view shows resource modifications:

- Created/Updated/Deleted events
- Resource kind and name
- Timestamp
- Health state changes
- Owner information

Enable persistent history with `--persist-history` to retain changes across restarts.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Escape` | Close panel/modal |
| `?` | Show keyboard shortcuts |
| `/` | Focus search |
| `r` | Refresh topology |
| `f` | Fit view to screen |
| `1` | Traffic view |
| `2` | Resources view |

## Tips & Tricks

### Finding Resources
Use the search box to filter the topology by resource name. Matching nodes are highlighted.

### Debugging Pods
1. Check Pod status and conditions
2. View container statuses for restart counts
3. Check logs for errors
4. Use terminal to inspect container state

### Understanding Relationships
Switch to Resources view to see the full ownership chain. Click "Related" tab to see all connected resources.

### Monitoring Deployments
Watch the topology during a rollout to see new ReplicaSets and Pods being created while old ones terminate.
