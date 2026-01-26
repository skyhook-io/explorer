package k8s

import (
	"context"
	"sync"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Capabilities represents the features available based on RBAC permissions
type Capabilities struct {
	Exec        bool `json:"exec"`        // Can create pods/exec (terminal feature)
	Logs        bool `json:"logs"`        // Can get pods/log (log viewer)
	PortForward bool `json:"portForward"` // Can create pods/portforward
	Secrets     bool `json:"secrets"`     // Can list secrets
}

var (
	cachedCapabilities *Capabilities
	capabilitiesMu     sync.RWMutex
	capabilitiesExpiry time.Time
	capabilitiesTTL    = 60 * time.Second
)

// CheckCapabilities checks RBAC permissions using SelfSubjectAccessReview
// Results are cached for 60 seconds to avoid hammering the API
func CheckCapabilities(ctx context.Context) (*Capabilities, error) {
	capabilitiesMu.RLock()
	if cachedCapabilities != nil && time.Now().Before(capabilitiesExpiry) {
		caps := *cachedCapabilities
		capabilitiesMu.RUnlock()
		return &caps, nil
	}
	capabilitiesMu.RUnlock()

	// Need to refresh capabilities
	capabilitiesMu.Lock()
	defer capabilitiesMu.Unlock()

	// Double-check after acquiring write lock
	if cachedCapabilities != nil && time.Now().Before(capabilitiesExpiry) {
		caps := *cachedCapabilities
		return &caps, nil
	}

	if GetClient() == nil {
		// Return all true if client not initialized (shouldn't happen)
		return &Capabilities{Exec: true, Logs: true, PortForward: true, Secrets: true}, nil
	}

	caps := &Capabilities{}

	// Check each capability in parallel
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		caps.Exec = canI(ctx, "", "pods/exec", "create")
	}()

	go func() {
		defer wg.Done()
		caps.Logs = canI(ctx, "", "pods/log", "get")
	}()

	go func() {
		defer wg.Done()
		caps.PortForward = canI(ctx, "", "pods/portforward", "create")
	}()

	go func() {
		defer wg.Done()
		caps.Secrets = canI(ctx, "", "secrets", "list")
	}()

	wg.Wait()

	// Cache the result
	cachedCapabilities = caps
	capabilitiesExpiry = time.Now().Add(capabilitiesTTL)

	return caps, nil
}

// canI checks if the current user/service account can perform an action
func canI(ctx context.Context, namespace, resource, verb string) bool {
	k8sClient := GetClient()
	if k8sClient == nil {
		return true // Assume allowed if no client
	}

	review := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace, // Empty = cluster-wide
				Verb:      verb,
				Resource:  resource,
			},
		},
	}

	result, err := k8sClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		// If we can't check, assume not allowed (fail closed)
		return false
	}

	return result.Status.Allowed
}

// InvalidateCapabilitiesCache forces the next CheckCapabilities call to refresh
func InvalidateCapabilitiesCache() {
	capabilitiesMu.Lock()
	defer capabilitiesMu.Unlock()
	cachedCapabilities = nil
}
