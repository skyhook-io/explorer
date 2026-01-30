package traffic

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	flowpb "github.com/cilium/cilium/api/v1/flow"
	observerpb "github.com/cilium/cilium/api/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	hubbleRelayService   = "hubble-relay"
	hubbleRelayNamespace = "kube-system"
	hubbleRelayGRPCPort  = 80 // Hubble Relay gRPC port (mapped from 4245)
)

// HubbleSource implements TrafficSource for Hubble/Cilium
type HubbleSource struct {
	k8sClient      kubernetes.Interface
	grpcConn       *grpc.ClientConn
	observerClient observerpb.ObserverClient
	localPort      int    // Port-forward local port
	currentContext string // K8s context for port-forward validation
	relayPort      int    // Hubble relay service port
	isConnected    bool
	mu             sync.RWMutex
}

// NewHubbleSource creates a new Hubble traffic source
func NewHubbleSource(client kubernetes.Interface) *HubbleSource {
	return &HubbleSource{
		k8sClient: client,
	}
}

// Name returns the source identifier
func (h *HubbleSource) Name() string {
	return "hubble"
}

// Detect checks if Hubble is available in the cluster
func (h *HubbleSource) Detect(ctx context.Context) (*DetectionResult, error) {
	result := &DetectionResult{
		Available: false,
	}

	// Step 1: Check for Cilium ConfigMap (indicates Cilium is installed)
	ciliumConfig, err := h.k8sClient.CoreV1().ConfigMaps(hubbleRelayNamespace).Get(ctx, "cilium-config", metav1.GetOptions{})
	hasCilium := err == nil

	// Check if Hubble is enabled in Cilium config
	hubbleEnabled := false
	if hasCilium && ciliumConfig.Data != nil {
		hubbleEnabled = ciliumConfig.Data["enable-hubble"] == "true"
	}

	// Step 2: Check for Hubble Relay pods
	relayPods, err := h.k8sClient.CoreV1().Pods(hubbleRelayNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=hubble-relay",
	})
	hasRelayPods := err == nil && len(relayPods.Items) > 0

	// Count running pods
	runningPods := 0
	if hasRelayPods {
		for _, pod := range relayPods.Items {
			if pod.Status.Phase == "Running" {
				runningPods++
			}
		}
	}

	// Step 3: Check for Hubble Relay service
	relaySvc, err := h.k8sClient.CoreV1().Services(hubbleRelayNamespace).Get(ctx, hubbleRelayService, metav1.GetOptions{})
	hasRelayService := err == nil

	// Step 4: Determine status
	isNative := h.isNativeHubble(ctx)

	if !hasCilium {
		result.Message = "Cilium CNI not detected. Install Cilium with Hubble for traffic visibility."
		return result, nil
	}

	if !hubbleEnabled {
		result.Message = "Cilium is installed but Hubble is not enabled. Enable Hubble observability."
		if isNative {
			result.Message += " For GKE, run: gcloud container clusters update CLUSTER --enable-dataplane-v2-observability"
		} else {
			result.Message += " Run: cilium hubble enable"
		}
		return result, nil
	}

	if !hasRelayPods {
		result.Message = "Hubble is enabled but Hubble Relay pods not found. The Relay may still be deploying."
		return result, nil
	}

	if runningPods == 0 {
		result.Message = fmt.Sprintf("Hubble Relay pods exist (%d) but none are running", len(relayPods.Items))
		return result, nil
	}

	if !hasRelayService {
		result.Message = "Hubble Relay pods are running but service not exposed"
		return result, nil
	}

	// All checks passed - Hubble is available
	h.mu.Lock()
	// Resolve the actual container port (may differ from service port due to named targetPort)
	h.relayPort = h.resolveTargetPort(ctx, relaySvc)
	h.mu.Unlock()

	result.Available = true
	result.Native = isNative
	result.Message = fmt.Sprintf("Hubble Relay detected with %d running pod(s)", runningPods)

	// Try to get version from Cilium config
	if ciliumConfig.Labels != nil {
		if ver, ok := ciliumConfig.Labels["cilium.io/version"]; ok {
			result.Version = ver
		}
	}

	return result, nil
}

