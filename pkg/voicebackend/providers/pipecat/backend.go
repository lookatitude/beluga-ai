package pipecat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
	"github.com/lookatitude/beluga-ai/pkg/voicebackend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voicebackend/internal"
	"go.opentelemetry.io/otel/codes"
)

// PipecatBackend implements the VoiceBackend interface for Pipecat (via Daily.co).
type PipecatBackend struct {
	config          *PipecatConfig
	sessionManager  *internal.SessionManager
	httpClient      *http.Client
	healthStatus    *iface.HealthStatus
	metrics         *voicebackend.Metrics
	connectionState iface.ConnectionState
	mu              sync.RWMutex
}

// NewPipecatBackend creates a new Pipecat voicebackend.
func NewPipecatBackend(config *PipecatConfig) (*PipecatBackend, error) {
	var metrics *voicebackend.Metrics
	if config.EnableMetrics {
		// Initialize metrics if enabled
		metrics = nil // Will be initialized via InitMetrics if needed
	}

	return &PipecatBackend{
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

// Start starts the Pipecat backend, initializing the Daily.co connection.
func (b *PipecatBackend) Start(ctx context.Context) error {
	ctx, span := voicebackend.StartSpan(ctx, "PipecatBackend.Start", "pipecat")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connectionState == iface.ConnectionStateConnected {
		span.SetStatus(codes.Ok, "already connected")
		return nil // Already connected
	}

	b.connectionState = iface.ConnectionStateConnecting
	voicebackend.AddSpanAttributes(span, map[string]any{
		"connection_state": string(b.connectionState),
		"daily_api_url":    b.config.DailyAPIURL,
	})

	// Test Daily.co API connection
	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	err := internal.RetryWithBackoff(ctx, retryConfig, "PipecatBackend.Start", func() error {
		// Test connection by making a simple API call
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.config.DailyAPIURL+"/rooms", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+b.config.DailyAPIKey)

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
		voicebackend.RecordSpanError(span, err)
		voicebackend.LogWithOTELContext(ctx, slog.LevelError, "Failed to connect to Daily.co API", "error", err)
		return voicebackend.NewBackendError("Start", voicebackend.ErrCodeConnectionFailed, err)
	}

	b.connectionState = iface.ConnectionStateConnected
	span.SetStatus(codes.Ok, "backend started successfully")
	voicebackend.LogWithOTELContext(ctx, slog.LevelInfo, "Pipecat backend started successfully")
	return nil
}

// Stop stops the Pipecat backend gracefully, completing in-flight conversations (T303, T304).
func (b *PipecatBackend) Stop(ctx context.Context) error {
	ctx, span := voicebackend.StartSpan(ctx, "PipecatBackend.Stop", "pipecat")
	defer span.End()

	b.mu.Lock()
	if b.connectionState == iface.ConnectionStateDisconnected {
		b.mu.Unlock()
		span.SetStatus(codes.Ok, "already stopped")
		return nil
	}

	sessions := b.sessionManager.ListSessions()
	shutdownTimeout := b.config.Timeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second // Default timeout
	}
	b.mu.Unlock()

	// Create shutdown context with configurable timeout (T304)
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	voicebackend.AddSpanAttributes(span, map[string]any{
		"active_sessions":  len(sessions),
		"shutdown_timeout": shutdownTimeout.String(),
	})

	// Close all active sessions gracefully
	var shutdownErrors []error
	for _, session := range sessions {
		// Give each session time to complete in-flight conversations
		if err := session.Stop(shutdownCtx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
			voicebackend.LogWithOTELContext(shutdownCtx, slog.LevelWarn, "Session shutdown error",
				"session_id", session.GetID(), "error", err)
		}
	}

	// Persist active sessions before shutdown (T305)
	if err := b.sessionManager.PersistActiveSessions(shutdownCtx); err != nil {
		voicebackend.LogWithOTELContext(shutdownCtx, slog.LevelWarn, "Failed to persist active sessions during shutdown",
			"error", err)
	}

	b.mu.Lock()
	b.connectionState = iface.ConnectionStateDisconnected
	b.mu.Unlock()

	if len(shutdownErrors) > 0 {
		voicebackend.LogWithOTELContext(ctx, slog.LevelWarn, "Some sessions failed to shutdown gracefully",
			"error_count", len(shutdownErrors))
	}

	span.SetStatus(codes.Ok, "backend stopped successfully")
	voicebackend.LogWithOTELContext(ctx, slog.LevelInfo, "Pipecat backend stopped successfully")
	return nil
}

// CreateSession creates a new voice session by creating a Daily.co room.
func (b *PipecatBackend) CreateSession(ctx context.Context, config *iface.SessionConfig) (iface.VoiceSession, error) {
	startTime := time.Now()
	ctx, span := voicebackend.StartSpan(ctx, "PipecatBackend.CreateSession", "pipecat")
	defer span.End()

	// Track concurrent operations
	if b.metrics != nil {
		b.metrics.IncrementConcurrentOps(ctx, "pipecat")
		defer b.metrics.DecrementConcurrentOps(ctx, "pipecat")
	}

	voicebackend.AddSpanAttributes(span, map[string]any{
		"user_id":       config.UserID,
		"transport":     config.Transport,
		"pipeline_type": string(config.PipelineType),
	})

	b.mu.RLock()
	connectionState := b.connectionState
	b.mu.RUnlock()

	if connectionState != iface.ConnectionStateConnected {
		err := voicebackend.NewBackendError("CreateSession", voicebackend.ErrCodeConnectionFailed,
			errors.New("backend not connected"))
		voicebackend.RecordSpanError(span, err)
		return nil, err
	}

	// Generate room name
	roomName := fmt.Sprintf("%s%s", b.config.RoomNamePrefix, config.UserID)

	// Create Daily.co room via API
	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	err := internal.RetryWithBackoff(ctx, retryConfig, "PipecatBackend.CreateSession", func() error {
		// Create Daily.co room
		roomURL := b.config.DailyAPIURL + "/rooms"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, roomURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+b.config.DailyAPIKey)
		req.Header.Set("Content-Type", "application/json")

		// TODO: Add room configuration (name, max_participants, etc.)
		// For now, create a basic room

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create room: status %d", resp.StatusCode)
		}

		return nil
	})
	if err != nil {
		voicebackend.RecordSpanError(span, err)
		voicebackend.LogWithOTELContext(ctx, slog.LevelError, "Failed to create Daily.co room", "error", err, "room_name", roomName)
		return nil, voicebackend.WrapError("CreateSession", err)
	}

	// Create Pipecat session
	session, err := NewPipecatSession(b.config, config, roomName, b.httpClient)
	if err != nil {
		voicebackend.RecordSpanError(span, err)
		return nil, voicebackend.WrapError("CreateSession", err)
	}

	// Add to session manager
	if err := b.sessionManager.AddSession(session.GetID(), session); err != nil {
		voicebackend.RecordSpanError(span, err)
		return nil, voicebackend.WrapError("CreateSession", err)
	}

	// Record session creation time
	creationTime := time.Since(startTime)
	if b.metrics != nil {
		b.metrics.RecordSessionCreationTime(ctx, "pipecat", creationTime)
	}
	voicebackend.AddSpanAttributes(span, map[string]any{
		"session_id":       session.GetID(),
		"room_name":        roomName,
		"creation_time_ms": creationTime.Milliseconds(),
	})

	if creationTime > 2*time.Second {
		voicebackend.LogWithOTELContext(ctx, slog.LevelWarn, "Session creation time exceeded target",
			"creation_time_ms", creationTime.Milliseconds(), "target_ms", 2000)
	}

	span.SetStatus(codes.Ok, "session created successfully")
	voicebackend.LogWithOTELContext(ctx, slog.LevelInfo, "Pipecat session created successfully",
		"session_id", session.GetID(), "room_name", roomName, "creation_time_ms", creationTime.Milliseconds())
	return session, nil
}

