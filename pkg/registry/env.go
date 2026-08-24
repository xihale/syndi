package registry

import (
	"os"
	"sort"
	"sync"
)

// EnvRequirement declares an environment variable (credential or option)
// that routes in a namespace may depend on, e.g. ZHIHU_COOKIES.
//
// Values are never stored here: only the key name and description. The
// configured state is resolved at render time via os.LookupEnv so the docs
// frontend always reflects the live process environment.
type EnvRequirement struct {
	// Key is the exact environment variable name, e.g. "ZHIHU_COOKIES".
	Key string
	// Description explains what the variable unlocks and where to get it.
	Description string
	// Scope describes coverage within the namespace,
	// e.g. "all routes" or "some routes (login-required ones)".
	Scope string
}

var (
	envMu         sync.RWMutex
	namespaceEnvs = map[string][]EnvRequirement{}
)

// RegisterNamespaceEnv declares env requirements for a namespace.
// Safe for concurrent use; typically called from package init().
func RegisterNamespaceEnv(namespace string, reqs ...EnvRequirement) {
	envMu.Lock()
	defer envMu.Unlock()
	namespaceEnvs[namespace] = append(namespaceEnvs[namespace], reqs...)
}

// NamespaceEnvReqs returns declared requirements for one namespace.
func NamespaceEnvReqs(namespace string) []EnvRequirement {
	envMu.RLock()
	defer envMu.RUnlock()
	out := make([]EnvRequirement, len(namespaceEnvs[namespace]))
	copy(out, namespaceEnvs[namespace])
	return out
}

// EnvStatus is an EnvRequirement plus its live configured state.
type EnvStatus struct {
	Namespace   string `json:"namespace"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	Configured  bool   `json:"configured"`
}

// AllEnvStatuses resolves every declared requirement against the current
// process environment, grouped by namespace in stable sorted order.
func AllEnvStatuses() map[string][]EnvStatus {
	envMu.RLock()
	namespaces := make([]string, 0, len(namespaceEnvs))
	for ns := range namespaceEnvs {
		namespaces = append(namespaces, ns)
	}
	envMu.RUnlock()
	sort.Strings(namespaces)

	result := make(map[string][]EnvStatus, len(namespaces))
	for _, ns := range namespaces {
		reqs := NamespaceEnvReqs(ns)
		statuses := make([]EnvStatus, 0, len(reqs))
		for _, r := range reqs {
			statuses = append(statuses, EnvStatus{
				Namespace:   ns,
				Key:         r.Key,
				Description: r.Description,
				Scope:       r.Scope,
				Configured:  envNonEmpty(r.Key),
			})
		}
		result[ns] = statuses
	}
	return result
}

func envNonEmpty(key string) bool {
	v, ok := os.LookupEnv(key)
	return ok && v != ""
}
