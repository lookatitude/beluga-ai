// Package providers provides concrete implementations of REST and MCP servers.
package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
	"github.com/lookatitude/beluga-ai/pkg/server/providers/rest"
)

// RESTProvider provides a ready-to-use REST server implementation.
type RESTProvider struct {
	server iface.RESTServer
}

// NewRESTProvider creates a new REST provider with default configuration.
func NewRESTProvider(opts ...iface.Option) (*RESTProvider, error) {
	// Set default REST configuration if not provided
	hasRESTConfig := false
	for _, opt := range opts {
		// Check if REST config is already provided
		_ = opt
		hasRESTConfig = true // Simplified check
	}

	if !hasRESTConfig {
		defaultOpts := []iface.Option{
			iface.WithRESTConfig(iface.RESTConfig{
				Config: iface.Config{
					Host:            "localhost",
					Port:            8080,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					IdleTimeout:     120 * time.Second,
					MaxHeaderBytes:  1 << 20, // 1MB
					EnableCORS:      true,
					CORSOrigins:     []string{"*"},
					EnableMetrics:   true,
					EnableTracing:   true,
					LogLevel:        "info",
					ShutdownTimeout: 30 * time.Second,
				},
				APIBasePath:       "/api/v1",
				EnableStreaming:   true,
				MaxRequestSize:    10 << 20, // 10MB
				RateLimitRequests: 1000,
				EnableRateLimit:   true,
			}),
		}
		opts = append(defaultOpts, opts...)
	}

	srv, err := rest.NewServer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST server: %w", err)
	}

	return &RESTProvider{
		server: srv,
	}, nil
}

// Start starts the REST server.
func (p *RESTProvider) Start(ctx context.Context) error {
	return p.server.Start(ctx)
}

// Stop stops the REST server.
func (p *RESTProvider) Stop(ctx context.Context) error {
	return p.server.Stop(ctx)
}

// RegisterAgentHandler registers an agent handler for REST endpoints.
func (p *RESTProvider) RegisterAgentHandler(name string, handler AgentRESTHandler) {
	adapter := &agentRESTAdapter{name: name, handler: handler}
	p.server.RegisterHTTPHandler("POST", fmt.Sprintf("/api/v1/agents/%s/execute", name), adapter.handleExecuteHTTP)
	p.server.RegisterHTTPHandler("GET", fmt.Sprintf("/api/v1/agents/%s/status", name), adapter.handleStatusHTTP)
}

// RegisterChainHandler registers a chain handler for REST endpoints.
func (p *RESTProvider) RegisterChainHandler(name string, handler ChainRESTHandler) {
	adapter := &chainRESTAdapter{name: name, handler: handler}
	p.server.RegisterHTTPHandler("POST", fmt.Sprintf("/api/v1/chains/%s/execute", name), adapter.handleExecuteHTTP)
	p.server.RegisterHTTPHandler("GET", fmt.Sprintf("/api/v1/chains/%s/status", name), adapter.handleStatusHTTP)
}

func (p *RESTProvider) RegisterWorkflowHandler(name string, handler WorkflowRESTHandler) {
	adapter := &workflowRESTAdapter{name: name, handler: handler}
	p.server.RegisterHTTPHandler("POST", fmt.Sprintf("/api/v1/workflows/%s/execute", name), adapter.handleExecuteHTTP)
	p.server.RegisterHTTPHandler("GET", fmt.Sprintf("/api/v1/workflows/%s/status", name), adapter.handleStatusHTTP)
}

// GetServer returns the underlying REST server for advanced usage.
func (p *RESTProvider) GetServer() iface.RESTServer {
	return p.server
}

// AgentRESTHandler handles REST requests for agents.
type AgentRESTHandler interface {
	Execute(ctx context.Context, request any) (any, error)
	GetStatus(ctx context.Context, id string) (any, error)
}

// ChainRESTHandler handles REST requests for chains.
type ChainRESTHandler interface {
	Execute(ctx context.Context, request any) (any, error)
	GetStatus(ctx context.Context, id string) (any, error)
}

type WorkflowRESTHandler interface {
	Execute(ctx context.Context, request any) (any, error)
	GetStatus(ctx context.Context, id string) (any, error)
}

// agentRESTAdapter adapts AgentRESTHandler to StreamingHandler interface.
type agentRESTAdapter struct {
	handler AgentRESTHandler
	name    string
}

