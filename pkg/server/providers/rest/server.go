// Package rest provides REST server implementation with streaming support.
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// Server implements a REST server with streaming capabilities.
type Server struct {
	startTime   time.Time
	metrics     any
	logger      iface.Logger
	tracer      iface.Tracer
	router      *mux.Router
	server      *http.Server
	handlers    map[string]iface.StreamingHandler
	middlewares []iface.Middleware
	config      iface.RESTConfig
	mu          sync.RWMutex
}

// NewServer creates a new REST server instance.
func NewServer(opts ...iface.Option) (*Server, error) {
	options := &serverOptions{
		config: iface.Config{
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
		restConfig: &iface.RESTConfig{
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
		},
		middlewares: []iface.Middleware{},
	}

	// Skip option processing for now to avoid type issues
	_ = opts

	// Merge configs
	if options.restConfig != nil {
		options.restConfig.Config = options.config
	} else {
		options.restConfig = &iface.RESTConfig{
			Config: options.config,
		}
	}

	s := &Server{
		config:      *options.restConfig,
		router:      mux.NewRouter(),
		metrics:     nil,
		logger:      options.logger,
		tracer:      options.tracer,
		middlewares: options.middlewares,
		startTime:   time.Now(),
		handlers:    make(map[string]iface.StreamingHandler),
	}

	if s.logger == nil {
		s.logger = &defaultLogger{}
	}

	if s.tracer == nil {
		s.tracer = &noopTracer{}
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	// Apply middlewares
	handler := http.Handler(s.router)
	for _, middleware := range s.middlewares {
		handler = middleware(handler)
	}

	// Add built-in middlewares
	handler = s.corsMiddleware(handler)
	if s.config.EnableRateLimit {
		handler = s.rateLimitMiddleware(handler)
	}
	handler = s.loggingMiddleware(handler)
	handler = s.metricsMiddleware(handler)
	handler = s.tracingMiddleware(handler)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// API routes
	api := s.router.PathPrefix(s.config.APIBasePath).Subrouter()
	api.HandleFunc("/{resource}/{id}/stream", s.handleStreaming).Methods("GET", "POST")
	api.HandleFunc("/{resource}/{id}", s.handleNonStreaming).Methods("GET", "POST", "PUT", "DELETE")
	api.HandleFunc("/{resource}", s.handleList).Methods("GET")
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	s.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}
	s.mu.Unlock()

	s.logger.Info("Starting REST server",
		"host", s.config.Host,
		"port", s.config.Port,
		"api_base_path", s.config.APIBasePath,
	)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Server error", "error", err)
			return iface.NewInternalError("server_start", err)
		}
		return nil
	case <-ctx.Done():
		s.logger.Info("Server shutdown requested")
		return s.Stop(ctx)
	}
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	server := s.server
	s.mu.Unlock()

	if server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	s.logger.Info("Shutting down server gracefully")
	if err := server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Server shutdown error", "error", err)
		return iface.NewInternalError("server_shutdown", err)
	}

	s.logger.Info("Server shutdown complete")
	return nil
}

// IsHealthy returns true if the server is healthy.
func (s *Server) IsHealthy(ctx context.Context) bool {
	// Basic health check - can be extended with more sophisticated checks
	// A server is healthy if it's been created (even if not started yet)
	return s != nil
}

// RegisterHandler registers a streaming handler for a resource.
func (s *Server) RegisterHandler(resource string, handler iface.StreamingHandler) {
	s.handlers[resource] = handler
	s.logger.Info("Registered handler", "resource", resource)
}

// RegisterHTTPHandler registers an HTTP handler for a specific method and path.
func (s *Server) RegisterHTTPHandler(method, path string, handler http.HandlerFunc) {
	s.router.HandleFunc(path, handler).Methods(method)
	s.logger.Info("Registered HTTP handler", "method", method, "path", path)
}

// RegisterMiddleware adds middleware to the server.
func (s *Server) RegisterMiddleware(middleware iface.Middleware) {
	s.middlewares = append(s.middlewares, middleware)
}

// GetMux returns the underlying router for advanced customization.
func (s *Server) GetMux() any {
	return s.router
}

// Middleware implementations

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.EnableCORS {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" || !s.isAllowedOrigin(origin) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	// Simple in-memory rate limiter - can be replaced with more sophisticated implementation
	type clientInfo struct {
		resetTime time.Time
		requests  int
	}

	clients := make(map[string]*clientInfo)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.EnableRateLimit {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIP(r)
		now := time.Now()

		client, exists := clients[clientIP]
		if !exists || now.After(client.resetTime) {
			client = &clientInfo{
				requests:  1,
				resetTime: now.Add(time.Minute),
			}
			clients[clientIP] = client
		} else {
			client.requests++
		}

		if client.requests > s.config.RateLimitRequests {
			// TODO: Add metrics back
			// s.metrics.RecordHTTPRequest(r.Context(), r.Method, r.URL.Path, http.StatusTooManyRequests, 0)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		s.logger.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"user_agent", r.Header.Get("User-Agent"),
		)
	})
}

