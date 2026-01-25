package livekit

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/livekit/protocol/livekit"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
	lksdkwrapper "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit/internal"
	"go.opentelemetry.io/otel/codes"
)

// LiveKitBackend implements the VoiceBackend interface for LiveKit.
type LiveKitBackend struct {
	config          *LiveKitConfig
	sessionManager  *internal.SessionManager
	roomService     *lksdkwrapper.RoomServiceClient
	healthStatus    *iface.HealthStatus
	metrics         *backend.Metrics
	connectionState iface.ConnectionState
	mu              sync.RWMutex
}

// NewLiveKitBackend creates a new LiveKit backend.
// Connection pooling: The roomService client can be reused across sessions (T175).
func NewLiveKitBackend(config *LiveKitConfig) (*LiveKitBackend, error) {
	var metrics *backend.Metrics
	if config.EnableMetrics {
		// Initialize metrics if enabled
		// Note: In a full implementation, this would use a global metrics instance
		// For now, metrics are optional and can be nil
		metrics = nil // Will be initialized via InitMetrics if needed
	}

	return &LiveKitBackend{
		config:          config,
		sessionManager:  internal.NewSessionManager(config.Config),
		connectionState: iface.ConnectionStateDisconnected,
		healthStatus: &iface.HealthStatus{
			Status:    "unknown",
			Details:   make(map[string]any),
			LastCheck: time.Now(),
		},
		metrics: metrics,
		// Connection pooling: roomService is shared across all sessions (T175)
	}, nil
}

// Start starts the LiveKit backend, initializing the connection.
func (b *LiveKitBackend) Start(ctx context.Context) error {
	ctx, span := backend.StartSpan(ctx, "LiveKitBackend.Start", "livekit")
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
	})

	// Initialize LiveKit room service client with retry logic
	var roomService *lksdkwrapper.RoomServiceClient
	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	err := internal.RetryWithBackoff(ctx, retryConfig, "LiveKitBackend.Start", func() error {
		var err error
		roomService, err = lksdkwrapper.NewRoomServiceClient(b.config.URL, b.config.APIKey, b.config.APISecret)
		if err != nil {
			return err
		}

		// Test connection by listing rooms
		_, err = roomService.ListRooms(ctx, &livekit.ListRoomsRequest{})
		return err
	})
	if err != nil {
		b.connectionState = iface.ConnectionStateError
		backend.RecordSpanError(span, err)
		backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to create LiveKit room service client", "error", err)
		return backend.NewBackendError("Start", backend.ErrCodeConnectionFailed, err)
	}

	b.roomService = roomService
	b.connectionState = iface.ConnectionStateConnected

	span.SetStatus(codes.Ok, "backend started successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "LiveKit backend started successfully")
	return nil
}

// Stop stops the LiveKit backend gracefully, completing in-flight conversations (T302, T304).
func (b *LiveKitBackend) Stop(ctx context.Context) error {
	ctx, span := backend.StartSpan(ctx, "LiveKitBackend.Stop", "livekit")
	defer span.End()

	b.mu.Lock()
	sessions := b.sessionManager.ListSessions()
	shutdownTimeout := b.config.Timeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second // Default timeout
	}
	b.mu.Unlock()

	// Create shutdown context with configurable timeout (T304)
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	backend.AddSpanAttributes(span, map[string]any{
		"active_sessions":  len(sessions),
		"shutdown_timeout": shutdownTimeout.String(),
	})

	// Close all active sessions gracefully
	var shutdownErrors []error
	for _, session := range sessions {
		// Give each session time to complete in-flight conversations
		if err := session.Stop(shutdownCtx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
			backend.LogWithOTELContext(shutdownCtx, slog.LevelWarn, "Session shutdown error",
				"session_id", session.GetID(), "error", err)
		}
	}

	// Persist active sessions before shutdown (T305)
	if err := b.sessionManager.PersistActiveSessions(shutdownCtx); err != nil {
		backend.LogWithOTELContext(shutdownCtx, slog.LevelWarn, "Failed to persist active sessions during shutdown",
			"error", err)
	}

	b.mu.Lock()
	b.connectionState = iface.ConnectionStateDisconnected
	b.mu.Unlock()

	if len(shutdownErrors) > 0 {
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "Some sessions failed to shutdown gracefully",
			"error_count", len(shutdownErrors))
		// Don't fail shutdown if some sessions had errors
	}

	span.SetStatus(codes.Ok, "backend stopped successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "LiveKit backend stopped successfully")
	return nil
}

