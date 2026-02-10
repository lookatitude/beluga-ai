package server

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
)

// ServerAdapter is the interface that HTTP framework adapters must implement.
// Each adapter wraps a specific HTTP framework and exposes a uniform API for
// registering agents and handlers.
type ServerAdapter interface {
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
		return nil, fmt.Errorf("server: unknown adapter %q (registered: %v)", name, List())
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
	mux    *http.ServeMux
	server *http.Server
	cfg    Config
	mu     sync.RWMutex
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
		return fmt.Errorf("server/register-agent: agent must not be nil")
	}
	handler := NewAgentHandler(a)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mux.Handle(path+"/", handler)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (s *StdlibAdapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/register-handler: handler must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mux.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (s *StdlibAdapter) Serve(ctx context.Context, addr string) error {
	s.mu.Lock()
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server/serve: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("server/serve: %w", err)
	}
}

// Shutdown gracefully shuts down the server.
func (s *StdlibAdapter) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	srv := s.server
	s.mu.RUnlock()
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server/shutdown: %w", err)
	}
	return nil
}

func init() {
	Register("stdlib", func(cfg Config) (ServerAdapter, error) {
		return NewStdlibAdapter(cfg), nil
	})
}
