package mock

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
)

// MockBackend implements the VoiceBackend interface for testing.
type MockBackend struct {
	config          *MockConfig
	sessionManager  *internal.SessionManager
	healthStatus    *vbiface.HealthStatus
	connectionState vbiface.ConnectionState
	mu              sync.RWMutex
}

// NewMockBackend creates a new mock backend.
func NewMockBackend(config *MockConfig) (*MockBackend, error) {
	return &MockBackend{
		config:          config,
		sessionManager:  internal.NewSessionManager(config.Config),
		connectionState: vbiface.ConnectionStateDisconnected,
		healthStatus: &vbiface.HealthStatus{
			Status:    "healthy",
			Details:   make(map[string]any),
			LastCheck: time.Now(),
		},
	}, nil
}

// Start starts the mock backend.
func (b *MockBackend) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.connectionState = vbiface.ConnectionStateConnected
	return nil
}

// Stop stops the mock backend gracefully.
func (b *MockBackend) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all active sessions
	sessions := b.sessionManager.ListSessions()
	for _, session := range sessions {
		_ = session.Stop(ctx)
	}

	b.connectionState = vbiface.ConnectionStateDisconnected
	return nil
}

// CreateSession creates a new voice session.
func (b *MockBackend) CreateSession(ctx context.Context, config *vbiface.SessionConfig) (vbiface.VoiceSession, error) {
	session, err := NewMockSession(b.config, config)
	if err != nil {
		return nil, err
	}

	// Add to session manager
	if err := b.sessionManager.AddSession(session.GetID(), session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a voice session by ID.
func (b *MockBackend) GetSession(ctx context.Context, sessionID string) (vbiface.VoiceSession, error) {
	return b.sessionManager.GetSession(sessionID)
}

// ListSessions returns all active voice sessions.
func (b *MockBackend) ListSessions(ctx context.Context) ([]vbiface.VoiceSession, error) {
	return b.sessionManager.ListSessions(), nil
}

// CloseSession closes a voice session.
func (b *MockBackend) CloseSession(ctx context.Context, sessionID string) error {
	return b.sessionManager.CloseSession(ctx, sessionID)
}

// HealthCheck checks the health status of the backend.
func (b *MockBackend) HealthCheck(ctx context.Context) (*vbiface.HealthStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.healthStatus.LastCheck = time.Now()
	return b.healthStatus, nil
}

// GetConnectionState returns the current connection state.
func (b *MockBackend) GetConnectionState() vbiface.ConnectionState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connectionState
}

// GetActiveSessionCount returns the number of active sessions.
func (b *MockBackend) GetActiveSessionCount() int {
	return b.sessionManager.GetActiveSessionCount()
}

// GetConfig returns the backend configuration.
func (b *MockBackend) GetConfig() *vbiface.Config {
	return b.config.Config
}

// UpdateConfig updates the backend configuration with validation.
func (b *MockBackend) UpdateConfig(ctx context.Context, config *vbiface.Config) error {
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Config = config
	return nil
}
