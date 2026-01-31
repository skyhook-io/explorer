package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/skyhook-io/radar/internal/k8s"
)

// ArgoCD API groups and versions
var (
	argoApplicationGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}
	argoApplicationSetGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applicationsets",
	}
	argoAppProjectGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "appprojects",
	}
)

// getArgoGVR returns the appropriate GVR for an ArgoCD resource kind
func getArgoGVR(kind string) (schema.GroupVersionResource, error) {
	switch strings.ToLower(kind) {
	case "application", "applications":
		return argoApplicationGVR, nil
	case "applicationset", "applicationsets":
		return argoApplicationSetGVR, nil
	case "appproject", "appprojects":
		return argoAppProjectGVR, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown ArgoCD resource kind: %s", kind)
	}
}

// handleArgoSync triggers a sync operation on an ArgoCD Application
// This sets the operation field to initiate a sync
func (s *Server) handleArgoSync(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	client := k8s.GetDynamicClient()
	if client == nil {
		s.writeError(w, http.StatusServiceUnavailable, "dynamic client not available")
		return
	}

	// First, get the current application to check its state
	app, err := client.Resource(argoApplicationGVR).Namespace(namespace).Get(
		r.Context(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if there's already an operation in progress
	if _, found := unstructuredNestedField(app.Object, "status", "operationState", "phase"); found {
		phase, _ := unstructuredNestedString(app.Object, "status", "operationState", "phase")
		if phase == "Running" {
			s.writeError(w, http.StatusConflict, "sync operation already in progress")
			return
		}
	}

	// ArgoCD sync is triggered by setting the operation field
	// The argocd-application-controller watches for this and performs the sync
	timestamp := time.Now().Format(time.RFC3339Nano)

	// We use a simpler approach: set the refresh annotation to trigger a sync
	// This is similar to running `argocd app sync`
	patch := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				"argocd.argoproj.io/refresh": "hard",
			},
		},
		"operation": map[string]any{
			"initiatedBy": map[string]any{
				"username": "radar",
			},
			"sync": map[string]any{
				"revision": "", // Empty means use the target revision from spec
				"prune":    true,
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create patch")
		return
	}

	_, err = client.Resource(argoApplicationGVR).Namespace(namespace).Patch(
		r.Context(),
		name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		log.Printf("[argo] Failed to sync application %s/%s: %v", namespace, name, err)
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, map[string]string{
		"message":     "Sync operation initiated",
		"requestedAt": timestamp,
	})
}

// handleArgoRefresh triggers a refresh (re-read from git) on an ArgoCD Application
// This is a lighter operation than sync - it just refreshes the app status
func (s *Server) handleArgoRefresh(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// Get refresh type from query param (default: normal, can be "hard")
	refreshType := r.URL.Query().Get("type")
	if refreshType == "" {
		refreshType = "normal"
	} else if refreshType != "normal" && refreshType != "hard" {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid refresh type %q: must be 'normal' or 'hard'", refreshType))
		return
	}

	client := k8s.GetDynamicClient()
	if client == nil {
		s.writeError(w, http.StatusServiceUnavailable, "dynamic client not available")
		return
	}

	timestamp := time.Now().Format(time.RFC3339Nano)

	// ArgoCD refresh is triggered by setting the refresh annotation
	patch := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				"argocd.argoproj.io/refresh": refreshType,
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create patch")
		return
	}

	_, err = client.Resource(argoApplicationGVR).Namespace(namespace).Patch(
		r.Context(),
		name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		log.Printf("[argo] Failed to refresh application %s/%s: %v", namespace, name, err)
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, map[string]string{
		"message":     fmt.Sprintf("Refresh (%s) triggered", refreshType),
		"requestedAt": timestamp,
	})
}