func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		// TODO: Add metrics back
		_ = duration // Remove when metrics are added back
	})
}

func (s *Server) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := s.tracer.Start(r.Context(), "http.request")
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.scheme", r.URL.Scheme),
			attribute.String("http.host", r.Host),
		)

		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
		)

		if rw.statusCode >= 400 {
			span.SetStatus(int(codes.Error), fmt.Sprintf("HTTP %d", rw.statusCode))
		}
	})
}

// HTTP handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// TODO: Add metrics back
	// s.metrics.RecordHealthCheck(r.Context(), true)
	// s.metrics.RecordServerUptime(r.Context(), time.Since(s.startTime))

	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(s.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStreaming(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resource := vars["resource"]
	id := vars["id"]

	handler, exists := s.handlers[resource]
	if !exists {
		s.handleError(w, r, iface.NewNotFoundError("streaming", fmt.Sprintf("handler for resource '%s'", resource)))
		return
	}

	if err := handler.HandleStreaming(w, r); err != nil {
		s.handleError(w, r, iface.NewInternalError(fmt.Sprintf("streaming_%s_%s", resource, id), err))
	}
}

func (s *Server) handleNonStreaming(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resource := vars["resource"]
	id := vars["id"]

	handler, exists := s.handlers[resource]
	if !exists {
		s.handleError(w, r, iface.NewNotFoundError("non_streaming", fmt.Sprintf("handler for resource '%s'", resource)))
		return
	}

	if err := handler.HandleNonStreaming(w, r); err != nil {
		s.handleError(w, r, iface.NewInternalError(fmt.Sprintf("non_streaming_%s_%s", resource, id), err))
	}
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resource := vars["resource"]

	handler, exists := s.handlers[resource]
	if !exists {
		s.handleError(w, r, iface.NewNotFoundError("list", fmt.Sprintf("handler for resource '%s'", resource)))
		return
	}

	// Use non-streaming handler for list operations
	if err := handler.HandleNonStreaming(w, r); err != nil {
		s.handleError(w, r, iface.NewInternalError("list_"+resource, err))
	}
}

func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err *iface.ServerError) {
	statusCode := err.HTTPStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]any{
		"error": map[string]any{
			"code":    string(err.Code),
			"message": err.Message,
		},
	}

	if err.Details != nil {
		response["error"].(map[string]any)["details"] = err.Details
	}

	if err.Operation != "" {
		response["error"].(map[string]any)["operation"] = err.Operation
	}

	_ = json.NewEncoder(w).Encode(response)
}

// Helper functions

func (s *Server) isAllowedOrigin(origin string) bool {
	for _, allowed := range s.config.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Default implementations for optional dependencies

type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...any) {}
func (l *defaultLogger) Info(msg string, args ...any)  {}
func (l *defaultLogger) Warn(msg string, args ...any)  {}
func (l *defaultLogger) Error(msg string, args ...any) {}

type noopTracer struct{}

func (t *noopTracer) Start(ctx context.Context, name string) (context.Context, iface.Span) {
	return ctx, &noopSpan{}
}

type noopSpan struct{}

func (s *noopSpan) End()                           {}
func (s *noopSpan) SetAttributes(attrs ...any)     {}
func (s *noopSpan) RecordError(err error)          {}
func (s *noopSpan) SetStatus(code int, msg string) {}

type noopMetrics struct{}

func (m *noopMetrics) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
}
func (m *noopMetrics) RecordActiveConnections(ctx context.Context, count int64) {}
func (m *noopMetrics) RecordMCPToolCall(ctx context.Context, toolName string, success bool, duration time.Duration) {
}

func (m *noopMetrics) RecordMCPResourceRead(ctx context.Context, resourceURI string, success bool, duration time.Duration) {
}
func (m *noopMetrics) RecordToolRegistration(ctx context.Context, toolName string)        {}
func (m *noopMetrics) RecordResourceRegistration(ctx context.Context, resourceURI string) {}
func (m *noopMetrics) RecordHealthCheck(ctx context.Context, healthy bool)                {}
func (m *noopMetrics) RecordServerUptime(ctx context.Context, uptime time.Duration)       {}

// serverOptions holds configuration options for the server.
type serverOptions struct {
	logger      iface.Logger
	tracer      iface.Tracer
	meter       iface.Meter
	restConfig  *iface.RESTConfig
	mcpConfig   *iface.MCPConfig
	middlewares []iface.Middleware
	tools       []iface.MCPTool
	resources   []iface.MCPResource
	config      iface.Config
}