// CreateSession creates a new voice session by creating a LiveKit room and participant.
// This method is thread-safe and supports concurrent session creation (T172).
func (b *LiveKitBackend) CreateSession(ctx context.Context, config *iface.SessionConfig) (iface.VoiceSession, error) {
	startTime := time.Now()
	ctx, span := backend.StartSpan(ctx, "LiveKitBackend.CreateSession", "livekit")
	defer span.End()

	// Track concurrent operations
	if b.metrics != nil {
		b.metrics.IncrementConcurrentOps(ctx, "livekit")
		defer b.metrics.DecrementConcurrentOps(ctx, "livekit")
	}

	backend.AddSpanAttributes(span, map[string]any{
		"user_id":       config.UserID,
		"transport":     config.Transport,
		"pipeline_type": string(config.PipelineType),
	})

	b.mu.RLock()
	roomService := b.roomService
	connectionState := b.connectionState
	b.mu.RUnlock()

	if connectionState != iface.ConnectionStateConnected {
		err := backend.NewBackendError("CreateSession", backend.ErrCodeConnectionFailed,
			errors.New("backend not connected"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	if roomService == nil {
		err := backend.NewBackendError("CreateSession", backend.ErrCodeConnectionFailed,
			errors.New("room service not initialized"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Authenticate user if AuthHook is provided (T325, FR-021, FR-022)
	if b.config.AuthHook != nil {
		// Extract token from metadata or config
		token := ""
		if config.Metadata != nil {
			if t, ok := config.Metadata["token"].(string); ok {
				token = t
			}
		}

		authResult, err := b.config.AuthHook.Authenticate(ctx, token, config.Metadata)
		if err != nil {
			backend.RecordSpanError(span, err)
			return nil, backend.NewBackendError("CreateSession", backend.ErrCodeAuthenticationFailed, err)
		}

		if !authResult.Authorized {
			err := backend.NewBackendError("CreateSession", backend.ErrCodeAuthenticationFailed,
				errors.New("user not authorized"))
			backend.RecordSpanError(span, err)
			return nil, err
		}

		// Update user ID from auth result if provided
		if authResult.UserID != "" {
			config.UserID = authResult.UserID
		}

		// Merge auth metadata into session metadata
		if authResult.Metadata != nil {
			if config.Metadata == nil {
				config.Metadata = make(map[string]any)
			}
			for k, v := range authResult.Metadata {
				config.Metadata[k] = v
			}
		}

		backend.AddSpanAttributes(span, map[string]any{
			"authenticated": true,
			"auth_user_id":  authResult.UserID,
		})
	}

	// Authorize operation if AuthHook is provided (T326, FR-021, FR-022)
	if b.config.AuthHook != nil {
		authorized, err := b.config.AuthHook.Authorize(ctx, config.UserID, "create_session")
		if err != nil {
			backend.RecordSpanError(span, err)
			return nil, backend.NewBackendError("CreateSession", backend.ErrCodeAuthorizationFailed, err)
		}

		if !authorized {
			err := backend.NewBackendError("CreateSession", backend.ErrCodeAuthorizationFailed,
				errors.New("user not authorized to create session"))
			backend.RecordSpanError(span, err)
			return nil, err
		}
	}

	// Rate limiting: Check if request is allowed (T327, FR-023, FR-024)
	rateLimiter := internal.GetOrCreateRateLimiter(b.config.Config)
	allowed, err := rateLimiter.Allow(ctx, config.UserID)
	if err != nil {
		backend.RecordSpanError(span, err)
		return nil, backend.WrapError("CreateSession", err)
	}

	if !allowed {
		// Wait for rate limit (T327, FR-024)
		if err := rateLimiter.Wait(ctx, config.UserID); err != nil {
			backend.RecordSpanError(span, err)
			return nil, backend.NewBackendError("CreateSession", backend.ErrCodeRateLimitExceeded, err)
		}
	}

	// Telephony hook: Route call if telephony hook is provided (T331, FR-015)
	var roomName string
	if b.config.TelephonyHook != nil && config.Transport == "telephony" {
		// Extract phone number from metadata
		phoneNumber := ""
		if config.Metadata != nil {
			if pn, ok := config.Metadata["phone_number"].(string); ok {
				phoneNumber = pn
			}
		}

		if phoneNumber != "" {
			// Route call via telephony hook
			providerName, err := b.config.TelephonyHook.RouteCall(ctx, phoneNumber, config.Metadata)
			if err != nil {
				backend.LogWithOTELContext(ctx, slog.LevelWarn, "Telephony hook routing failed, using default",
					"error", err, "phone_number", phoneNumber)
			} else if providerName != "" && providerName != b.config.Provider {
				// Hook suggests different provider - log for monitoring
				backend.LogWithOTELContext(ctx, slog.LevelInfo, "Telephony hook suggests different provider",
					"current_provider", b.config.Provider, "suggested_provider", providerName,
					"phone_number", phoneNumber)
			}
		}
	}

	// Generate room name if not provided
	if roomName == "" {
		roomName = b.config.RoomName
		if roomName == "" {
			roomName = "room-" + config.UserID
		}
	}

	// Create LiveKit room with retry logic
	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	var createErr error
	_ = internal.RetryWithBackoff(ctx, retryConfig, "LiveKitBackend.CreateSession", func() error {
		_, createErr = roomService.CreateRoom(ctx, &livekit.CreateRoomRequest{
			Name: roomName,
		})
		return createErr
	})

	if err != nil {
		backend.RecordSpanError(span, err)
		backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to create LiveKit room", "error", err, "room_name", roomName)
		return nil, backend.WrapError("CreateSession", err)
	}

	// Create LiveKit session
	session, err := NewLiveKitSession(b.config, config, roomName, roomService)
	if err != nil {
		backend.RecordSpanError(span, err)
		return nil, backend.WrapError("CreateSession", err)
	}

	// Add to session manager
	if err := b.sessionManager.AddSession(session.GetID(), session); err != nil {
		backend.RecordSpanError(span, err)
		return nil, backend.WrapError("CreateSession", err)
	}

	// Record session creation time (T179 - should be <2 seconds per SC-007)
	creationTime := time.Since(startTime)
	if b.metrics != nil {
		b.metrics.RecordSessionCreationTime(ctx, "livekit", creationTime)
	}
	backend.AddSpanAttributes(span, map[string]any{
		"session_id":       session.GetID(),
		"room_name":        roomName,
		"creation_time_ms": creationTime.Milliseconds(),
	})

	if creationTime > 2*time.Second {
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "Session creation time exceeded target",
			"creation_time_ms", creationTime.Milliseconds(), "target_ms", 2000)
	}

	span.SetStatus(codes.Ok, "session created successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "LiveKit session created successfully",
		"session_id", session.GetID(), "room_name", roomName, "creation_time_ms", creationTime.Milliseconds())
	return session, nil
}

// GetSession retrieves a voice session by ID.
func (b *LiveKitBackend) GetSession(ctx context.Context, sessionID string) (iface.VoiceSession, error) {
	return b.sessionManager.GetSession(sessionID)
}

// ListSessions returns all active voice sessions.
func (b *LiveKitBackend) ListSessions(ctx context.Context) ([]iface.VoiceSession, error) {
	return b.sessionManager.ListSessions(), nil
}

// CloseSession closes a voice session, closing the LiveKit room and cleaning up.
func (b *LiveKitBackend) CloseSession(ctx context.Context, sessionID string) error {
	session, err := b.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Get room name from session (stored in metadata)
	livekitSession, ok := session.(*LiveKitSession)
	if !ok {
		return backend.NewBackendError("CloseSession", backend.ErrCodeSessionNotFound,
			errors.New("session is not a LiveKit session"))
	}

	roomName := livekitSession.GetRoomName()

	// Delete LiveKit room
	b.mu.RLock()
	roomService := b.roomService
	b.mu.RUnlock()

	if roomService != nil {
		_, err := roomService.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
			Room: roomName,
		})
		if err != nil {
			// Log error but continue with session cleanup
			_ = err
		}
	}

	// Close session in session manager
	return b.sessionManager.CloseSession(ctx, sessionID)
}

// HealthCheck checks the health status of the LiveKit server.
func (b *LiveKitBackend) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.healthStatus.LastCheck = time.Now()

	// Check connection state
	if b.connectionState != iface.ConnectionStateConnected {
		// Attempt to reconnect if disconnected
		if b.connectionState == iface.ConnectionStateDisconnected || b.connectionState == iface.ConnectionStateError {
			b.connectionState = iface.ConnectionStateReconnecting
			go b.reconnect(ctx)
		}
		b.healthStatus.Status = "unhealthy"
		b.healthStatus.Details["reason"] = "not connected"
		b.healthStatus.Details["connection_state"] = string(b.connectionState)
		return b.healthStatus, nil
	}

	// Try to list rooms as a health check
	if b.roomService != nil {
		_, err := b.roomService.ListRooms(ctx, &livekit.ListRoomsRequest{})
		if err != nil {
			// Connection may have been lost, mark for reconnection
			if backend.IsRetryableError(err) {
				b.connectionState = iface.ConnectionStateReconnecting
				go b.reconnect(ctx)
			} else {
				b.connectionState = iface.ConnectionStateError
			}
			b.healthStatus.Status = "unhealthy"
			b.healthStatus.Details["reason"] = err.Error()
			b.healthStatus.Details["connection_state"] = string(b.connectionState)
			return b.healthStatus, nil
		}
	}

	// Comprehensive health status tracking (T299, T301)
	activeSessions := b.sessionManager.GetActiveSessionCount()
	maxSessions := b.config.MaxConcurrentSessions

	// Determine health status: healthy, degraded, or unhealthy
	status := "healthy"

	// Check if we're approaching session limits (degraded state)
	if maxSessions > 0 && activeSessions >= int(float64(maxSessions)*0.9) {
		status = "degraded"
		b.healthStatus.Details["reason"] = "approaching session limit"
	}

	// Check connection quality
	if b.connectionState != iface.ConnectionStateConnected {
		status = "unhealthy"
		b.healthStatus.Details["reason"] = "connection not established"
	}

	b.healthStatus.Status = status
	b.healthStatus.Details["active_sessions"] = activeSessions
	b.healthStatus.Details["max_sessions"] = maxSessions
	b.healthStatus.Details["connection_state"] = string(b.connectionState)
	b.healthStatus.Details["session_utilization"] = float64(activeSessions) / float64(maxSessions) * 100

	return b.healthStatus, nil
}

// reconnect attempts to reconnect to LiveKit server.
func (b *LiveKitBackend) reconnect(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connectionState != iface.ConnectionStateReconnecting {
		return
	}

	retryConfig := &internal.RetryConfig{
		MaxRetries: b.config.MaxRetries,
		Delay:      b.config.RetryDelay,
		Backoff:    2.0,
	}

	err := internal.RetryWithBackoff(ctx, retryConfig, "LiveKitBackend.reconnect", func() error {
		roomService, err := lksdkwrapper.NewRoomServiceClient(b.config.URL, b.config.APIKey, b.config.APISecret)
		if err != nil {
			return err
		}

		// Test connection
		_, err = roomService.ListRooms(ctx, &livekit.ListRoomsRequest{})
		if err != nil {
			return err
		}

		b.roomService = roomService
		b.connectionState = iface.ConnectionStateConnected
		return nil
	})

	if err != nil {
		b.connectionState = iface.ConnectionStateError
		backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to reconnect to LiveKit", "error", err)
	} else {
		backend.LogWithOTELContext(ctx, slog.LevelInfo, "Successfully reconnected to LiveKit")
	}
}

// GetConnectionState returns the current connection state.
func (b *LiveKitBackend) GetConnectionState() iface.ConnectionState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connectionState
}

// GetActiveSessionCount returns the number of active sessions.
func (b *LiveKitBackend) GetActiveSessionCount() int {
	return b.sessionManager.GetActiveSessionCount()
}

// GetConfig returns the backend configuration.
func (b *LiveKitBackend) GetConfig() *iface.Config {
	return b.config.Config
}

// UpdateConfig updates the backend configuration with validation.
func (b *LiveKitBackend) UpdateConfig(ctx context.Context, config *iface.Config) error {
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate LiveKit-specific config
	provider := NewLiveKitProvider()
	if err := provider.ValidateConfig(ctx, config); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Update config
	b.config = NewLiveKitConfig(config)

	// If connection details changed, reconnect
	if b.connectionState == iface.ConnectionStateConnected {
		// Reinitialize room service with new credentials
		roomService, err := lksdkwrapper.NewRoomServiceClient(b.config.URL, b.config.APIKey, b.config.APISecret)
		if err != nil {
			return backend.NewBackendError("UpdateConfig", backend.ErrCodeConnectionFailed, err)
		}
		b.roomService = roomService
	}

	return nil
}