func (a *agentRESTAdapter) HandleStreaming(w http.ResponseWriter, r *http.Request) error {
	// Parse request
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return iface.NewInvalidRequestError("agent_streaming", "invalid JSON request", err)
	}

	// Execute agent
	ctx := r.Context()
	result, err := a.handler.Execute(ctx, req)
	if err != nil {
		return iface.NewInternalError("agent_execution", err)
	}

	// Stream response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(map[string]any{
		"agent":     a.name,
		"result":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode streaming response: %w", err)
	}
	return nil
}

func (a *agentRESTAdapter) HandleNonStreaming(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return a.handleExecute(w, r)
	case http.MethodGet:
		return a.handleStatus(w, r)
	default:
		return iface.NewInvalidRequestError("agent_method", "unsupported method: "+r.Method, nil)
	}
}

func (a *agentRESTAdapter) handleExecute(w http.ResponseWriter, r *http.Request) error {
	// Parse request
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return iface.NewInvalidRequestError("agent_execute", "invalid JSON request", err)
	}

	// Execute agent
	ctx := r.Context()
	result, err := a.handler.Execute(ctx, req)
	if err != nil {
		return iface.NewInternalError("agent_execution", err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"agent":     a.name,
		"result":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}

func (a *agentRESTAdapter) handleStatus(w http.ResponseWriter, r *http.Request) error {
	// Extract ID from URL (assuming it's in the path)
	id := r.URL.Query().Get("id")
	if id == "" {
		return iface.NewInvalidRequestError("agent_status", "missing id parameter", nil)
	}

	// Get status
	ctx := r.Context()
	result, err := a.handler.GetStatus(ctx, id)
	if err != nil {
		return iface.NewInternalError("agent_status", err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"agent":     a.name,
		"id":        id,
		"status":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode status response: %w", err)
	}
	return nil
}

func (a *agentRESTAdapter) handleExecuteHTTP(w http.ResponseWriter, r *http.Request) {
	if err := a.handleExecute(w, r); err != nil {
		serverError := func() *iface.ServerError {
			target := &iface.ServerError{}
			_ = errors.As(err, &target)
			return target
		}()
		w.WriteHeader(serverError.HTTPStatus())
		_ = json.NewEncoder(w).Encode(serverError) //nolint:errcheck // Error already handled, best effort to send error response
	}
}

func (a *agentRESTAdapter) handleStatusHTTP(w http.ResponseWriter, r *http.Request) {
	if err := a.handleStatus(w, r); err != nil {
		serverError := func() *iface.ServerError {
			target := &iface.ServerError{}
			_ = errors.As(err, &target)
			return target
		}()
		w.WriteHeader(serverError.HTTPStatus())
		_ = json.NewEncoder(w).Encode(serverError) //nolint:errcheck // Error already handled, best effort to send error response
	}
}

// chainRESTAdapter adapts ChainRESTHandler to StreamingHandler interface.
type chainRESTAdapter struct {
	handler ChainRESTHandler
	name    string
}

func (c *chainRESTAdapter) HandleStreaming(w http.ResponseWriter, r *http.Request) error {
	// Parse request
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return iface.NewInvalidRequestError("chain_streaming", "invalid JSON request", err)
	}

	// Execute chain
	ctx := r.Context()
	result, err := c.handler.Execute(ctx, req)
	if err != nil {
		return iface.NewInternalError("chain_execution", err)
	}

	// Stream response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(map[string]any{
		"chain":     c.name,
		"result":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode streaming response: %w", err)
	}
	return nil
}

func (c *chainRESTAdapter) HandleNonStreaming(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return c.handleExecute(w, r)
	case http.MethodGet:
		return c.handleStatus(w, r)
	default:
		return iface.NewInvalidRequestError("chain_method", "unsupported method: "+r.Method, nil)
	}
}