// isNativeHubble checks if this is GKE Dataplane V2 (native Hubble)
func (h *HubbleSource) isNativeHubble(ctx context.Context) bool {
	// Check for GKE by looking at node provider ID
	nodes, err := h.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil || len(nodes.Items) == 0 {
		return false
	}

	node := nodes.Items[0]

	// GKE nodes have gce:// provider ID
	if strings.HasPrefix(node.Spec.ProviderID, "gce://") {
		// Check for Dataplane V2 specific labels or annotations
		if _, ok := node.Labels["cloud.google.com/gke-nodepool"]; ok {
			return true
		}
	}

	return false
}

// resolveTargetPort resolves the actual container port from the service
// The service may use a named targetPort (e.g., "grpc") that maps to a container port
func (h *HubbleSource) resolveTargetPort(ctx context.Context, svc *corev1.Service) int {
	if len(svc.Spec.Ports) == 0 {
		return hubbleRelayGRPCPort
	}

	svcPort := svc.Spec.Ports[0]

	// If targetPort is a number, use it directly
	if svcPort.TargetPort.IntValue() > 0 {
		return svcPort.TargetPort.IntValue()
	}

	// If targetPort is a named port, we need to find the actual port from pods
	if svcPort.TargetPort.StrVal != "" {
		// Find a pod backing this service
		if svc.Spec.Selector != nil {
			var labelSelector string
			for k, v := range svc.Spec.Selector {
				if labelSelector != "" {
					labelSelector += ","
				}
				labelSelector += k + "=" + v
			}

			pods, err := h.k8sClient.CoreV1().Pods(svc.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
				Limit:         1,
			})
			if err == nil && len(pods.Items) > 0 {
				pod := pods.Items[0]
				for _, container := range pod.Spec.Containers {
					for _, port := range container.Ports {
						if port.Name == svcPort.TargetPort.StrVal {
							log.Printf("[hubble] Resolved named port %q to %d", svcPort.TargetPort.StrVal, port.ContainerPort)
							return int(port.ContainerPort)
						}
					}
				}
			}
		}
	}

	// Fallback to service port or default
	if svcPort.Port > 0 {
		return int(svcPort.Port)
	}
	return hubbleRelayGRPCPort
}