// handleArgoTerminate terminates an ongoing sync operation
func (s *Server) handleArgoTerminate(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	client := k8s.GetDynamicClient()
	if client == nil {
		s.writeError(w, http.StatusServiceUnavailable, "dynamic client not available")
		return
	}

	// First, check if there's an operation in progress
	app, err := client.Resource(argoApplicationGVR).Namespace(namespace).Get(
		r.Context(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check the operation state
	phase, found := unstructuredNestedString(app.Object, "status", "operationState", "phase")
	if !found || phase != "Running" {
		s.writeError(w, http.StatusBadRequest, "no sync operation in progress")
		return
	}

	// Terminate by removing the operation field - ArgoCD will cancel it
	// Actually, ArgoCD termination is done by setting operation to nil
	// We use a JSON patch to remove the operation field
	patchBytes := []byte(`[{"op": "remove", "path": "/operation"}]`)

	_, err = client.Resource(argoApplicationGVR).Namespace(namespace).Patch(
		r.Context(),
		name,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		// If the operation field doesn't exist, the operation may have already completed
		if strings.Contains(err.Error(), "nonexistent") {
			log.Printf("[argo] Terminate: operation field already removed for %s/%s (may have completed)", namespace, name)
		} else {
			log.Printf("[argo] Failed to terminate operation for %s/%s: %v", namespace, name, err)
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	s.writeJSON(w, map[string]string{
		"message": "Sync operation terminated",
	})
}

// handleArgoSuspend disables automated sync on an ArgoCD Application
// ArgoCD doesn't have a direct suspend like Flux, but we can disable automated sync
func (s *Server) handleArgoSuspend(w http.ResponseWriter, r *http.Request) {
	s.setArgoAutomatedSync(w, r, false)
}

// handleArgoResume re-enables automated sync on an ArgoCD Application
func (s *Server) handleArgoResume(w http.ResponseWriter, r *http.Request) {
	s.setArgoAutomatedSync(w, r, true)
}

// setArgoAutomatedSync enables or disables automated sync policy
func (s *Server) setArgoAutomatedSync(w http.ResponseWriter, r *http.Request, enable bool) {
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	client := k8s.GetDynamicClient()
	if client == nil {
		s.writeError(w, http.StatusServiceUnavailable, "dynamic client not available")
		return
	}

	// Get current application to check existing sync policy
	app, err := client.Resource(argoApplicationGVR).Namespace(namespace).Get(
		r.Context(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var patch map[string]any

	if enable {
		// Re-enable automated sync
		// Get the existing prune and selfHeal settings if they exist
		prune := true
		selfHeal := true

		// Try to get existing settings from annotations (we store them when suspending)
		annotations, _ := unstructuredNestedStringMap(app.Object, "metadata", "annotations")
		if annotations != nil {
			if v, ok := annotations["radar.skyhook.io/suspended-prune"]; ok {
				prune = v == "true"
			}
			if v, ok := annotations["radar.skyhook.io/suspended-selfheal"]; ok {
				selfHeal = v == "true"
			}
		}

		patch = map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"radar.skyhook.io/suspended-prune":    nil, // Remove
					"radar.skyhook.io/suspended-selfheal": nil, // Remove
				},
			},
			"spec": map[string]any{
				"syncPolicy": map[string]any{
					"automated": map[string]any{
						"prune":    prune,
						"selfHeal": selfHeal,
					},
				},
			},
		}
	} else {
		// Disable automated sync (suspend)
		// First, save current automated settings to annotations for later restore
		prune := false
		selfHeal := false

		if automated, found := unstructuredNestedMap(app.Object, "spec", "syncPolicy", "automated"); found && automated != nil {
			if v, ok := automated["prune"].(bool); ok {
				prune = v
			}
			if v, ok := automated["selfHeal"].(bool); ok {
				selfHeal = v
			}
		}

		patch = map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]string{
					"radar.skyhook.io/suspended-prune":    fmt.Sprintf("%v", prune),
					"radar.skyhook.io/suspended-selfheal": fmt.Sprintf("%v", selfHeal),
				},
			},
			"spec": map[string]any{
				"syncPolicy": map[string]any{
					"automated": nil, // Remove automated sync
				},
			},
		}
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create patch")
		return
	}

	_, err = client.Resource(argoApplicationGVR).Namespace(namespace).Patch(
		r.Context(),
		name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		action := "suspend"
		if enable {
			action = "resume"
		}
		log.Printf("[argo] Failed to %s application %s/%s: %v", action, namespace, name, err)
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	action := "suspended"
	if enable {
		action = "resumed"
	}

	s.writeJSON(w, map[string]string{
		"message": fmt.Sprintf("Application %s (automated sync %s)", action, action),
	})
}

// Helper functions for unstructured access (similar to k8s.io/apimachinery unstructured helpers)

func unstructuredNestedField(obj map[string]any, fields ...string) (any, bool) {
	var val any = obj
	for _, field := range fields {
		if m, ok := val.(map[string]any); ok {
			val, ok = m[field]
			if !ok {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return val, true
}

func unstructuredNestedString(obj map[string]any, fields ...string) (string, bool) {
	val, found := unstructuredNestedField(obj, fields...)
	if !found {
		return "", false
	}
	if s, ok := val.(string); ok {
		return s, true
	}
	return "", false
}

func unstructuredNestedMap(obj map[string]any, fields ...string) (map[string]any, bool) {
	val, found := unstructuredNestedField(obj, fields...)
	if !found {
		return nil, false
	}
	if m, ok := val.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

func unstructuredNestedStringMap(obj map[string]any, fields ...string) (map[string]string, bool) {
	val, found := unstructuredNestedField(obj, fields...)
	if !found {
		return nil, false
	}
	if m, ok := val.(map[string]any); ok {
		result := make(map[string]string)
		for k, v := range m {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result, true
	}
	return nil, false
}
