package twilio

import (
	"context"
	"fmt"
	"sync"
	"time"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/twilio/twilio-go"
	twiliov2010 "github.com/twilio/twilio-go/rest/api/v2010"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TwilioBackend implements the VoiceBackend interface for Twilio.
type TwilioBackend struct {
	config          *TwilioConfig
	client          *twilio.RestClient
	sessions        map[string]vbiface.VoiceSession
	healthStatus    *vbiface.HealthStatus
	metrics         *Metrics
	connectionState vbiface.ConnectionState
	mu              sync.RWMutex
}

// NewTwilioBackend creates a new Twilio backend.
func NewTwilioBackend(config *TwilioConfig) (*TwilioBackend, error) {
	var metrics *Metrics
	if config.EnableMetrics {
		// Initialize metrics using OTEL global meter and tracer
		// This follows the standard pattern used across all Beluga AI packages
		meter := otel.Meter("beluga.voice.providers.twilio")
		tracer := otel.Tracer("beluga.voice.providers.twilio")
		var err error
		metrics, err = NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics as fallback
			metrics = NoOpMetrics()
		}
	} else {
		// Use no-op metrics when metrics are disabled
		metrics = NoOpMetrics()
	}

	return &TwilioBackend{
		config:          config,
		sessions:        make(map[string]vbiface.VoiceSession),
		connectionState: vbiface.ConnectionStateDisconnected,
		healthStatus: &vbiface.HealthStatus{
			Status:    "unknown",
			Details:   make(map[string]any),
			LastCheck: time.Now(),
		},
		metrics: metrics,
	}, nil
}

// Start starts the Twilio backend, initializing the Twilio client.
func (b *TwilioBackend) Start(ctx context.Context) error {
	startTime := time.Now()
	ctx, span := b.startSpan(ctx, "TwilioBackend.Start")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connectionState == vbiface.ConnectionStateConnected {
		span.SetStatus(codes.Ok, "already connected")
		return nil
	}

	b.connectionState = vbiface.ConnectionStateConnecting
	span.SetAttributes(attribute.String("connection_state", string(b.connectionState)))

	// Initialize Twilio REST client
	clientParams := twilio.ClientParams{
		Username: b.config.AccountSID,
		Password: b.config.AuthToken,
	}
	b.client = twilio.NewRestClientWithParams(clientParams)

	// Verify connectivity by making a test API call
	_, err := b.client.Api.FetchAccount(b.config.AccountSID)
	if err != nil {
		b.connectionState = vbiface.ConnectionStateError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordOperation(ctx, "start", time.Since(startTime), false)
		}
		return NewTwilioError("Start", ErrCodeTwilioAuthError, err)
	}

	b.connectionState = vbiface.ConnectionStateConnected
	span.SetStatus(codes.Ok, "connected")
	span.SetAttributes(attribute.String("connection_state", string(b.connectionState)))

	if b.metrics != nil {
		b.metrics.RecordOperation(ctx, "start", time.Since(startTime), true)
	}

	return nil
}

// Stop stops the Twilio backend gracefully.
func (b *TwilioBackend) Stop(ctx context.Context) error {
	startTime := time.Now()
	ctx, span := b.startSpan(ctx, "TwilioBackend.Stop")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all active sessions
	for sessionID, session := range b.sessions {
		if err := b.closeSessionInternal(ctx, sessionID, session); err != nil {
			span.RecordError(err)
		}
	}

	b.sessions = make(map[string]vbiface.VoiceSession)
	b.connectionState = vbiface.ConnectionStateDisconnected
	span.SetStatus(codes.Ok, "stopped")

	if b.metrics != nil {
		b.metrics.RecordOperation(ctx, "stop", time.Since(startTime), true)
	}

	return nil
}

