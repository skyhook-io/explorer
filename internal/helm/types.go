package helm

import (
	"time"
)

// HelmRelease represents a Helm release in the list view
type HelmRelease struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Chart        string    `json:"chart"`
	ChartVersion string    `json:"chartVersion"`
	AppVersion   string    `json:"appVersion"`
	Status       string    `json:"status"`
	Revision     int       `json:"revision"`
	Updated      time.Time `json:"updated"`
}

// HelmRevision represents a single revision in the release history
type HelmRevision struct {
	Revision    int       `json:"revision"`
	Status      string    `json:"status"`
	Chart       string    `json:"chart"`
	AppVersion  string    `json:"appVersion"`
	Description string    `json:"description"`
	Updated     time.Time `json:"updated"`
}

// HelmReleaseDetail contains full details of a Helm release
type HelmReleaseDetail struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Chart        string            `json:"chart"`
	ChartVersion string            `json:"chartVersion"`
	AppVersion   string            `json:"appVersion"`
	Status       string            `json:"status"`
	Revision     int               `json:"revision"`
	Updated      time.Time         `json:"updated"`
	Description  string            `json:"description"`
	Notes        string            `json:"notes"`
	History      []HelmRevision    `json:"history"`
	Resources    []OwnedResource   `json:"resources"`
	Hooks        []HelmHook        `json:"hooks,omitempty"`
	Readme       string            `json:"readme,omitempty"`
	Dependencies []ChartDependency `json:"dependencies,omitempty"`
}

// HelmHook represents a Helm hook (pre/post install, upgrade, etc.)
type HelmHook struct {
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	Events []string `json:"events"`
	Weight int      `json:"weight"`
	Status string   `json:"status,omitempty"`
}

// ChartDependency represents a chart dependency
type ChartDependency struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository,omitempty"`
	Condition  string `json:"condition,omitempty"`
	Enabled    bool   `json:"enabled"`
}

// OwnedResource represents a K8s resource created by a Helm release
type OwnedResource struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status,omitempty"`    // Running, Pending, Failed, etc.
	Ready     string `json:"ready,omitempty"`     // e.g., "3/3" for deployments
	Message   string `json:"message,omitempty"`   // Status message or reason
}

// HelmValues represents the values for a release
type HelmValues struct {
	UserSupplied map[string]any `json:"userSupplied"`
	Computed     map[string]any `json:"computed,omitempty"`
}

// ManifestDiff represents a diff between two revisions
type ManifestDiff struct {
	Revision1 int    `json:"revision1"`
	Revision2 int    `json:"revision2"`
	Diff      string `json:"diff"`
}

// UpgradeInfo contains information about available upgrades
type UpgradeInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	RepositoryName  string `json:"repositoryName,omitempty"`
	Error           string `json:"error,omitempty"`
}

// BatchUpgradeInfo contains upgrade info for multiple releases
type BatchUpgradeInfo struct {
	// Map of "namespace/name" to UpgradeInfo
	Releases map[string]*UpgradeInfo `json:"releases"`
}