// GetSession retrieves a voice session by ID.
func (b *PipecatBackend) GetSession(ctx context.Context, sessionID string) (iface.VoiceSession, error) {
	return b.sessionManager.GetSession(sessionID)
}

// ListSessions returns all active voice sessions.
func (b *PipecatBackend) ListSessions(ctx context.Context) ([]iface.VoiceSession, error) {
	return b.sessionManager.ListSessions(), nil
}

// CloseSession closes a voice session.
func (b *PipecatBackend) CloseSession(ctx context.Context, sessionID string) error {
	return b.sessionManager.CloseSession(ctx, sessionID)
}

// HealthCheck checks the health status of the voicebackend.
func (b *PipecatBackend) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check Daily.co API health
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.config.DailyAPIURL+"/rooms", nil)
	if err != nil {
		return nil, voicebackend.WrapError("HealthCheck", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.config.DailyAPIKey)

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
func (b *PipecatBackend) GetConnectionState() iface.ConnectionState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connectionState
}

// GetActiveSessionCount returns the number of active sessions.
func (b *PipecatBackend) GetActiveSessionCount() int {
	return b.sessionManager.GetActiveSessionCount()
}

// GetConfig returns the backend configuration.
func (b *PipecatBackend) GetConfig() *iface.Config {
	return b.config.Config
}

// UpdateConfig updates the backend configuration with validation.
func (b *PipecatBackend) UpdateConfig(ctx context.Context, config *iface.Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Validate new config
	if err := voicebackend.ValidateConfig(config); err != nil {
		return voicebackend.NewBackendError("UpdateConfig", voicebackend.ErrCodeInvalidConfig, err)
	}

	// Update config
	b.config = NewPipecatConfig(config)

	// If connection state is connected, verify connection still works
	if b.connectionState == iface.ConnectionStateConnected {
		// Test connection
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.config.DailyAPIURL+"/rooms", nil)
		if err != nil {
			return voicebackend.NewBackendError("UpdateConfig", voicebackend.ErrCodeConnectionFailed, err)
		}
		req.Header.Set("Authorization", "Bearer "+b.config.DailyAPIKey)

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return voicebackend.NewBackendError("UpdateConfig", voicebackend.ErrCodeConnectionFailed, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.connectionState = iface.ConnectionStateError
			return voicebackend.NewBackendError("UpdateConfig", voicebackend.ErrCodeConnectionFailed,
				fmt.Errorf("connection test failed: status %d", resp.StatusCode))
		}
	}

	return nil
}