// Connect establishes connection to Hubble Relay via port-forward and gRPC
func (h *HubbleSource) Connect(ctx context.Context, contextName string) (*MetricsConnectionInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If already connected to the same context, verify connection is still valid
	if h.grpcConn != nil && h.currentContext == contextName {
		// Test the connection
		if h.testConnection(ctx) {
			return &MetricsConnectionInfo{
				Connected:   true,
				LocalPort:   h.localPort,
				Address:     fmt.Sprintf("localhost:%d", h.localPort),
				Namespace:   hubbleRelayNamespace,
				ServiceName: hubbleRelayService,
				ContextName: contextName,
			}, nil
		}
		// Connection lost, clean up
		h.closeConnectionLocked()
	}

	// Clear stale state if context changed
	if h.currentContext != contextName {
		h.closeConnectionLocked()
		h.currentContext = contextName
	}

	// Get the relay port from detection if not already set
	if h.relayPort == 0 {
		relaySvc, err := h.k8sClient.CoreV1().Services(hubbleRelayNamespace).Get(ctx, hubbleRelayService, metav1.GetOptions{})
		if err != nil {
			return &MetricsConnectionInfo{
				Connected: false,
				Error:     fmt.Sprintf("Hubble Relay service not found: %v", err),
			}, nil
		}
		// Get the actual target port (container port), not the service port
		// The service may map port 80 -> containerPort 4245 (named "grpc")
		h.relayPort = h.resolveTargetPort(ctx, relaySvc)
	}

	// Start port-forward to Hubble Relay
	log.Printf("[hubble] Starting port-forward to %s/%s:%d", hubbleRelayNamespace, hubbleRelayService, h.relayPort)
	connInfo, err := StartMetricsPortForward(ctx, hubbleRelayNamespace, hubbleRelayService, h.relayPort, contextName)
	if err != nil {
		return &MetricsConnectionInfo{
			Connected:   false,
			Namespace:   hubbleRelayNamespace,
			ServiceName: hubbleRelayService,
			Error:       fmt.Sprintf("Failed to start port-forward: %v", err),
		}, nil
	}

	if !connInfo.Connected {
		return connInfo, nil
	}

	h.localPort = connInfo.LocalPort

	// Create gRPC connection
	grpcAddr := fmt.Sprintf("localhost:%d", h.localPort)
	log.Printf("[hubble] Connecting to gRPC at %s", grpcAddr)

	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		// Clean up port-forward on gRPC connection failure
		StopMetricsPortForward()
		h.localPort = 0
		return &MetricsConnectionInfo{
			Connected: false,
			Error:     fmt.Sprintf("Failed to create gRPC connection: %v", err),
		}, nil
	}

	h.grpcConn = conn
	h.observerClient = observerpb.NewObserverClient(conn)
	h.isConnected = true

	// Test the connection
	if !h.testConnection(ctx) {
		h.closeConnectionLocked()
		// Also stop port-forward on connection test failure
		StopMetricsPortForward()
		return &MetricsConnectionInfo{
			Connected: false,
			Error:     "Failed to connect to Hubble Relay gRPC service",
		}, nil
	}

	log.Printf("[hubble] Connected to Hubble Relay at %s", grpcAddr)

	return &MetricsConnectionInfo{
		Connected:   true,
		LocalPort:   h.localPort,
		Address:     grpcAddr,
		Namespace:   hubbleRelayNamespace,
		ServiceName: hubbleRelayService,
		ContextName: contextName,
	}, nil
}

// testConnection tests the gRPC connection by calling ServerStatus
func (h *HubbleSource) testConnection(ctx context.Context) bool {
	if h.observerClient == nil {
		return false
	}

	testCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := h.observerClient.ServerStatus(testCtx, &observerpb.ServerStatusRequest{})
	if err != nil {
		log.Printf("[hubble] Connection test failed: %v", err)
		return false
	}
	return true
}

// closeConnectionLocked closes the gRPC connection (caller must hold lock)
func (h *HubbleSource) closeConnectionLocked() {
	if h.grpcConn != nil {
		h.grpcConn.Close()
		h.grpcConn = nil
	}
	h.observerClient = nil
	h.isConnected = false
	h.localPort = 0
}

// GetFlows retrieves flows from Hubble via gRPC
func (h *HubbleSource) GetFlows(ctx context.Context, opts FlowOptions) (*FlowsResponse, error) {
	h.mu.RLock()
	client := h.observerClient
	connected := h.isConnected
	h.mu.RUnlock()

	if !connected || client == nil {
		// Not connected yet - return empty with message
		return &FlowsResponse{
			Source:    "hubble",
			Timestamp: time.Now(),
			Flows:     []Flow{},
			Warning:   "Not connected to Hubble Relay. Call Connect() first or use the Traffic view to establish connection.",
		}, nil
	}

	flows, err := h.fetchFlowsViaGRPC(ctx, opts)
	if err != nil {
		log.Printf("[hubble] gRPC error: %v", err)
		return &FlowsResponse{
			Source:    "hubble",
			Timestamp: time.Now(),
			Flows:     []Flow{},
			Warning:   fmt.Sprintf("Failed to fetch flows: %v", err),
		}, nil
	}

	return &FlowsResponse{
		Source:    "hubble",
		Timestamp: time.Now(),
		Flows:     flows,
	}, nil
}

