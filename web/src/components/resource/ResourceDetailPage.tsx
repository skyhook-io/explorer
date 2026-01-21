import { useState, useMemo, useEffect, useRef } from 'react'
import { clsx } from 'clsx'
import {
  ArrowLeft,
  Clock,
  AlertCircle,
  CheckCircle,
  RefreshCw,
  Plus,
  Trash2,
  ChevronRight,
  Layers,
  Server,
  Eye,
  EyeOff,
  Terminal,
  FileText,
  Activity,
  Container,
  MoreVertical,
  RotateCcw,
  Scale,
  Copy,
  Check,
} from 'lucide-react'
import type { TimelineEvent, TimeRange, ResourceRef, Relationships } from '../../types'
import { useChanges, useResourceWithRelationships, usePodLogs } from '../../api/client'
import { DiffViewer } from '../events/DiffViewer'

// Known noisy resources that update constantly
const NOISY_NAME_PATTERNS = [
  /^kube-scheduler$/,
  /^kube-controller-manager$/,
  /-leader-election$/,
  /-lock$/,
  /-lease$/,
]

const NOISY_KINDS = new Set(['Lease', 'Endpoints', 'EndpointSlice', 'Event'])

function isRoutineEvent(event: TimelineEvent): boolean {
  if (event.kind === 'Event' && event.type === 'change') return true
  if (event.operation !== 'update') return false
  if (NOISY_KINDS.has(event.kind)) return true
  if (NOISY_NAME_PATTERNS.some(pattern => pattern.test(event.name))) return true
  if (event.kind === 'ConfigMap') {
    if (event.name.endsWith('-lock') || event.name.endsWith('-lease') || event.name.endsWith('-leader')) {
      return true
    }
  }
  return false
}

const PROBLEMATIC_REASONS = new Set([
  'BackOff', 'CrashLoopBackOff', 'Failed', 'FailedScheduling', 'FailedMount',
  'FailedAttachVolume', 'FailedCreate', 'FailedDelete', 'Unhealthy', 'Killing',
  'Evicted', 'OOMKilling', 'OOMKilled', 'NodeNotReady', 'NetworkNotReady',
  'FailedSync', 'FailedValidation', 'InvalidImageName', 'ErrImagePull',
  'ImagePullBackOff', 'FailedPreStopHook', 'FailedPostStartHook',
])

function isProblematicEvent(event: TimelineEvent): boolean {
  if (event.eventType === 'Warning') return true
  if (event.reason && PROBLEMATIC_REASONS.has(event.reason)) return true
  return false
}

type TabType = 'overview' | 'pods' | 'events' | 'yaml'

interface ResourceDetailPageProps {
  kind: string
  namespace: string
  name: string
  onBack: () => void
  onNavigateToResource?: (kind: string, namespace: string, name: string) => void
}

