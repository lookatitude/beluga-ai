package cartesia

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
	"go.opentelemetry.io/otel/codes"
)

// CartesiaBackend implements the VoiceBackend interface for Cartesia.
type CartesiaBackend struct {
	config          *CartesiaConfig
	sessionManager  *internal.SessionManager
	httpClient      *http.Client
	connectionState iface.ConnectionState
	healthStatus    *iface.HealthStatus
	metrics         *backend.Metrics
	mu              sync.RWMutex
}

// NewCartesiaBackend creates a new Cartesia backend.
func NewCartesiaBackend(config *CartesiaConfig) (*CartesiaBackend, error) {
	var metrics *backend.Metrics
	if config.EnableMetrics {
		// Initialize metrics if enabled
		metrics = nil // Will be initialized via InitMetrics if needed
	}

	return &CartesiaBackend{
		config:          config,
		sessionManager:  internal.NewSessionManager(config.Config),
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		connectionState: iface.ConnectionStateDisconnected,
		healthStatus: &iface.HealthStatus{
			Status:    "unknown",
			Details:   make(map[string]any),
			LastCheck: time.Now(),
		},
		metrics: metrics,
	}, nil
}

// Start starts the Cartesia backend, initializing the connection.
func (b *CartesiaBackend) Start(ctx context.Context) error {
	ctx, span := backend.StartSpan(ctx, "CartesiaBackend.Start", "cartesia")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connectionState == iface.ConnectionStateConnected {
		span.SetStatus(codes.Ok, "already connected")
		return nil // Already connected
	}

	b.connectionState = iface.ConnectionStateConnecting
	backend.AddSpanAttributes(span, map[string]any{
		"connection_state": string(b.connectionState),
		"api_url":          b.config.APIURL,
	})

	// Test Cartesia API connection
	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	err := internal.RetryWithBackoff(ctx, retryConfig, "CartesiaBackend.Start", func() error {
		// Test connection by making a simple API call
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", b.config.APIURL), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+b.config.APIKey)

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return nil
	})

	if err != nil {
		b.connectionState = iface.ConnectionStateError
		backend.RecordSpanError(span, err)
		backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to connect to Cartesia API", "error", err)
		return backend.NewBackendError("Start", backend.ErrCodeConnectionFailed, err)
	}

	b.connectionState = iface.ConnectionStateConnected
	span.SetStatus(codes.Ok, "backend started successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "Cartesia backend started successfully")
	return nil
}

// Stop stops the Cartesia backend gracefully.
func (b *CartesiaBackend) Stop(ctx context.Context) error {
	ctx, span := backend.StartSpan(ctx, "CartesiaBackend.Stop", "cartesia")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connectionState == iface.ConnectionStateDisconnected {
		span.SetStatus(codes.Ok, "already stopped")
		return nil
	}

	// Close all active sessions
	sessions := b.sessionManager.ListSessions()
	for _, session := range sessions {
		_ = b.sessionManager.CloseSession(ctx, session.GetID())
	}

	b.connectionState = iface.ConnectionStateDisconnected
	span.SetStatus(codes.Ok, "backend stopped successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "Cartesia backend stopped successfully")
	return nil
}

// CreateSession creates a new voice session by creating a Cartesia streaming session.
func (b *CartesiaBackend) CreateSession(ctx context.Context, config *iface.SessionConfig) (iface.VoiceSession, error) {
	startTime := time.Now()
	ctx, span := backend.StartSpan(ctx, "CartesiaBackend.CreateSession", "cartesia")
	defer span.End()

	// Track concurrent operations
	if b.metrics != nil {
		b.metrics.IncrementConcurrentOps(ctx, "cartesia")
		defer b.metrics.DecrementConcurrentOps(ctx, "cartesia")
	}

	backend.AddSpanAttributes(span, map[string]any{
		"user_id":       config.UserID,
		"transport":     config.Transport,
		"pipeline_type": string(config.PipelineType),
	})

	b.mu.RLock()
	connectionState := b.connectionState
	b.mu.RUnlock()

	if connectionState != iface.ConnectionStateConnected {
		err := backend.NewBackendError("CreateSession", backend.ErrCodeConnectionFailed,
			fmt.Errorf("backend not connected"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Create Cartesia streaming session via API
	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	err := internal.RetryWithBackoff(ctx, retryConfig, "CartesiaBackend.CreateSession", func() error {
		// Create Cartesia streaming session
		sessionURL := fmt.Sprintf("%s/stream", b.config.APIURL)
		req, err := http.NewRequestWithContext(ctx, "POST", sessionURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+b.config.APIKey)
		req.Header.Set("Content-Type", "application/json")

		// TODO: Add session configuration (model_id, voice_id, sample_rate, etc.)
		// For now, create a basic session

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create session: status %d", resp.StatusCode)
		}

		return nil
	})

	if err != nil {
		backend.RecordSpanError(span, err)
		backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to create Cartesia session", "error", err)
		return nil, backend.WrapError("CreateSession", err)
	}

	// Create Cartesia session
	session, err := NewCartesiaSession(b.config, config, b.httpClient)
	if err != nil {
		backend.RecordSpanError(span, err)
		return nil, backend.WrapError("CreateSession", err)
	}

	// Add to session manager
	if err := b.sessionManager.AddSession(session.GetID(), session); err != nil {
		backend.RecordSpanError(span, err)
		return nil, backend.WrapError("CreateSession", err)
	}

	// Record session creation time
	creationTime := time.Since(startTime)
	if b.metrics != nil {
		b.metrics.RecordSessionCreationTime(ctx, "cartesia", creationTime)
	}
	backend.AddSpanAttributes(span, map[string]any{
		"session_id":       session.GetID(),
		"creation_time_ms": creationTime.Milliseconds(),
	})

	if creationTime > 2*time.Second {
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "Session creation time exceeded target",
			"creation_time_ms", creationTime.Milliseconds(), "target_ms", 2000)
	}

	span.SetStatus(codes.Ok, "session created successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "Cartesia session created successfully",
		"session_id", session.GetID(), "creation_time_ms", creationTime.Milliseconds())
	return session, nil
}

// GetSession retrieves a voice session by ID.
func (b *CartesiaBackend) GetSession(ctx context.Context, sessionID string) (iface.VoiceSession, error) {
	return b.sessionManager.GetSession(sessionID)
}

// ListSessions returns all active voice sessions.
func (b *CartesiaBackend) ListSessions(ctx context.Context) ([]iface.VoiceSession, error) {
	return b.sessionManager.ListSessions(), nil
}

// CloseSession closes a voice session.
func (b *CartesiaBackend) CloseSession(ctx context.Context, sessionID string) error {
	return b.sessionManager.CloseSession(ctx, sessionID)
}

// HealthCheck checks the health status of the backend.
func (b *CartesiaBackend) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check Cartesia API health
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", b.config.APIURL), nil)
	if err != nil {
		return nil, backend.WrapError("HealthCheck", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.config.APIKey)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		b.healthStatus.Status = "unhealthy"
		b.healthStatus.Details["error"] = err.Error()
		return b.healthStatus, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		b.healthStatus.Status = "healthy"
	} else {
		b.healthStatus.Status = "degraded"
		b.healthStatus.Details["status_code"] = resp.StatusCode
	}

	b.healthStatus.LastCheck = time.Now()
	return b.healthStatus, nil
}

// GetConnectionState returns the current connection state.
func (b *CartesiaBackend) GetConnectionState() iface.ConnectionState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connectionState
}

// GetActiveSessionCount returns the number of active sessions.
func (b *CartesiaBackend) GetActiveSessionCount() int {
	return b.sessionManager.GetActiveSessionCount()
}

// GetConfig returns the backend configuration.
func (b *CartesiaBackend) GetConfig() *iface.Config {
	return b.config.Config
}

// UpdateConfig updates the backend configuration with validation.
func (b *CartesiaBackend) UpdateConfig(ctx context.Context, config *iface.Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Validate new config
	if err := backend.ValidateConfig(config); err != nil {
		return backend.NewBackendError("UpdateConfig", backend.ErrCodeInvalidConfig, err)
	}

	// Update config
	b.config = NewCartesiaConfig(config)

	// If connection state is connected, verify connection still works
	if b.connectionState == iface.ConnectionStateConnected {
		// Test connection
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", b.config.APIURL), nil)
		if err != nil {
			return backend.NewBackendError("UpdateConfig", backend.ErrCodeConnectionFailed, err)
		}
		req.Header.Set("Authorization", "Bearer "+b.config.APIKey)

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return backend.NewBackendError("UpdateConfig", backend.ErrCodeConnectionFailed, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.connectionState = iface.ConnectionStateError
			return backend.NewBackendError("UpdateConfig", backend.ErrCodeConnectionFailed,
				fmt.Errorf("connection test failed: status %d", resp.StatusCode))
		}
	}

	return nil
}
