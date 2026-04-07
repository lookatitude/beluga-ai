package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compile-time assertion that InMemorySessionService implements SessionService.
var _ SessionService = (*InMemorySessionService)(nil)

// sessionOptions holds configuration for InMemorySessionService.
type sessionOptions struct {
	ttl      time.Duration
	tenantID string
}

// defaultSessionOptions returns the default options: no TTL, no tenant scope.
func defaultSessionOptions() sessionOptions {
	return sessionOptions{}
}

// SessionOption is a functional option for configuring InMemorySessionService.
type SessionOption func(*sessionOptions)

// WithSessionTTL sets the time-to-live for sessions managed by the service.
// Sessions whose ExpiresAt has passed will be treated as not found on Get.
// A zero duration means sessions never expire.
func WithSessionTTL(d time.Duration) SessionOption {
	return func(o *sessionOptions) {
		o.ttl = d
	}
}

// WithSessionTenantID sets the default tenant ID applied to all newly created
// sessions. It can be overridden per-session after creation.
func WithSessionTenantID(tenantID string) SessionOption {
	return func(o *sessionOptions) {
		o.tenantID = tenantID
	}
}

// InMemorySessionService is a thread-safe, in-memory implementation of
// SessionService. It is suitable for testing and single-process deployments.
type InMemorySessionService struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	opts     sessionOptions
}

// NewInMemorySessionService creates a new InMemorySessionService with the
// supplied options. Zero-config works: just call NewInMemorySessionService().
func NewInMemorySessionService(opts ...SessionOption) *InMemorySessionService {
	o := defaultSessionOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &InMemorySessionService{
		sessions: make(map[string]*Session),
		opts:     o,
	}
}

// Create allocates a new Session for the given agentID, assigns a
// crypto-random ID, and stores it. The returned session is a copy.
func (s *InMemorySessionService) Create(ctx context.Context, agentID string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	id, err := generateSessionID()
	if err != nil {
		return nil, core.NewError("runtime.session.create", core.ErrInvalidInput, "failed to generate session ID", err)
	}

	now := time.Now().UTC()
	session := &Session{
		ID:        id,
		AgentID:   agentID,
		TenantID:  s.opts.tenantID,
		State:     make(map[string]any),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if s.opts.ttl > 0 {
		session.ExpiresAt = now.Add(s.opts.ttl)
	}

	s.mu.Lock()
	s.sessions[id] = cloneSession(session)
	s.mu.Unlock()

	return session, nil
}

// Get retrieves the Session identified by sessionID. It returns a core.Error
// with code ErrNotFound if the session does not exist or has expired.
func (s *InMemorySessionService) Get(ctx context.Context, sessionID string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	stored, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return nil, core.NewError("runtime.session.get", core.ErrNotFound,
			"session not found: "+sessionID, nil)
	}

	if !stored.ExpiresAt.IsZero() && time.Now().UTC().After(stored.ExpiresAt) {
		return nil, core.NewError("runtime.session.get", core.ErrNotFound,
			"session expired: "+sessionID, nil)
	}

	return cloneSession(stored), nil
}

// Update persists the current state of an existing session. The session is
// stored by value (a deep copy is made). It returns a core.Error with code
// ErrNotFound if the session does not exist.
func (s *InMemorySessionService) Update(ctx context.Context, session *Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if session == nil {
		return core.NewError("runtime.session.update", core.ErrInvalidInput, "session must not be nil", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[session.ID]; !ok {
		return core.NewError("runtime.session.update", core.ErrNotFound,
			"session not found: "+session.ID, nil)
	}

	copied := cloneSession(session)
	copied.UpdatedAt = time.Now().UTC()
	s.sessions[session.ID] = copied
	return nil
}

// Delete removes the session identified by sessionID. It returns a core.Error
// with code ErrNotFound if no such session exists.
func (s *InMemorySessionService) Delete(ctx context.Context, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sessionID]; !ok {
		return core.NewError("runtime.session.delete", core.ErrNotFound,
			"session not found: "+sessionID, nil)
	}

	delete(s.sessions, sessionID)
	return nil
}

// generateSessionID returns a 16-byte, hex-encoded, crypto-random string.
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// cloneSession returns a deep copy of s so callers cannot mutate stored
// sessions through the returned pointer.
func cloneSession(s *Session) *Session {
	cp := *s

	if s.State != nil {
		cp.State = make(map[string]any, len(s.State))
		for k, v := range s.State {
			cp.State[k] = v
		}
	}

	if len(s.Turns) > 0 {
		cp.Turns = make([]schema.Turn, len(s.Turns))
		copy(cp.Turns, s.Turns)
	}

	return &cp
}