// CreateSession creates a new voice session (Twilio call).
func (b *TwilioBackend) CreateSession(ctx context.Context, config *vbiface.SessionConfig) (vbiface.VoiceSession, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.CreateSession")
	defer span.End()

	startTime := time.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connectionState != vbiface.ConnectionStateConnected {
		err := NewTwilioError("CreateSession", ErrCodeTwilioNetworkError, "backend not connected")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Create Twilio Call resource
	// Note: SessionConfig uses ConnectionURL and Metadata, not To/From
	to := b.config.PhoneNumber // Default to configured number
	from := b.config.PhoneNumber
	if config.Metadata != nil {
		if t, ok := config.Metadata["to"].(string); ok && t != "" {
			to = t
		}
		if f, ok := config.Metadata["from"].(string); ok && f != "" {
			from = f
		}
	}

	callParams := &twiliov2010.CreateCallParams{}
	callParams.SetTo(to)
	callParams.SetFrom(from)

	if b.config.WebhookURL != "" {
		callParams.SetUrl(b.config.WebhookURL)
	}

	if b.config.StatusCallbackURL != "" {
		callParams.SetStatusCallback(b.config.StatusCallbackURL)
		events := []string{"initiated", "ringing", "answered", "completed"}
		callParams.SetStatusCallbackEvent(events)
	}

	call, err := b.client.Api.CreateCall(callParams)
	if err != nil {
		err = MapTwilioError("CreateSession", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordCall(ctx, "", time.Since(startTime), false)
		}
		return nil, err
	}

	span.SetAttributes(
		attribute.String("call_sid", getStringValue(call.Sid)),
		attribute.String("call_to", getStringValue(call.To)),
		attribute.String("call_from", getStringValue(call.From)),
	)

	// Create voice session using adapter
	sessionID := getStringValue(call.Sid)
	session, err := NewTwilioSessionAdapter(ctx, sessionID, b.config, config, b)
	if err != nil {
		err = MapTwilioError("CreateSession", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if b.metrics != nil {
			b.metrics.RecordCall(ctx, sessionID, time.Since(startTime), false)
		}
		return nil, err
	}

	// Store session (already holding lock from function start)
	b.sessions[sessionID] = session

	if b.metrics != nil {
		b.metrics.RecordCall(ctx, sessionID, time.Since(startTime), true)
		b.metrics.IncrementActiveCalls(ctx)
	}

	span.SetStatus(codes.Ok, "session created")
	return session, nil
}

// GetSession retrieves a voice session by ID.
func (b *TwilioBackend) GetSession(ctx context.Context, sessionID string) (vbiface.VoiceSession, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.GetSession")
	defer span.End()

	span.SetAttributes(attribute.String("session_id", sessionID))

	b.mu.RLock()
	session, exists := b.sessions[sessionID]
	b.mu.RUnlock()

	if !exists {
		err := NewTwilioError("GetSession", "session_not_found", fmt.Errorf("session %s not found", sessionID))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "session found")
	return session, nil
}

// ListSessions returns all active voice sessions.
func (b *TwilioBackend) ListSessions(ctx context.Context) ([]vbiface.VoiceSession, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.ListSessions")
	defer span.End()

	b.mu.RLock()
	sessions := make([]vbiface.VoiceSession, 0, len(b.sessions))
	for _, session := range b.sessions {
		sessions = append(sessions, session)
	}
	b.mu.RUnlock()

	span.SetAttributes(attribute.Int("session_count", len(sessions)))
	span.SetStatus(codes.Ok, "sessions listed")
	return sessions, nil
}

// CloseSession closes a voice session.
func (b *TwilioBackend) CloseSession(ctx context.Context, sessionID string) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.CloseSession")
	defer span.End()

	span.SetAttributes(attribute.String("session_id", sessionID))

	b.mu.Lock()
	session, exists := b.sessions[sessionID]
	if !exists {
		b.mu.Unlock()
		err := NewTwilioError("CloseSession", "session_not_found", fmt.Errorf("session %s not found", sessionID))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	b.mu.Unlock()

	return b.closeSessionInternal(ctx, sessionID, session)
}

// closeSessionInternal closes a session internally (called with lock held or after lock released).
func (b *TwilioBackend) closeSessionInternal(ctx context.Context, sessionID string, session vbiface.VoiceSession) error {
	// Update Twilio Call resource status if needed
	// Close WebSocket stream if active
	// Remove from sessions map

	b.mu.Lock()
	delete(b.sessions, sessionID)
	b.mu.Unlock()

	if b.metrics != nil {
		b.metrics.DecrementActiveCalls(ctx)
	}

	return nil
}

// HealthCheck performs a health check on the backend.
func (b *TwilioBackend) HealthCheck(ctx context.Context) (*vbiface.HealthStatus, error) {
	ctx, span := b.startSpan(ctx, "TwilioBackend.HealthCheck")
	defer span.End()

	startTime := time.Now()

	b.mu.RLock()
	connectionState := b.connectionState
	b.mu.RUnlock()

	// Verify API connectivity
	var status string
	var errors []string

	if connectionState != vbiface.ConnectionStateConnected {
		status = "unhealthy"
		errors = append(errors, "backend not connected")
	} else {
		// Test API call
		_, err := b.client.Api.FetchAccount(b.config.AccountSID)
		if err != nil {
			status = "unhealthy"
			errors = append(errors, fmt.Sprintf("API connectivity failed: %v", err))
		} else {
			status = "healthy"
		}
	}

	details := map[string]any{
		"connection_state": string(connectionState),
		"active_sessions":  len(b.sessions),
	}
	if len(errors) > 0 {
		details["errors"] = errors
	}

	healthStatus := &vbiface.HealthStatus{
		Status:    status,
		LastCheck: time.Now(),
		Details:   details,
	}

	b.mu.Lock()
	b.healthStatus = healthStatus
	b.mu.Unlock()

	span.SetAttributes(
		attribute.String("health_status", status),
		attribute.Int("active_sessions", len(b.sessions)),
	)

	if b.metrics != nil {
		b.metrics.RecordOperation(ctx, "health_check", time.Since(startTime), len(errors) == 0)
	}

	span.SetStatus(codes.Ok, "health check completed")
	return healthStatus, nil
}

// GetConnectionState returns the current connection state.
func (b *TwilioBackend) GetConnectionState() vbiface.ConnectionState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connectionState
}

// GetActiveSessionCount returns the number of active sessions.
func (b *TwilioBackend) GetActiveSessionCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.sessions)
}

// GetConfig returns the backend configuration.
func (b *TwilioBackend) GetConfig() *vbiface.Config {
	return b.config.Config
}

// UpdateConfig updates the backend configuration.
func (b *TwilioBackend) UpdateConfig(ctx context.Context, config *vbiface.Config) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.UpdateConfig")
	defer span.End()

	// Validate new config
	twilioConfig := NewTwilioConfig(config)
	if err := twilioConfig.Validate(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	b.mu.Lock()
	b.config = twilioConfig
	b.mu.Unlock()

	span.SetStatus(codes.Ok, "config updated")
	return nil
}

// startSpan starts an OTEL span for tracing.
func (b *TwilioBackend) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if b.metrics != nil && b.metrics.Tracer() != nil {
		return b.metrics.Tracer().Start(ctx, operation)
	}
	return ctx, trace.SpanFromContext(ctx)
}

// Helper functions for handling Twilio SDK pointer types

// getStringValue safely extracts a string from a pointer.
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// getTimeValue safely extracts a time.Time from a pointer.
func getTimeValue(ptr *time.Time) time.Time {
	if ptr == nil {
		return time.Time{}
	}
	return *ptr
}