export function ResourceDetailPage({
  kind,
  namespace,
  name,
  onBack,
  onNavigateToResource,
}: ResourceDetailPageProps) {
  const [activeTab, setActiveTab] = useState<TabType>('overview')
  const [timeRange, setTimeRange] = useState<TimeRange>('1h')
  const [selectedPod, setSelectedPod] = useState<string | null>(null)
  const [showRoutineEvents, setShowRoutineEvents] = useState(false)

  // Fetch resource with relationships
  const { data: resourceResponse, isLoading: resourceLoading } = useResourceWithRelationships<any>(kind, namespace, name)
  const resource = resourceResponse?.resource
  const relationships = resourceResponse?.relationships

  // Fetch events
  const { data: allEvents, isLoading: eventsLoading } = useChanges({
    namespace,
    timeRange,
    includeK8sEvents: true,
    includeManaged: true,
    limit: 500,
  })

  // Filter events to this resource and children
  const resourceEvents = useMemo(() => {
    if (!allEvents) return []
    let events = allEvents.filter(e =>
      (e.kind === kind && e.namespace === namespace && e.name === name) ||
      (e.owner?.kind === kind && e.owner?.name === name)
    )
    if (!showRoutineEvents) {
      events = events.filter(e => !isRoutineEvent(e))
    }
    return events
  }, [allEvents, kind, namespace, name, showRoutineEvents])

  const routineEventCount = useMemo(() => {
    if (!allEvents) return 0
    const relevant = allEvents.filter(e =>
      (e.kind === kind && e.namespace === namespace && e.name === name) ||
      (e.owner?.kind === kind && e.owner?.name === name)
    )
    return relevant.filter(isRoutineEvent).length
  }, [allEvents, kind, namespace, name])

  // Extract metadata from resource
  const metadata = useMemo(() => extractMetadata(kind, resource), [kind, resource])

  // Get pods from relationships (only direct pods, for Services)
  const pods = relationships?.pods || []

  // Determine health from resource status
  const healthState = useMemo(() => {
    if (!resource) return 'unknown'
    return determineHealth(kind, resource)
  }, [kind, resource])

  // Get warning count
  const warningCount = resourceEvents.filter(isProblematicEvent).length

  return (
    <div className="flex flex-col h-full w-full bg-slate-900">
      {/* Header */}
      <div className="flex-shrink-0 border-b border-slate-700 bg-slate-800">
        {/* Top bar with back button, title, actions */}
        <div className="px-4 py-3 flex items-center gap-4">
          <button
            onClick={onBack}
            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <KindBadge kind={kind} />
              <HealthBadge state={healthState} />
              {warningCount > 0 && (
                <span className="flex items-center gap-1 text-xs px-2 py-0.5 rounded bg-amber-500/20 text-amber-400">
                  <AlertCircle className="w-3 h-3" />
                  {warningCount} warning{warningCount !== 1 ? 's' : ''}
                </span>
              )}
            </div>
            <h1 className="text-lg font-semibold text-white truncate">{name}</h1>
            <p className="text-sm text-slate-400">{namespace}</p>
          </div>

          <div className="flex items-center gap-2">
            <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
            <ActionsDropdown kind={kind} namespace={namespace} name={name} />
          </div>
        </div>

        {/* Metadata bar */}
        {metadata.length > 0 && (
          <div className="px-4 py-2 bg-slate-800/50 border-t border-slate-700/50 flex flex-wrap gap-x-6 gap-y-1">
            {metadata.map((item, i) => (
              <div key={i} className="flex items-center gap-2 text-sm">
                <span className="text-slate-500">{item.label}:</span>
                <span className="text-slate-300 font-mono text-xs">{item.value}</span>
              </div>
            ))}
          </div>
        )}

        {/* Mini timeline */}
        <MiniTimeline events={resourceEvents} timeRange={timeRange} />

        {/* Tabs */}
        <div className="px-4 flex gap-1">
          <TabButton active={activeTab === 'overview'} onClick={() => setActiveTab('overview')}>
            <Activity className="w-4 h-4" />
            Overview
          </TabButton>
          {pods.length > 0 && (
            <TabButton active={activeTab === 'pods'} onClick={() => setActiveTab('pods')}>
              <Container className="w-4 h-4" />
              Logs
              <span className="ml-1 px-1.5 py-0.5 text-xs bg-slate-700 rounded">{pods.length}</span>
            </TabButton>
          )}
          <TabButton active={activeTab === 'events'} onClick={() => setActiveTab('events')}>
            <Clock className="w-4 h-4" />
            Events
            {resourceEvents.length > 0 && (
              <span className="ml-1 px-1.5 py-0.5 text-xs bg-slate-700 rounded">{resourceEvents.length}</span>
            )}
          </TabButton>
          <TabButton active={activeTab === 'yaml'} onClick={() => setActiveTab('yaml')}>
            <FileText className="w-4 h-4" />
            YAML
          </TabButton>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'overview' && (
          <OverviewTab
            resource={resource}
            relationships={relationships}
            events={resourceEvents}
            isLoading={resourceLoading || eventsLoading}
            onNavigate={onNavigateToResource}
            kind={kind}
          />
        )}
        {activeTab === 'pods' && (
          <LogsTab
            pods={pods}
            namespace={namespace}
            selectedPod={selectedPod}
            onSelectPod={setSelectedPod}
          />
        )}
        {activeTab === 'events' && (
          <EventsTab
            events={resourceEvents}
            isLoading={eventsLoading}
            showRoutine={showRoutineEvents}
            routineCount={routineEventCount}
            onToggleRoutine={setShowRoutineEvents}
            resourceKind={kind}
            resourceName={name}
          />
        )}
        {activeTab === 'yaml' && (
          <YamlTab resource={resource} isLoading={resourceLoading} />
        )}
      </div>
    </div>
  )
}

