package credential

import "time"

// AgentCredential represents a time-bounded, scoped credential issued to an
// agent. It carries explicit permissions and expiry information.
type AgentCredential struct {
	// ID uniquely identifies this credential.
	ID string

	// AgentID identifies the agent this credential was issued to.
	AgentID string

	// Permissions lists the actions this credential grants.
	Permissions []string

	// IssuedAt records when the credential was created.
	IssuedAt time.Time

	// ExpiresAt records when the credential becomes invalid.
	ExpiresAt time.Time

	// Revoked indicates whether the credential has been explicitly revoked.
	Revoked bool

	// Metadata carries optional key-value pairs for auditing and tracing.
	Metadata map[string]string
}

// IsExpired reports whether the credential has passed its expiry time.
func (c *AgentCredential) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsValid reports whether the credential is neither expired nor revoked.
func (c *AgentCredential) IsValid() bool {
	return !c.Revoked && !c.IsExpired()
}

// HasPermission reports whether the credential includes the given permission.
func (c *AgentCredential) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}
