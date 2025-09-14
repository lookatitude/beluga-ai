// Package server provides HTTP API endpoints for exposing the Beluga AI framework.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// Server represents the HTTP server for the Beluga AI framework.
type Server struct {
	router      *mux.Router
	port        int
	agents      map[string]AgentHandler
	chains      map[string]ChainHandler
	middlewares []Middleware
}

// AgentHandler handles HTTP requests for agent operations.
type AgentHandler interface {
	HandleExecute(w http.ResponseWriter, r *http.Request)
	HandleStatus(w http.ResponseWriter, r *http.Request)
}

// ChainHandler handles HTTP requests for chain operations.
type ChainHandler interface {
	HandleExecute(w http.ResponseWriter, r *http.Request)
	HandleStatus(w http.ResponseWriter, r *http.Request)
}

// Middleware represents HTTP middleware functions.
type Middleware func(http.Handler) http.Handler

// NewServer creates a new HTTP server instance.
func NewServer(port int) *Server {
	s := &Server{
		router:      mux.NewRouter(),
		port:        port,
		agents:      make(map[string]AgentHandler),
		chains:      make(map[string]ChainHandler),
		middlewares: []Middleware{},
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures the HTTP routes for the server.
func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Agent routes
	api.HandleFunc("/agents/{name}/execute", s.handleAgentExecute).Methods("POST")
	api.HandleFunc("/agents/{name}/status", s.handleAgentStatus).Methods("GET")
	api.HandleFunc("/agents", s.handleListAgents).Methods("GET")

	// Chain routes
	api.HandleFunc("/chains/{name}/execute", s.handleChainExecute).Methods("POST")
	api.HandleFunc("/chains/{name}/status", s.handleChainStatus).Methods("GET")
	api.HandleFunc("/chains", s.handleListChains).Methods("GET")

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

// RegisterAgent registers an agent handler with the server.
func (s *Server) RegisterAgent(name string, handler AgentHandler) {
	s.agents[name] = handler
}

// RegisterChain registers a chain handler with the server.
func (s *Server) RegisterChain(name string, handler ChainHandler) {
	s.chains[name] = handler
}

// AddMiddleware adds middleware to the server.
func (s *Server) AddMiddleware(middleware Middleware) {
	s.middlewares = append(s.middlewares, middleware)
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	// Apply middlewares
	handler := s.router
	for _, middleware := range s.middlewares {
		handler = middleware(handler)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// Handler methods (placeholders)
func (s *Server) handleAgentExecute(w http.ResponseWriter, r *http.Request) {
	// Implementation would extract agent name from URL and delegate to handler
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleChainExecute(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleChainStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleListChains(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy"}`))
}
