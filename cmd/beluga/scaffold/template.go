package scaffold

import (
	"io/fs"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// ScaffoldVars is the fixed substitution set for the strings.ReplaceAll-based
// renderer. Every __BELUGA_<FIELD>__ sentinel in a .tmpl file maps to exactly
// one field here. Substitution is deterministic: applyTemplate calls
// strings.ReplaceAll for each field in alphabetical order of the sentinel
// name so output is stable across runs (required for golden-file tests).
type ScaffoldVars struct {
	AgentName      string // "__BELUGA_AGENT_NAME__"       — derived from ProjectName (e.g. "myproject-agent")
	BelugaVersion  string // "__BELUGA_VERSION__"          — Options.BelugaVersion or "(devel)"
	ModelName      string // "__BELUGA_MODEL_NAME__"       — "gpt-4o-mini" for basic
	ModulePath     string // "__BELUGA_MODULE_PATH__"      — Options.ModulePath
	ProjectName    string // "__BELUGA_PROJECT_NAME__"     — Options.ProjectName
	ProviderImport string // "__BELUGA_PROVIDER_IMPORT__"  — full provider import path
	ProviderName   string // "__BELUGA_PROVIDER_NAME__"    — "openai" for basic
	ScaffoldedAt   string // "__BELUGA_SCAFFOLDED_AT__"    — RFC3339 UTC timestamp
}

// Registry is a scaffold-internal named-template registry. It is NOT a
// framework extensibility point (no consumer of beluga-ai/v2 can reach it),
// so it does not participate in the Layer 3 registry invariant.
//
// The zero value is NOT usable — construct via NewRegistry.
type Registry struct {
	mu        sync.RWMutex
	templates map[string]fs.FS
}

// NewRegistry constructs an empty Registry. Returned value is safe for
// concurrent use via Register / Get / Names.
func NewRegistry() *Registry {
	return &Registry{templates: make(map[string]fs.FS)}
}

// Register associates a template name with a read-only embedded filesystem.
// Returns *core.Error with ErrInvalidInput when name is empty, fsys is nil,
// or name is already registered — the registry is append-only by design.
func (r *Registry) Register(name string, fsys fs.FS) error {
	if name == "" {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: template name is empty; registration rejected")
	}
	if fsys == nil {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: template %q registered with nil filesystem", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.templates[name]; exists {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: template %q is already registered", name)
	}
	r.templates[name] = fsys
	return nil
}

// Get returns the filesystem for a template. ok == false when the name is
// not registered (callers must treat this as a user-facing error naming the
// available templates via Names()).
func (r *Registry) Get(name string) (fs.FS, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fsys, ok := r.templates[name]
	return fsys, ok
}

// Names returns the registered template names in sorted order. Returns an
// empty (non-nil) slice when the registry holds no entries — suitable for
// flag help-text construction at startup.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// defaultRegistry is the process-wide registry populated at package init
// by templates_builtin.go.
var defaultRegistry = NewRegistry()

// DefaultRegistry returns the process-wide registry. Exported so cobra
// commands can populate --template help text via DefaultRegistry().Names().
func DefaultRegistry() *Registry {
	return defaultRegistry
}
