package credential

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// CredentialIssuer manages the lifecycle of agent credentials. Implementations
// must be safe for concurrent use.
type CredentialIssuer interface {
	// Issue creates a new credential for the given agent with the specified
	// permissions and TTL.
	Issue(ctx context.Context, agentID string, permissions []string, ttl time.Duration) (*AgentCredential, error)

	// Revoke marks a credential as revoked by its ID.
	Revoke(ctx context.Context, credentialID string) error

	// Get retrieves a credential by its ID. Returns a core.ErrNotFound error
	// if the credential does not exist.
	Get(ctx context.Context, credentialID string) (*AgentCredential, error)
}

// issuerOptions holds configuration for InMemoryIssuer.
type issuerOptions struct {
	defaultTTL     time.Duration
	maxCredentials int
}

// IssuerOption configures an InMemoryIssuer.
type IssuerOption func(*issuerOptions)

// WithDefaultTTL sets the default time-to-live for issued credentials.
// Defaults to 5 minutes.
func WithDefaultTTL(d time.Duration) IssuerOption {
	return func(o *issuerOptions) { o.defaultTTL = d }
}

// WithMaxCredentials sets the maximum number of credentials the issuer will
// store. When the limit is reached, Issue returns an error. Defaults to 10000.
func WithMaxCredentials(max int) IssuerOption {
	return func(o *issuerOptions) { o.maxCredentials = max }
}

func defaultIssuerOptions() issuerOptions {
	return issuerOptions{
		defaultTTL:     5 * time.Minute,
		maxCredentials: 10000,
	}
}

// InMemoryIssuer is a thread-safe, bounded in-memory implementation of
// CredentialIssuer.
type InMemoryIssuer struct {
	mu    sync.RWMutex
	store map[string]*AgentCredential
	opts  issuerOptions
}

// Compile-time check that InMemoryIssuer implements CredentialIssuer.
var _ CredentialIssuer = (*InMemoryIssuer)(nil)

// NewInMemoryIssuer creates a new InMemoryIssuer with the given options.
func NewInMemoryIssuer(opts ...IssuerOption) *InMemoryIssuer {
	o := defaultIssuerOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &InMemoryIssuer{
		store: make(map[string]*AgentCredential),
		opts:  o,
	}
}

// Issue creates a new credential for the given agent. If ttl is zero, the
// issuer's default TTL is used.
func (iss *InMemoryIssuer) Issue(_ context.Context, agentID string, permissions []string, ttl time.Duration) (*AgentCredential, error) {
	if agentID == "" {
		return nil, core.NewError("credential.issue", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}
	if len(permissions) == 0 {
		return nil, core.NewError("credential.issue", core.ErrInvalidInput, "permissions must not be empty", nil)
	}

	if ttl <= 0 {
		ttl = iss.opts.defaultTTL
	}

	id, err := generateID()
	if err != nil {
		return nil, core.NewError("credential.issue", core.ErrAuth, "failed to generate credential ID", err)
	}

	now := time.Now()
	cred := &AgentCredential{
		ID:          id,
		AgentID:     agentID,
		Permissions: append([]string{}, permissions...), // defensive copy
		IssuedAt:    now,
		ExpiresAt:   now.Add(ttl),
		Metadata:    make(map[string]string),
	}

	iss.mu.Lock()
	defer iss.mu.Unlock()

	if len(iss.store) >= iss.opts.maxCredentials {
		return nil, core.NewError("credential.issue", core.ErrBudgetExhausted, "credential store is full", nil)
	}

	iss.store[id] = cred
	return cloneCredential(cred), nil
}

// cloneCredential returns a deep copy of a credential so that callers cannot
// mutate the issuer's internal state (and so concurrent reads on the returned
// value do not race with the issuer's own writes under its mutex).
func cloneCredential(c *AgentCredential) *AgentCredential {
	if c == nil {
		return nil
	}
	cp := *c
	if len(c.Permissions) > 0 {
		cp.Permissions = append([]string{}, c.Permissions...)
	}
	if len(c.Metadata) > 0 {
		cp.Metadata = make(map[string]string, len(c.Metadata))
		for k, v := range c.Metadata {
			cp.Metadata[k] = v
		}
	}
	return &cp
}

// Revoke marks a credential as revoked. Returns an error if the credential
// does not exist.
func (iss *InMemoryIssuer) Revoke(_ context.Context, credentialID string) error {
	iss.mu.Lock()
	defer iss.mu.Unlock()

	cred, ok := iss.store[credentialID]
	if !ok {
		return core.NewError("credential.revoke", core.ErrNotFound, "credential not found", nil)
	}
	cred.Revoked = true
	return nil
}

// Get retrieves a credential by its ID. Returns a core.ErrNotFound error if
// the credential does not exist.
func (iss *InMemoryIssuer) Get(_ context.Context, credentialID string) (*AgentCredential, error) {
	iss.mu.RLock()
	defer iss.mu.RUnlock()

	cred, ok := iss.store[credentialID]
	if !ok {
		return nil, core.NewError("credential.get", core.ErrNotFound, "credential not found", nil)
	}
	return cloneCredential(cred), nil
}

// Expired returns credentials that have passed their expiry time but have not
// yet been revoked. This is used by AutoRevoker to find credentials that need
// automatic revocation. Returned credentials are deep copies.
func (iss *InMemoryIssuer) Expired() []*AgentCredential {
	iss.mu.RLock()
	defer iss.mu.RUnlock()

	var result []*AgentCredential
	for _, cred := range iss.store {
		if cred.IsExpired() && !cred.Revoked {
			result = append(result, cloneCredential(cred))
		}
	}
	return result
}

// Cleanup removes credentials that are both expired and revoked from the
// store. This prevents unbounded growth that would otherwise cause the
// capacity check in Issue() to fail permanently. Returns the number of
// credentials removed.
func (iss *InMemoryIssuer) Cleanup(_ context.Context) int {
	iss.mu.Lock()
	defer iss.mu.Unlock()

	removed := 0
	for id, cred := range iss.store {
		if cred.Revoked && cred.IsExpired() {
			delete(iss.store, id)
			removed++
		}
	}
	return removed
}

// generateID produces a cryptographically random credential ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "cred-" + hex.EncodeToString(b), nil
}

// Count returns the number of stored credentials. This is primarily useful
// for testing.
func (iss *InMemoryIssuer) Count() int {
	iss.mu.RLock()
	defer iss.mu.RUnlock()
	return len(iss.store)
}

// DefaultTTL returns the issuer's default time-to-live for credentials.
func (iss *InMemoryIssuer) DefaultTTL() time.Duration {
	return iss.opts.defaultTTL
}

// String returns a human-readable summary of the issuer.
func (iss *InMemoryIssuer) String() string {
	iss.mu.RLock()
	defer iss.mu.RUnlock()
	return fmt.Sprintf("InMemoryIssuer(count=%d, max=%d, ttl=%s)", len(iss.store), iss.opts.maxCredentials, iss.opts.defaultTTL)
}
