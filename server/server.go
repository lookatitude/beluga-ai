package server

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/httputil"
)

// ServerAdapter is the interface that HTTP framework adapters must implement.
// Each adapter wraps a specific HTTP framework and exposes a uniform API for
// registering agents and handlers.
type ServerAdapter interface { //nolint:revive // stuttering name is part of the published v2 API; renaming would break callers
	// RegisterAgent registers an agent at the given path prefix. The adapter
	// creates sub-routes for invoke and stream endpoints automatically.
	RegisterAgent(path string, a agent.Agent) error

	// RegisterHandler registers a raw http.Handler at the given path.
	RegisterHandler(path string, handler http.Handler) error

	// Serve starts the HTTP server on the given address. It blocks until the
	// server exits or the context is canceled.
	Serve(ctx context.Context, addr string) error

	// Shutdown gracefully shuts down the server, allowing in-flight requests
	// to complete within the context deadline.
	Shutdown(ctx context.Context) error
}

// Config holds configuration for creating a ServerAdapter.
type Config struct {
	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum duration to wait for the next request.
	IdleTimeout time.Duration

	// Extra holds adapter-specific configuration values.
	Extra map[string]any
}

// Factory creates a ServerAdapter from a Config.
type Factory func(cfg Config) (ServerAdapter, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds an adapter factory to the global registry. It is intended to
// be called from provider init() functions. Duplicate registrations for the
// same name silently overwrite the previous factory.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a ServerAdapter by looking up the adapter name in the registry
// and calling its factory with the given configuration.
func New(name string, cfg Config) (ServerAdapter, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "server: unknown adapter %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered adapters, sorted alphabetically.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// StdlibAdapter is the built-in ServerAdapter that uses the standard library
// net/http package.
type StdlibAdapter struct {
	mux *http.ServeMux
	lc  httputil.ServerLifecycle
	cfg Config
}

// NewStdlibAdapter creates a new StdlibAdapter with the given configuration.
func NewStdlibAdapter(cfg Config) *StdlibAdapter {
	return &StdlibAdapter{
		mux: http.NewServeMux(),
		cfg: cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// two sub-routes: {path}/invoke for synchronous invocation and {path}/stream
// for SSE streaming.
func (s *StdlibAdapter) RegisterAgent(path string, a agent.Agent) error {
	if a == nil {
		return core.Errorf(core.ErrInvalidInput, "server/register-agent: agent must not be nil")
	}
	handler := NewAgentHandler(a)
	prefix := strings.TrimRight(path, "/")
	if prefix == "" {
		s.mux.Handle("/", handler)
		return nil
	}
	// The agent handler routes on "/invoke" and "/stream", so the mount
	// prefix must be stripped before the request reaches it.
	s.mux.Handle(prefix+"/", http.StripPrefix(prefix, handler))
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (s *StdlibAdapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return core.Errorf(core.ErrInvalidInput, "server/register-handler: handler must not be nil")
	}
	s.mux.Handle(path, handler)
	return nil
}

// Default timeouts applied by StdlibAdapter when the corresponding Config
// field is zero. WriteTimeout has no default because agent streams (SSE) are
// long-lived; set Config.WriteTimeout explicitly if only short responses are
// served.
const (
	DefaultReadTimeout = 30 * time.Second
	DefaultIdleTimeout = 120 * time.Second
)

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled. Zero ReadTimeout and IdleTimeout
// values are replaced by DefaultReadTimeout and DefaultIdleTimeout.
func (s *StdlibAdapter) Serve(ctx context.Context, addr string) error {
	read, idle := s.effectiveTimeouts()
	return s.lc.Serve(ctx, addr, s.mux,
		read, s.cfg.WriteTimeout, idle,
		"server/serve")
}

// effectiveTimeouts returns the read and idle timeouts after defaults.
func (s *StdlibAdapter) effectiveTimeouts() (read, idle time.Duration) {
	read, idle = s.cfg.ReadTimeout, s.cfg.IdleTimeout
	if read <= 0 {
		read = DefaultReadTimeout
	}
	if idle <= 0 {
		idle = DefaultIdleTimeout
	}
	return read, idle
}

// Shutdown gracefully shuts down the server.
func (s *StdlibAdapter) Shutdown(ctx context.Context) error {
	return s.lc.Shutdown(ctx, "server/shutdown")
}

func init() {
	Register("stdlib", func(cfg Config) (ServerAdapter, error) {
		return NewStdlibAdapter(cfg), nil
	})
}