// Helper to extract key metadata from resource
function extractMetadata(kind: string, resource: any): { label: string; value: string }[] {
  if (!resource) return []
  const items: { label: string; value: string }[] = []

  const spec = resource.spec || {}
  const status = resource.status || {}

  switch (kind) {
    case 'Deployment':
    case 'StatefulSet':
    case 'DaemonSet':
      if (spec.replicas !== undefined) {
        const ready = status.readyReplicas || 0
        items.push({ label: 'Replicas', value: `${ready}/${spec.replicas}` })
      }
      // Get image from first container
      const containers = spec.template?.spec?.containers || []
      if (containers[0]?.image) {
        items.push({ label: 'Image', value: containers[0].image })
      }
      if (spec.selector?.matchLabels?.app) {
        items.push({ label: 'App', value: spec.selector.matchLabels.app })
      }
      break

    case 'ReplicaSet': {
      const rsReady = status.readyReplicas || 0
      const rsTotal = spec.replicas || status.replicas || 0
      items.push({ label: 'Replicas', value: `${rsReady}/${rsTotal}` })
      // Get image from first container
      const rsContainers = spec.template?.spec?.containers || []
      if (rsContainers[0]?.image) {
        items.push({ label: 'Image', value: rsContainers[0].image })
      }
      // Get owner deployment name from ownerReferences
      const ownerRefs = resource.metadata?.ownerReferences || []
      const deployOwner = ownerRefs.find((ref: any) => ref.kind === 'Deployment')
      if (deployOwner) {
        items.push({ label: 'Owner', value: deployOwner.name })
      }
      break
    }

    case 'Service':
      if (spec.type) items.push({ label: 'Type', value: spec.type })
      if (spec.clusterIP) items.push({ label: 'ClusterIP', value: spec.clusterIP })
      if (spec.ports?.length) {
        const ports = spec.ports.map((p: any) => `${p.port}${p.targetPort ? ':' + p.targetPort : ''}/${p.protocol || 'TCP'}`).join(', ')
        items.push({ label: 'Ports', value: ports })
      }
      break

    case 'Pod':
      if (status.phase) items.push({ label: 'Phase', value: status.phase })
      if (status.podIP) items.push({ label: 'Pod IP', value: status.podIP })
      if (spec.nodeName) items.push({ label: 'Node', value: spec.nodeName })
      const restarts = status.containerStatuses?.reduce((sum: number, c: any) => sum + (c.restartCount || 0), 0) || 0
      if (restarts > 0) items.push({ label: 'Restarts', value: String(restarts) })
      break

    case 'Ingress':
      const rules = spec.rules || []
      if (rules[0]?.host) items.push({ label: 'Host', value: rules[0].host })
      if (spec.ingressClassName) items.push({ label: 'Class', value: spec.ingressClassName })
      break

    case 'ConfigMap':
    case 'Secret':
      const dataKeys = Object.keys(resource.data || {})
      items.push({ label: 'Keys', value: dataKeys.length > 3 ? `${dataKeys.slice(0, 3).join(', ')}...` : dataKeys.join(', ') || '(empty)' })
      break

    case 'HPA':
      if (spec.minReplicas) items.push({ label: 'Min', value: String(spec.minReplicas) })
      if (spec.maxReplicas) items.push({ label: 'Max', value: String(spec.maxReplicas) })
      if (status.currentReplicas) items.push({ label: 'Current', value: String(status.currentReplicas) })
      break
  }

  return items
}

// Determine health from resource
function determineHealth(kind: string, resource: any): string {
  if (!resource) return 'unknown'
  const status = resource.status || {}

  switch (kind) {
    case 'Deployment':
    case 'StatefulSet':
    case 'DaemonSet': {
      const desired = resource.spec?.replicas || 0
      const ready = status.readyReplicas || 0
      const updated = status.updatedReplicas || 0
      if (ready === 0 && desired > 0) return 'unhealthy'
      if (ready < desired || updated < desired) return 'degraded'
      return 'healthy'
    }
    case 'ReplicaSet': {
      const desired = resource.spec?.replicas || 0
      const ready = status.readyReplicas || 0
      if (ready === 0 && desired > 0) return 'unhealthy'
      if (ready < desired) return 'degraded'
      return 'healthy'
    }
    case 'Pod': {
      const phase = status.phase
      if (phase === 'Running' || phase === 'Succeeded') {
        const containers = status.containerStatuses || []
        const allReady = containers.every((c: any) => c.ready)
        return allReady ? 'healthy' : 'degraded'
      }
      if (phase === 'Pending') return 'degraded'
      return 'unhealthy'
    }
    case 'Service':
      return 'healthy' // Services are always "healthy" if they exist
    default:
      return 'unknown'
  }
}

// Sub-components

function KindBadge({ kind }: { kind: string }) {
  const colors: Record<string, string> = {
    Deployment: 'bg-blue-900/50 text-blue-400',
    StatefulSet: 'bg-purple-900/50 text-purple-400',
    DaemonSet: 'bg-blue-900/50 text-blue-400',
    Service: 'bg-cyan-900/50 text-cyan-400',
    Pod: 'bg-green-900/50 text-green-400',
    ReplicaSet: 'bg-violet-900/50 text-violet-400',
    Ingress: 'bg-orange-900/50 text-orange-400',
    ConfigMap: 'bg-yellow-900/50 text-yellow-400',
    Secret: 'bg-red-900/50 text-red-400',
    Job: 'bg-teal-900/50 text-teal-400',
    CronJob: 'bg-teal-900/50 text-teal-400',
  }
  return (
    <span className={clsx('text-xs px-2 py-0.5 rounded font-medium', colors[kind] || 'bg-slate-700 text-slate-300')}>
      {kind}
    </span>
  )
}