// fetchFlowsViaGRPC fetches flows using gRPC client
func (h *HubbleSource) fetchFlowsViaGRPC(ctx context.Context, opts FlowOptions) ([]Flow, error) {
	h.mu.RLock()
	client := h.observerClient
	h.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected to Hubble Relay")
	}

	// Build request
	req := &observerpb.GetFlowsRequest{
		Number: 1000, // Default limit
		Follow: false,
	}

	if opts.Limit > 0 {
		req.Number = uint64(opts.Limit)
	}

	// Add namespace filter if specified
	// Use separate filters for source OR destination (each filter is AND within itself,
	// but multiple filters are OR'd together)
	if opts.Namespace != "" {
		req.Whitelist = []*flowpb.FlowFilter{
			{SourcePod: []string{opts.Namespace + "/"}},
			{DestinationPod: []string{opts.Namespace + "/"}},
		}
	}

	// Add time filter based on Since
	if opts.Since > 0 {
		since := time.Now().Add(-opts.Since)
		req.Since = timestamppb.New(since)
	}

	// Create context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := client.GetFlows(reqCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get flows stream: %w", err)
	}

	var flows []Flow
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Check if we got any flows before the error
			if len(flows) > 0 {
				log.Printf("[hubble] Stream ended with partial results: %v", err)
				break
			}
			return nil, fmt.Errorf("stream error: %w", err)
		}

		// Extract flow from response
		pbFlow := resp.GetFlow()
		if pbFlow == nil {
			continue
		}

		flow := convertHubbleFlow(pbFlow)
		flows = append(flows, flow)
	}

	log.Printf("[hubble] Retrieved %d flows", len(flows))
	return flows, nil
}

// convertHubbleFlow converts a Hubble protobuf Flow to our internal Flow type
func convertHubbleFlow(pbFlow *flowpb.Flow) Flow {
	// Extract IP addresses safely (IP may be nil for some flow types)
	var srcIP, dstIP string
	if ip := pbFlow.GetIP(); ip != nil {
		srcIP = ip.GetSource()
		dstIP = ip.GetDestination()
	}

	flow := Flow{
		Source:      convertEndpoint(pbFlow.GetSource(), srcIP),
		Destination: convertEndpoint(pbFlow.GetDestination(), dstIP),
		Verdict:     strings.ToLower(pbFlow.GetVerdict().String()),
		Connections: 1,
	}

	// Extract L4 info
	l4 := pbFlow.GetL4()
	if l4 != nil {
		if tcp := l4.GetTCP(); tcp != nil {
			flow.Protocol = "tcp"
			flow.Port = int(tcp.GetDestinationPort())
		} else if udp := l4.GetUDP(); udp != nil {
			flow.Protocol = "udp"
			flow.Port = int(udp.GetDestinationPort())
		} else if icmpv4 := l4.GetICMPv4(); icmpv4 != nil {
			flow.Protocol = "icmp"
		} else if icmpv6 := l4.GetICMPv6(); icmpv6 != nil {
			flow.Protocol = "icmpv6"
		} else if sctp := l4.GetSCTP(); sctp != nil {
			flow.Protocol = "sctp"
			flow.Port = int(sctp.GetDestinationPort())
		}
	}

	// Extract L7 info if available
	l7 := pbFlow.GetL7()
	if l7 != nil {
		if http := l7.GetHttp(); http != nil {
			flow.L7Protocol = "HTTP"
			flow.HTTPMethod = http.GetMethod()
			flow.HTTPPath = http.GetUrl()
			flow.HTTPStatus = int(http.GetCode())
		} else if dns := l7.GetDns(); dns != nil {
			flow.L7Protocol = "DNS"
		}
	}

	// Parse timestamp
	if ts := pbFlow.GetTime(); ts != nil {
		flow.LastSeen = ts.AsTime()
	} else {
		flow.LastSeen = time.Now()
	}

	return flow
}

