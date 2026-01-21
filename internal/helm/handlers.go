package helm

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers provides HTTP handlers for Helm endpoints
type Handlers struct{}

// NewHandlers creates a new Handlers instance
func NewHandlers() *Handlers {
	return &Handlers{}
}

// RegisterRoutes registers Helm routes on the given router
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/helm", func(r chi.Router) {
		r.Get("/releases", h.handleListReleases)
		r.Get("/releases/{namespace}/{name}", h.handleGetRelease)
		r.Get("/releases/{namespace}/{name}/manifest", h.handleGetManifest)
		r.Get("/releases/{namespace}/{name}/values", h.handleGetValues)
		r.Get("/releases/{namespace}/{name}/diff", h.handleGetDiff)
		r.Get("/releases/{namespace}/{name}/upgrade-info", h.handleCheckUpgrade)
		r.Get("/upgrade-check", h.handleBatchUpgradeCheck)
		// Actions (write operations)
		r.Post("/releases/{namespace}/{name}/rollback", h.handleRollback)
		r.Post("/releases/{namespace}/{name}/upgrade", h.handleUpgrade)
		r.Delete("/releases/{namespace}/{name}", h.handleUninstall)
	})
}

// handleListReleases returns all Helm releases
func (h *Handlers) handleListReleases(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	releases, err := client.ListReleases(namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, releases)
}

// handleGetRelease returns details for a specific release
func (h *Handlers) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	release, err := client.GetRelease(namespace, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, release)
}

// handleGetManifest returns the rendered manifest for a release
func (h *Handlers) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// Optional revision parameter
	revision := 0
	if revStr := r.URL.Query().Get("revision"); revStr != "" {
		if rev, err := strconv.Atoi(revStr); err == nil {
			revision = rev
		}
	}

	manifest, err := client.GetManifest(namespace, name, revision)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return as plain text YAML
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(manifest))
}

// handleGetValues returns the values for a release
func (h *Handlers) handleGetValues(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")
	allValues := r.URL.Query().Get("all") == "true"

	values, err := client.GetValues(namespace, name, allValues)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, values)
}

// handleGetDiff returns the diff between two revisions
func (h *Handlers) handleGetDiff(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	rev1Str := r.URL.Query().Get("revision1")
	rev2Str := r.URL.Query().Get("revision2")

	if rev1Str == "" || rev2Str == "" {
		writeError(w, http.StatusBadRequest, "revision1 and revision2 parameters are required")
		return
	}

	rev1, err := strconv.Atoi(rev1Str)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid revision1 parameter")
		return
	}

	rev2, err := strconv.Atoi(rev2Str)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid revision2 parameter")
		return
	}

	diff, err := client.GetManifestDiff(namespace, name, rev1, rev2)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, diff)
}

// handleCheckUpgrade checks if a newer version is available
func (h *Handlers) handleCheckUpgrade(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	info, err := client.CheckForUpgrade(namespace, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, info)
}

// handleBatchUpgradeCheck checks all releases for upgrades at once
func (h *Handlers) handleBatchUpgradeCheck(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	info, err := client.BatchCheckUpgrades(namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, info)
}

// handleRollback rolls back a release to a previous revision
func (h *Handlers) handleRollback(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	revStr := r.URL.Query().Get("revision")
	if revStr == "" {
		writeError(w, http.StatusBadRequest, "revision parameter is required")
		return
	}

	revision, err := strconv.Atoi(revStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid revision parameter")
		return
	}

	if err := client.Rollback(namespace, name, revision); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, map[string]string{"status": "success", "message": "Rollback completed"})
}

// handleUninstall removes a release
func (h *Handlers) handleUninstall(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := client.Uninstall(namespace, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, map[string]string{"status": "success", "message": "Release uninstalled"})
}

// handleUpgrade upgrades a release to a new version
func (h *Handlers) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	client := GetClient()
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "Helm client not initialized")
		return
	}

	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	version := r.URL.Query().Get("version")
	if version == "" {
		writeError(w, http.StatusBadRequest, "version parameter is required")
		return
	}

	if err := client.Upgrade(namespace, name, version); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, map[string]string{"status": "success", "message": "Upgrade completed"})
}

// Helper functions

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