function HealthBadge({ state }: { state: string }) {
  const config: Record<string, { bg: string; text: string; icon: typeof CheckCircle }> = {
    healthy: { bg: 'bg-green-500/20', text: 'text-green-400', icon: CheckCircle },
    degraded: { bg: 'bg-yellow-500/20', text: 'text-yellow-400', icon: AlertCircle },
    unhealthy: { bg: 'bg-red-500/20', text: 'text-red-400', icon: AlertCircle },
    unknown: { bg: 'bg-slate-500/20', text: 'text-slate-400', icon: Clock },
  }
  const { bg, text, icon: Icon } = config[state] || config.unknown
  return (
    <span className={clsx('inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded', bg, text)}>
      <Icon className="w-3 h-3" />
      {state}
    </span>
  )
}

function TabButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={clsx(
        'flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 transition-colors',
        active
          ? 'text-white border-blue-500'
          : 'text-slate-400 border-transparent hover:text-white hover:border-slate-600'
      )}
    >
      {children}
    </button>
  )
}

function TimeRangeSelector({ value, onChange }: { value: TimeRange; onChange: (v: TimeRange) => void }) {
  const options: { value: TimeRange; label: string }[] = [
    { value: '5m', label: '5m' },
    { value: '30m', label: '30m' },
    { value: '1h', label: '1h' },
    { value: '6h', label: '6h' },
    { value: '24h', label: '24h' },
  ]
  return (
    <div className="flex items-center gap-0.5 p-0.5 bg-slate-700 rounded-lg">
      {options.map(opt => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={clsx(
            'px-2 py-1 text-xs rounded-md transition-colors',
            value === opt.value ? 'bg-slate-600 text-white' : 'text-slate-400 hover:text-white'
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

function ActionsDropdown({ kind }: { kind: string; namespace: string; name: string }) {
  const [open, setOpen] = useState(false)

  // Placeholder actions - would need backend support
  const actions = [
    { label: 'Describe', icon: FileText, action: () => console.log('describe') },
    { label: 'Restart', icon: RotateCcw, action: () => console.log('restart'), disabled: !['Deployment', 'StatefulSet', 'DaemonSet'].includes(kind) },
    { label: 'Scale', icon: Scale, action: () => console.log('scale'), disabled: !['Deployment', 'StatefulSet'].includes(kind) },
    { label: 'Delete', icon: Trash2, action: () => console.log('delete'), danger: true },
  ]

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg"
      >
        <MoreVertical className="w-5 h-5" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full mt-1 z-50 w-48 bg-slate-800 border border-slate-700 rounded-lg shadow-xl py-1">
            {actions.map((action, i) => (
              <button
                key={i}
                onClick={() => { action.action(); setOpen(false) }}
                disabled={action.disabled}
                className={clsx(
                  'w-full px-3 py-2 text-sm text-left flex items-center gap-2 transition-colors',
                  action.disabled
                    ? 'text-slate-600 cursor-not-allowed'
                    : action.danger
                    ? 'text-red-400 hover:bg-red-900/30'
                    : 'text-slate-300 hover:bg-slate-700'
                )}
              >
                <action.icon className="w-4 h-4" />
                {action.label}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function MiniTimeline({ events, timeRange }: { events: TimelineEvent[]; timeRange: TimeRange }) {
  const timeRangeMs: Record<TimeRange, number> = {
    '5m': 5 * 60 * 1000,
    '30m': 30 * 60 * 1000,
    '1h': 60 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    'all': 24 * 60 * 60 * 1000,
  }

  const now = Date.now()
  const windowMs = timeRangeMs[timeRange]
  const start = now - windowMs

  // Calculate health segments (simplified: just show event dots for now)
  const eventPositions = events.map(e => {
    const ts = new Date(e.timestamp).getTime()
    const x = ((ts - start) / windowMs) * 100
    return { x: Math.max(0, Math.min(100, x)), event: e }
  }).filter(e => e.x >= 0 && e.x <= 100)

  return (
    <div className="px-4 py-2 border-t border-slate-700/50">
      <div className="relative h-6 bg-slate-700/30 rounded overflow-hidden">
        {/* Health bar background - gradient from past to now */}
        <div className="absolute inset-0 bg-gradient-to-r from-slate-700/50 to-slate-600/30" />

        {/* Event markers */}
        {eventPositions.map((ep, i) => {
          const isProblematic = isProblematicEvent(ep.event)
          const isDelete = ep.event.operation === 'delete'
          const isAdd = ep.event.operation === 'add'
          return (
            <div
              key={i}
              className={clsx(
                'absolute top-1/2 -translate-y-1/2 w-2 h-2 rounded-full',
                isProblematic ? 'bg-amber-500' :
                isDelete ? 'bg-red-500' :
                isAdd ? 'bg-green-500' :
                'bg-blue-500'
              )}
              style={{ left: `${ep.x}%` }}
              title={`${ep.event.operation || ep.event.reason} at ${new Date(ep.event.timestamp).toLocaleTimeString()}`}
            />
          )
        })}

        {/* Time labels */}
        <div className="absolute left-2 top-1/2 -translate-y-1/2 text-xs text-slate-500">
          -{timeRange}
        </div>
        <div className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-slate-500">
          Now
        </div>
      </div>
    </div>
  )
}

// Tab content components

function OverviewTab({
  resource,
  relationships,
  events,
  isLoading,
  onNavigate,
  kind,
}: {
  resource: any
  relationships?: Relationships
  events: TimelineEvent[]
  isLoading: boolean
  onNavigate?: (kind: string, namespace: string, name: string) => void
  kind: string
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-500">
        <RefreshCw className="w-5 h-5 animate-spin mr-2" />
        Loading...
      </div>
    )
  }

  const recentEvents = events.slice(0, 5)
  const warnings = events.filter(isProblematicEvent).slice(0, 3)

  return (
    <div className="h-full overflow-auto">
      <div className="p-4 grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Active Issues */}
        {warnings.length > 0 && (
          <div className="lg:col-span-2 bg-amber-950/30 border border-amber-800/50 rounded-lg p-4">
            <h3 className="text-sm font-medium text-amber-400 mb-3 flex items-center gap-2">
              <AlertCircle className="w-4 h-4" />
              Active Issues ({warnings.length})
            </h3>
            <div className="space-y-2">
              {warnings.map(event => (
                <div key={event.id} className="flex items-start gap-3 text-sm">
                  <div className="w-2 h-2 rounded-full bg-amber-500 mt-1.5 flex-shrink-0" />
                  <div>
                    <span className="text-amber-300 font-medium">{event.reason || event.operation}</span>
                    {event.message && <p className="text-amber-200/70 text-xs mt-0.5">{event.message}</p>}
                    <p className="text-amber-200/50 text-xs">{new Date(event.timestamp).toLocaleString()}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Recent Activity */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
            <Activity className="w-4 h-4" />
            Recent Activity
          </h3>
          {recentEvents.length === 0 ? (
            <p className="text-sm text-slate-500">No recent events</p>
          ) : (
            <div className="space-y-2">
              {recentEvents.map(event => (
                <div key={event.id} className="flex items-start gap-3 text-sm">
                  <div className={clsx(
                    'w-2 h-2 rounded-full mt-1.5 flex-shrink-0',
                    isProblematicEvent(event) ? 'bg-amber-500' :
                    event.operation === 'delete' ? 'bg-red-500' :
                    event.operation === 'add' ? 'bg-green-500' :
                    'bg-blue-500'
                  )} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-slate-300">{event.reason || event.operation}</span>
                      {event.kind !== kind && (
                        <span className="text-xs px-1 py-0.5 bg-slate-700 rounded text-slate-400">{event.kind}</span>
                      )}
                    </div>
                    {event.diff?.summary && <p className="text-slate-500 text-xs truncate">{event.diff.summary}</p>}
                    <p className="text-slate-600 text-xs">{new Date(event.timestamp).toLocaleTimeString()}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Related Resources */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
            <Layers className="w-4 h-4" />
            Related Resources
          </h3>
          <RelatedResources relationships={relationships} isLoading={isLoading} onNavigate={onNavigate} />
        </div>

        {/* Resource Status (for workloads) */}
        {resource?.status && (
          <div className="lg:col-span-2 bg-slate-800/50 border border-slate-700 rounded-lg p-4">
            <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
              <Server className="w-4 h-4" />
              Status
            </h3>
            <StatusGrid status={resource.status} kind={kind} />
          </div>
        )}
      </div>
    </div>
  )
}

function StatusGrid({ status, kind }: { status: any; kind: string }) {
  const items: { label: string; value: string | number; color?: string }[] = []

  if (['Deployment', 'StatefulSet', 'DaemonSet'].includes(kind)) {
    items.push({ label: 'Ready', value: status.readyReplicas || 0, color: status.readyReplicas > 0 ? 'text-green-400' : 'text-slate-400' })
    items.push({ label: 'Available', value: status.availableReplicas || 0 })
    items.push({ label: 'Updated', value: status.updatedReplicas || 0 })
    if (status.unavailableReplicas) {
      items.push({ label: 'Unavailable', value: status.unavailableReplicas, color: 'text-red-400' })
    }
  } else if (kind === 'ReplicaSet') {
    const ready = status.readyReplicas || 0
    const replicas = status.replicas || 0
    items.push({ label: 'Ready', value: ready, color: ready === replicas && ready > 0 ? 'text-green-400' : ready > 0 ? 'text-yellow-400' : 'text-red-400' })
    items.push({ label: 'Replicas', value: replicas })
    if (status.availableReplicas !== undefined) {
      items.push({ label: 'Available', value: status.availableReplicas })
    }
    if (status.fullyLabeledReplicas !== undefined && status.fullyLabeledReplicas !== replicas) {
      items.push({ label: 'Labeled', value: status.fullyLabeledReplicas })
    }
  } else if (kind === 'Pod') {
    items.push({ label: 'Phase', value: status.phase || 'Unknown' })
    if (status.conditions) {
      const ready = status.conditions.find((c: any) => c.type === 'Ready')
      if (ready) {
        items.push({ label: 'Ready', value: ready.status, color: ready.status === 'True' ? 'text-green-400' : 'text-red-400' })
      }
    }
    // Show container restart counts if any
    if (status.containerStatuses) {
      const totalRestarts = status.containerStatuses.reduce((sum: number, c: any) => sum + (c.restartCount || 0), 0)
      if (totalRestarts > 0) {
        items.push({ label: 'Restarts', value: totalRestarts, color: totalRestarts > 5 ? 'text-red-400' : 'text-yellow-400' })
      }
    }
  }

  if (items.length === 0) {
    return <p className="text-sm text-slate-500">No status information available</p>
  }

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
      {items.map((item, i) => (
        <div key={i}>
          <p className="text-xs text-slate-500">{item.label}</p>
          <p className={clsx('text-lg font-semibold', item.color || 'text-white')}>{item.value}</p>
        </div>
      ))}
    </div>
  )
}

function RelatedResources({
  relationships,
  isLoading,
  onNavigate
}: {
  relationships?: Relationships
  isLoading?: boolean
  onNavigate?: (kind: string, namespace: string, name: string) => void
}) {
  if (isLoading) {
    return <p className="text-sm text-slate-500">Loading relationships...</p>
  }

  if (!relationships) {
    return <p className="text-sm text-slate-500">No related resources</p>
  }

  const sections: { title: string; items: ResourceRef[] }[] = []

  if (relationships.owner) sections.push({ title: 'Owner', items: [relationships.owner] })
  if (relationships.services?.length) sections.push({ title: 'Services', items: relationships.services })
  if (relationships.ingresses?.length) sections.push({ title: 'Ingresses', items: relationships.ingresses })
  if (relationships.children?.length) sections.push({ title: 'Children', items: relationships.children.slice(0, 5) })
  if (relationships.configRefs?.length) sections.push({ title: 'Config', items: relationships.configRefs })
  if (relationships.pods?.length) sections.push({ title: 'Pods', items: relationships.pods.slice(0, 5) })

  if (sections.length === 0) {
    return <p className="text-sm text-slate-500">No related resources</p>
  }

  return (
    <div className="space-y-3">
      {sections.map(section => (
        <div key={section.title}>
          <p className="text-xs text-slate-500 mb-1">{section.title}</p>
          <div className="space-y-1">
            {section.items.map(item => (
              <button
                key={`${item.kind}/${item.namespace}/${item.name}`}
                onClick={() => onNavigate?.(item.kind, item.namespace, item.name)}
                className="w-full text-left px-2 py-1.5 rounded hover:bg-slate-700/50 flex items-center gap-2 group"
              >
                <KindBadge kind={item.kind} />
                <span className="text-sm text-slate-300 truncate flex-1 group-hover:text-white">{item.name}</span>
                <ChevronRight className="w-3 h-3 text-slate-600 group-hover:text-slate-400" />
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function LogsTab({
  pods,
  namespace,
  selectedPod,
  onSelectPod,
}: {
  pods: ResourceRef[]
  namespace: string
  selectedPod: string | null
  onSelectPod: (name: string | null) => void
}) {
  // Auto-select first pod if none selected
  useEffect(() => {
    if (pods.length > 0 && !selectedPod) {
      onSelectPod(pods[0].name)
    }
  }, [pods, selectedPod, onSelectPod])

  if (pods.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-500">
        <Terminal className="w-12 h-12 mb-4 opacity-50" />
        <p>No pods available</p>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Pod selector - horizontal tabs */}
      {pods.length > 1 && (
        <div className="flex-shrink-0 border-b border-slate-700 bg-slate-800/50 px-4 py-2 flex gap-2 overflow-x-auto">
          {pods.map(pod => (
            <button
              key={pod.name}
              onClick={() => onSelectPod(pod.name)}
              className={clsx(
                'px-3 py-1.5 text-sm rounded-lg whitespace-nowrap transition-colors',
                selectedPod === pod.name
                  ? 'bg-blue-500 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              )}
            >
              {pod.name.length > 40 ? '...' + pod.name.slice(-37) : pod.name}
            </button>
          ))}
        </div>
      )}

      {/* Logs panel */}
      {selectedPod && (
        <div className="flex-1 min-h-0">
          <PodLogsPanel
            namespace={pods.find(p => p.name === selectedPod)?.namespace || namespace}
            podName={selectedPod}
            onClose={() => {}} // No close needed when it's the main content
            showHeader={pods.length === 1}
          />
        </div>
      )}
    </div>
  )
}

function PodLogsPanel({
  namespace,
  podName,
  onClose,
  showHeader = true
}: {
  namespace: string
  podName: string
  onClose: () => void
  showHeader?: boolean
}) {
  const [follow, setFollow] = useState(true)
  const [container, setContainer] = useState<string>('')
  const logsRef = useRef<HTMLPreElement>(null)

  const { data: logsData, isLoading, refetch } = usePodLogs(namespace, podName, {
    container: container || undefined,
    tailLines: 500,
  })

  // Get container list from logs response
  const containers = logsData?.containers || []
  const logs = container && logsData?.logs ? logsData.logs[container] : Object.values(logsData?.logs || {})[0] || ''

  // Auto-scroll when following
  useEffect(() => {
    if (follow && logsRef.current) {
      logsRef.current.scrollTop = logsRef.current.scrollHeight
    }
  }, [logs, follow])

  // Auto-select first container
  useEffect(() => {
    if (containers.length > 0 && !container) {
      setContainer(containers[0])
    }
  }, [containers, container])

  return (
    <div className="flex flex-col h-full">
      {/* Controls bar - always shown for container selector, follow, refresh */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-700 bg-slate-800/50">
        <div className="flex items-center gap-3">
          {showHeader && (
            <>
              <Terminal className="w-4 h-4 text-slate-400" />
              <span className="text-sm font-medium text-white">{podName}</span>
            </>
          )}
          {containers.length > 1 && (
            <select
              value={container}
              onChange={(e) => setContainer(e.target.value)}
              className="text-xs bg-slate-700 border border-slate-600 rounded px-2 py-1 text-white"
            >
              {containers.map(c => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setFollow(!follow)}
            className={clsx(
              'px-2 py-1 text-xs rounded transition-colors',
              follow ? 'bg-blue-500 text-white' : 'bg-slate-700 text-slate-400 hover:text-white'
            )}
          >
            {follow ? 'Following' : 'Follow'}
          </button>
          <button
            onClick={() => refetch()}
            className="p-1 text-slate-400 hover:text-white"
            title="Refresh"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          {showHeader && (
            <button
              onClick={onClose}
              className="p-1 text-slate-400 hover:text-white"
            >
              ×
            </button>
          )}
        </div>
      </div>
      <pre
        ref={logsRef}
        className="flex-1 overflow-auto p-4 text-xs font-mono text-slate-300 bg-slate-950"
      >
        {isLoading ? (
          <span className="text-slate-500">Loading logs...</span>
        ) : logs ? (
          logs
        ) : (
          <span className="text-slate-500">No logs available</span>
        )}
      </pre>
    </div>
  )
}

function EventsTab({
  events,
  isLoading,
  showRoutine,
  routineCount,
  onToggleRoutine,
  resourceKind,
  resourceName,
}: {
  events: TimelineEvent[]
  isLoading: boolean
  showRoutine: boolean
  routineCount: number
  onToggleRoutine: (show: boolean) => void
  resourceKind: string
  resourceName: string
}) {
  const [expandedEvents, setExpandedEvents] = useState<Set<string>>(new Set())

  const toggleEvent = (id: string) => {
    setExpandedEvents(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-500">
        <RefreshCw className="w-5 h-5 animate-spin mr-2" />
        Loading events...
      </div>
    )
  }

  // Group events by day
  const groupedEvents = useMemo(() => {
    const groups: { date: string; events: TimelineEvent[] }[] = []
    let currentDate = ''

    for (const event of events) {
      const date = new Date(event.timestamp).toLocaleDateString()
      if (date !== currentDate) {
        currentDate = date
        groups.push({ date, events: [] })
      }
      groups[groups.length - 1].events.push(event)
    }
    return groups
  }, [events])

  return (
    <div className="h-full flex flex-col">
      {/* Filter bar */}
      {routineCount > 0 && (
        <div className="px-4 py-2 border-b border-slate-700 flex items-center justify-end">
          <label className="flex items-center gap-2 text-xs text-slate-400 cursor-pointer hover:text-slate-300">
            <input
              type="checkbox"
              checked={showRoutine}
              onChange={(e) => onToggleRoutine(e.target.checked)}
              className="w-3.5 h-3.5 rounded border-slate-600 bg-slate-700 text-blue-500"
            />
            {showRoutine ? <Eye className="w-3.5 h-3.5" /> : <EyeOff className="w-3.5 h-3.5" />}
            Show routine events ({routineCount})
          </label>
        </div>
      )}

      {/* Events list */}
      <div className="flex-1 overflow-auto">
        {events.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-slate-500">
            <Clock className="w-12 h-12 mb-4 opacity-50" />
            <p className="text-lg">No events</p>
            <p className="text-sm">Events will appear here as things change</p>
          </div>
        ) : (
          <div className="divide-y divide-slate-700/30">
            {groupedEvents.map(group => (
              <div key={group.date}>
                <div className="sticky top-0 bg-slate-900/95 backdrop-blur px-4 py-2 text-xs font-medium text-slate-500 border-b border-slate-700/30">
                  {group.date}
                </div>
                <div>
                  {group.events.map(event => {
                    const isOwn = event.kind === resourceKind && event.name === resourceName
                    const isExpanded = expandedEvents.has(event.id)
                    const hasDiff = event.diff && event.diff.fields.length > 0

                    return (
                      <div key={event.id}>
                        <button
                          onClick={() => hasDiff && toggleEvent(event.id)}
                          className={clsx(
                            'w-full px-4 py-3 flex items-start gap-4 text-left transition-colors',
                            isExpanded ? 'bg-slate-800/50' : 'hover:bg-slate-800/30',
                            !isOwn && 'pl-8',
                            !hasDiff && 'cursor-default'
                          )}
                        >
                          <EventIcon event={event} />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              {!isOwn && (
                                <span className={clsx(
                                  'text-xs px-1.5 py-0.5 rounded',
                                  event.kind === 'ReplicaSet' ? 'bg-violet-900/50 text-violet-400' :
                                  event.kind === 'Pod' ? 'bg-green-900/50 text-green-400' :
                                  'bg-slate-700 text-slate-400'
                                )}>
                                  {event.kind}
                                </span>
                              )}
                              <span className="text-sm font-medium text-white">
                                {event.type === 'change' ? event.operation : event.reason}
                              </span>
                              {event.healthState && event.healthState !== 'unknown' && (
                                <HealthBadge state={event.healthState} />
                              )}
                              {event.type === 'k8s_event' && (
                                <span className={clsx(
                                  'text-xs px-1.5 py-0.5 rounded',
                                  isProblematicEvent(event) ? 'bg-amber-500/20 text-amber-400' : 'bg-green-500/20 text-green-400'
                                )}>
                                  {event.eventType}
                                </span>
                              )}
                            </div>
                            {!isOwn && <p className="text-xs text-slate-500 mb-1">{event.name}</p>}
                            {event.diff?.summary && <p className="text-sm text-slate-400">{event.diff.summary}</p>}
                            {event.message && <p className="text-sm text-slate-400 line-clamp-2">{event.message}</p>}
                            <p className="text-xs text-slate-500 mt-1">{new Date(event.timestamp).toLocaleTimeString()}</p>
                          </div>
                          {hasDiff && (
                            <ChevronRight className={clsx('w-4 h-4 text-slate-500 transition-transform flex-shrink-0', isExpanded && 'rotate-90')} />
                          )}
                        </button>

                        {/* Inline diff - shown when expanded */}
                        {isExpanded && event.diff && (
                          <div className="px-4 pb-4 pl-16 bg-slate-800/30">
                            <DiffViewer diff={event.diff} />
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function EventIcon({ event }: { event: TimelineEvent }) {
  const isChange = event.type === 'change'
  const isProblematic = isProblematicEvent(event)

  const config = isChange ? {
    add: { bg: 'bg-green-500', icon: Plus },
    delete: { bg: 'bg-red-500', icon: Trash2 },
    update: { bg: 'bg-blue-500', icon: RefreshCw },
  }[event.operation || 'update'] : isProblematic ? {
    bg: 'bg-amber-500', icon: AlertCircle
  } : {
    bg: 'bg-slate-500', icon: CheckCircle
  }

  const Icon = config?.icon || Clock

  return (
    <div className={clsx('w-8 h-8 rounded-full flex items-center justify-center text-white flex-shrink-0', config?.bg || 'bg-slate-500')}>
      <Icon className="w-4 h-4" />
    </div>
  )
}

function YamlTab({ resource, isLoading }: { resource: any; isLoading: boolean }) {
  const [copied, setCopied] = useState(false)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-500">
        <RefreshCw className="w-5 h-5 animate-spin mr-2" />
        Loading...
      </div>
    )
  }

  const yaml = resource ? JSON.stringify(resource, null, 2) : ''

  const handleCopy = () => {
    navigator.clipboard.writeText(yaml)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="h-full flex flex-col">
      <div className="px-4 py-2 border-b border-slate-700 flex items-center justify-end">
        <button
          onClick={handleCopy}
          className="flex items-center gap-1.5 px-2 py-1 text-xs text-slate-400 hover:text-white hover:bg-slate-700 rounded"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
      <pre className="flex-1 overflow-auto p-4 text-xs font-mono text-slate-300 bg-slate-950">
        {yaml || 'No data available'}
      </pre>
    </div>
  )
}