// convertEndpoint converts a Hubble Endpoint to our internal Endpoint type
func convertEndpoint(ep *flowpb.Endpoint, ip string) Endpoint {
	if ep == nil {
		return Endpoint{
			Kind: "External",
			IP:   ip,
			Name: ip,
		}
	}

	endpoint := Endpoint{
		Namespace: ep.GetNamespace(),
		IP:        ip,
	}

	// Determine the name and kind
	if podName := ep.GetPodName(); podName != "" {
		endpoint.Name = podName
		endpoint.Kind = "Pod"
	} else if ep.GetIdentity() != 0 {
		// Use identity for reserved labels (like host, world, etc.)
		labels := ep.GetLabels()
		for _, label := range labels {
			if strings.HasPrefix(label, "reserved:") {
				endpoint.Kind = "External"
				endpoint.Name = strings.TrimPrefix(label, "reserved:")
				break
			}
		}
		if endpoint.Name == "" {
			endpoint.Kind = "External"
			endpoint.Name = ip
		}
	} else {
		endpoint.Kind = "External"
		endpoint.Name = ip
	}

	// Extract workload name from labels
	endpoint.Workload = extractWorkloadFromHubbleLabels(ep.GetLabels())

	return endpoint
}

// extractWorkloadFromHubbleLabels extracts workload name from Hubble labels
func extractWorkloadFromHubbleLabels(labels []string) string {
	labelMap := make(map[string]string)
	for _, l := range labels {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 2 {
			labelMap[parts[0]] = parts[1]
		}
	}

	// Common workload labels in order of preference
	for _, key := range []string{"app", "app.kubernetes.io/name", "k8s-app", "name"} {
		if name, ok := labelMap[key]; ok {
			return name
		}
	}

	return ""
}

// StreamFlows returns a channel of flows for real-time updates
func (h *HubbleSource) StreamFlows(ctx context.Context, opts FlowOptions) (<-chan Flow, error) {
	flowCh := make(chan Flow, 100)

	go func() {
		defer close(flowCh)

		h.mu.RLock()
		client := h.observerClient
		h.mu.RUnlock()

		if client == nil {
			log.Printf("[hubble] Cannot stream: not connected")
			return
		}

		// Build streaming request
		req := &observerpb.GetFlowsRequest{
			Follow: true,
		}

		if opts.Namespace != "" {
			req.Whitelist = []*flowpb.FlowFilter{
				{SourcePod: []string{opts.Namespace + "/"}},
				{DestinationPod: []string{opts.Namespace + "/"}},
			}
		}

		stream, err := client.GetFlows(ctx, req)
		if err != nil {
			log.Printf("[hubble] Failed to start flow stream: %v", err)
			return
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				if ctx.Err() != nil {
					return // Context cancelled
				}
				log.Printf("[hubble] Stream error: %v", err)
				return
			}

			pbFlow := resp.GetFlow()
			if pbFlow == nil {
				continue
			}

			flow := convertHubbleFlow(pbFlow)

			select {
			case flowCh <- flow:
			case <-ctx.Done():
				return
			default:
				// Channel full, drop flow
			}
		}
	}()

	return flowCh, nil
}

// Close cleans up resources
func (h *HubbleSource) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closeConnectionLocked()
	h.currentContext = ""
	return nil
}

// GetPortForwardInstructions returns kubectl commands for manual access
func (h *HubbleSource) GetPortForwardInstructions() string {
	return `To access Hubble flows directly, run:

# Port-forward Hubble Relay (gRPC API)
kubectl -n kube-system port-forward svc/hubble-relay 4245:80

# Then use Hubble CLI:
hubble observe --server localhost:4245

# Or port-forward Hubble UI:
kubectl -n kube-system port-forward svc/hubble-ui 12000:80
# Then open http://localhost:12000`
}