func (c *chainRESTAdapter) handleExecute(w http.ResponseWriter, r *http.Request) error {
	// Parse request
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return iface.NewInvalidRequestError("chain_execute", "invalid JSON request", err)
	}

	// Execute chain
	ctx := r.Context()
	result, err := c.handler.Execute(ctx, req)
	if err != nil {
		return iface.NewInternalError("chain_execution", err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"chain":     c.name,
		"result":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}

func (c *chainRESTAdapter) handleStatus(w http.ResponseWriter, r *http.Request) error {
	// Extract ID from URL (assuming it's in the path)
	id := r.URL.Query().Get("id")
	if id == "" {
		return iface.NewInvalidRequestError("chain_status", "missing id parameter", nil)
	}

	// Get status
	ctx := r.Context()
	result, err := c.handler.GetStatus(ctx, id)
	if err != nil {
		return iface.NewInternalError("chain_status", err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"chain":     c.name,
		"id":        id,
		"status":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode status response: %w", err)
	}
	return nil
}

func (c *chainRESTAdapter) handleExecuteHTTP(w http.ResponseWriter, r *http.Request) {
	if err := c.handleExecute(w, r); err != nil {
		serverError := func() *iface.ServerError {
			target := &iface.ServerError{}
			_ = errors.As(err, &target)
			return target
		}()
		w.WriteHeader(serverError.HTTPStatus())
		_ = json.NewEncoder(w).Encode(serverError) //nolint:errcheck // Error already handled, best effort to send error response
	}
}

func (c *chainRESTAdapter) handleStatusHTTP(w http.ResponseWriter, r *http.Request) {
	if err := c.handleStatus(w, r); err != nil {
		serverError := func() *iface.ServerError {
			target := &iface.ServerError{}
			_ = errors.As(err, &target)
			return target
		}()
		w.WriteHeader(serverError.HTTPStatus())
		_ = json.NewEncoder(w).Encode(serverError) //nolint:errcheck // Error already handled, best effort to send error response
	}
}

type workflowRESTAdapter struct {
	handler WorkflowRESTHandler
	name    string
}

func (wf *workflowRESTAdapter) HandleStreaming(w http.ResponseWriter, r *http.Request) error {
	// Parse request
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return iface.NewInvalidRequestError("workflow_streaming", "invalid JSON request", err)
	}

	// Execute workflow
	ctx := r.Context()
	result, err := wf.handler.Execute(ctx, req)
	if err != nil {
		return iface.NewInternalError("workflow_execution", err)
	}

	// Stream response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(map[string]any{
		"workflow":  wf.name,
		"result":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode streaming response: %w", err)
	}
	return nil
}

func (wf *workflowRESTAdapter) HandleNonStreaming(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return wf.handleExecute(w, r)
	case http.MethodGet:
		return wf.handleStatus(w, r)
	default:
		return iface.NewInvalidRequestError("workflow_method", "unsupported method: "+r.Method, nil)
	}
}

func (wf *workflowRESTAdapter) handleExecute(w http.ResponseWriter, r *http.Request) error {
	// Parse request
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return iface.NewInvalidRequestError("workflow_execute", "invalid JSON request", err)
	}

	// Execute workflow
	ctx := r.Context()
	result, err := wf.handler.Execute(ctx, req)
	if err != nil {
		return iface.NewInternalError("workflow_execution", err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"workflow":  wf.name,
		"result":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}

func (wf *workflowRESTAdapter) handleStatus(w http.ResponseWriter, r *http.Request) error {
	// Extract ID from URL
	id := r.URL.Query().Get("id")
	if id == "" {
		return iface.NewInvalidRequestError("workflow_status", "missing id parameter", nil)
	}

	// Get status
	ctx := r.Context()
	result, err := wf.handler.GetStatus(ctx, id)
	if err != nil {
		return iface.NewInternalError("workflow_status", err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"workflow":  wf.name,
		"id":        id,
		"status":    result,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to encode status response: %w", err)
	}
	return nil
}

func (wf *workflowRESTAdapter) handleExecuteHTTP(w http.ResponseWriter, r *http.Request) {
	if err := wf.handleExecute(w, r); err != nil {
		serverError := func() *iface.ServerError {
			target := &iface.ServerError{}
			_ = errors.As(err, &target)
			return target
		}()
		w.WriteHeader(serverError.HTTPStatus())
		_ = json.NewEncoder(w).Encode(serverError) //nolint:errcheck // Error already handled, best effort to send error response
	}
}

func (wf *workflowRESTAdapter) handleStatusHTTP(w http.ResponseWriter, r *http.Request) {
	if err := wf.handleStatus(w, r); err != nil {
		serverError := func() *iface.ServerError {
			target := &iface.ServerError{}
			_ = errors.As(err, &target)
			return target
		}()
		w.WriteHeader(serverError.HTTPStatus())
		_ = json.NewEncoder(w).Encode(serverError) //nolint:errcheck // Error already handled, best effort to send error response
	}
}
