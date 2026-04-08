package learning

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/tool"
)

// VersionRecord is an immutable record of a tool version in the registry.
type VersionRecord struct {
	// Version is the sequential version number.
	Version int
	// Tool is the tool at this version.
	Tool tool.Tool
	// CreatedAt is the timestamp when this version was created.
	CreatedAt time.Time
	// Active indicates whether this version is the currently active one.
	Active bool
}

// toolEntry holds all versions and the current pointer for a single tool name.
type toolEntry struct {
	versions []VersionRecord
	current  int // index into versions for the active version
}

// VersionedRegistry wraps a tool.Registry with version tracking. Each tool can
// have multiple immutable versions, with a "current" pointer that determines
// which version is served. Supports Upsert, Activate, Rollback, and History.
type VersionedRegistry struct {
	mu      sync.RWMutex
	inner   *tool.Registry
	entries map[string]*toolEntry
	hooks   Hooks
}

// VersionedRegistryOption configures a VersionedRegistry.
type VersionedRegistryOption func(*VersionedRegistry)

// WithRegistryHooks sets lifecycle hooks on the versioned registry.
func WithRegistryHooks(h Hooks) VersionedRegistryOption {
	return func(vr *VersionedRegistry) {
		vr.hooks = h
	}
}

// NewVersionedRegistry creates a new VersionedRegistry backed by the given
// tool.Registry. If registry is nil, a new one is created.
func NewVersionedRegistry(registry *tool.Registry, opts ...VersionedRegistryOption) *VersionedRegistry {
	if registry == nil {
		registry = tool.NewRegistry()
	}
	vr := &VersionedRegistry{
		inner:   registry,
		entries: make(map[string]*toolEntry),
	}
	for _, opt := range opts {
		opt(vr)
	}
	return vr
}

// Upsert adds a new version of a tool. If the tool already exists, a new version
// is appended and activated. If it is new, the first version is created and activated.
// Returns the new version number.
func (vr *VersionedRegistry) Upsert(t tool.Tool) (int, error) {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	name := t.Name()

	entry, exists := vr.entries[name]
	if !exists {
		entry = &toolEntry{
			versions: nil,
			current:  0,
		}
		vr.entries[name] = entry
	}

	// Deactivate current version.
	if len(entry.versions) > 0 {
		entry.versions[entry.current].Active = false
	}

	version := len(entry.versions) + 1
	record := VersionRecord{
		Version:   version,
		Tool:      t,
		CreatedAt: time.Now(),
		Active:    true,
	}
	entry.versions = append(entry.versions, record)
	entry.current = len(entry.versions) - 1

	// Update inner registry. Remove first if exists, then add.
	if exists {
		_ = vr.inner.Remove(name)
	}
	if err := vr.inner.Add(t); err != nil {
		return 0, fmt.Errorf("versioned registry: failed to add tool %q: %w", name, err)
	}

	// Fire hook.
	if vr.hooks.OnVersionActivated != nil {
		vr.hooks.OnVersionActivated(name, version)
	}

	return version, nil
}

// Activate sets the given version as the current active version for the named tool.
// Returns an error if the tool or version does not exist.
func (vr *VersionedRegistry) Activate(name string, version int) error {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	entry, exists := vr.entries[name]
	if !exists {
		return fmt.Errorf("versioned registry: tool %q not found", name)
	}

	idx := version - 1
	if idx < 0 || idx >= len(entry.versions) {
		return fmt.Errorf("versioned registry: version %d not found for tool %q", version, name)
	}

	// Deactivate current, activate target.
	entry.versions[entry.current].Active = false
	entry.versions[idx].Active = true
	entry.current = idx

	// Update inner registry.
	_ = vr.inner.Remove(name)
	if err := vr.inner.Add(entry.versions[idx].Tool); err != nil {
		return fmt.Errorf("versioned registry: failed to activate tool %q v%d: %w", name, version, err)
	}

	if vr.hooks.OnVersionActivated != nil {
		vr.hooks.OnVersionActivated(name, version)
	}

	return nil
}

// Rollback activates the previous version of the named tool. Returns an error
// if there is no previous version or the tool does not exist.
func (vr *VersionedRegistry) Rollback(name string) (int, error) {
	vr.mu.Lock()

	entry, exists := vr.entries[name]
	if !exists {
		vr.mu.Unlock()
		return 0, fmt.Errorf("versioned registry: tool %q not found", name)
	}

	if entry.current == 0 {
		vr.mu.Unlock()
		return 0, fmt.Errorf("versioned registry: no previous version for tool %q", name)
	}

	prevVersion := entry.versions[entry.current-1].Version
	vr.mu.Unlock()

	// Use Activate which acquires the lock.
	if err := vr.Activate(name, prevVersion); err != nil {
		return 0, err
	}

	return prevVersion, nil
}

// History returns all version records for the named tool, ordered by version.
// Returns an error if the tool does not exist.
func (vr *VersionedRegistry) History(name string) ([]VersionRecord, error) {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	entry, exists := vr.entries[name]
	if !exists {
		return nil, fmt.Errorf("versioned registry: tool %q not found", name)
	}

	// Return a copy to prevent mutation.
	records := make([]VersionRecord, len(entry.versions))
	copy(records, entry.versions)
	return records, nil
}

// Get returns the currently active version of the named tool.
func (vr *VersionedRegistry) Get(name string) (tool.Tool, error) {
	return vr.inner.Get(name)
}

// List returns a sorted list of all tool names in the registry.
func (vr *VersionedRegistry) List() []string {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	names := make([]string, 0, len(vr.entries))
	for name := range vr.entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// All returns all currently active tools from the inner registry.
func (vr *VersionedRegistry) All() []tool.Tool {
	return vr.inner.All()
}
