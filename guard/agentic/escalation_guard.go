package agentic

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/guard"
)

// Compile-time check.
var _ guard.Guard = (*PrivilegeEscalationGuard)(nil)

// defaultMaxHandoffDepth is the default maximum handoff chain depth.
const defaultMaxHandoffDepth = 5

// PrivilegeEscalationGuard validates agent handoff chains to prevent
// privilege escalation. It enforces that delegated permissions are a subset
// of the delegator's permissions, limits handoff chain depth, and rejects
// handoffs to disallowed agents. It addresses OWASP AG04 (Privilege
// Escalation).
type PrivilegeEscalationGuard struct {
	maxDepth      int
	agentPerms    map[string]map[string]bool // agent -> set of permissions
	blockedAgents map[string]bool
	requireSubset bool // enforce permission intersection on delegation
	strictMode    bool // reject handoffs involving agents with no registered permissions
}

// EscalationOption configures a PrivilegeEscalationGuard.
type EscalationOption func(*PrivilegeEscalationGuard)

// WithMaxHandoffDepth sets the maximum allowed handoff chain depth.
func WithMaxHandoffDepth(depth int) EscalationOption {
	return func(g *PrivilegeEscalationGuard) {
		if depth > 0 {
			g.maxDepth = depth
		}
	}
}

// WithAgentPermissions registers the permission set for an agent. Permissions
// are arbitrary strings such as "read:files", "execute:code".
func WithAgentPermissions(agent string, perms ...string) EscalationOption {
	return func(g *PrivilegeEscalationGuard) {
		set := make(map[string]bool, len(perms))
		for _, p := range perms {
			set[p] = true
		}
		g.agentPerms[agent] = set
	}
}

// WithBlockedAgents marks agents that must never receive handoffs.
func WithBlockedAgents(agents ...string) EscalationOption {
	return func(g *PrivilegeEscalationGuard) {
		for _, a := range agents {
			g.blockedAgents[a] = true
		}
	}
}

// WithPermissionSubsetEnforcement enables or disables the requirement that a
// target agent's permissions must be a subset of the source agent's
// permissions. Enabled by default.
func WithPermissionSubsetEnforcement(enabled bool) EscalationOption {
	return func(g *PrivilegeEscalationGuard) {
		g.requireSubset = enabled
	}
}

// WithStrictMode causes permission-subset enforcement to reject handoffs when
// either the source or target agent has no registered permission set. Without
// strict mode, the guard fails open when permission metadata is missing, which
// can silently disable protection for newly introduced agents.
func WithStrictMode(enabled bool) EscalationOption {
	return func(g *PrivilegeEscalationGuard) {
		g.strictMode = enabled
	}
}

// NewPrivilegeEscalationGuard creates a PrivilegeEscalationGuard with the
// given options.
func NewPrivilegeEscalationGuard(opts ...EscalationOption) *PrivilegeEscalationGuard {
	g := &PrivilegeEscalationGuard{
		maxDepth:      defaultMaxHandoffDepth,
		agentPerms:    make(map[string]map[string]bool),
		blockedAgents: make(map[string]bool),
		requireSubset: true,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Name returns "privilege_escalation_guard".
func (g *PrivilegeEscalationGuard) Name() string {
	return "privilege_escalation_guard"
}

// Validate checks a handoff request carried in input.Metadata. Expected
// metadata keys:
//
//   - "handoff_chain": []string -- ordered list of agents in the chain
//   - "source_agent": string   -- agent initiating the handoff
//   - "target_agent": string   -- agent receiving the handoff
func (g *PrivilegeEscalationGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	select {
	case <-ctx.Done():
		return guard.GuardResult{}, ctx.Err()
	default:
	}

	targetAgent, _ := input.Metadata["target_agent"].(string)
	sourceAgent, _ := input.Metadata["source_agent"].(string)

	// If no handoff metadata, nothing to validate.
	if targetAgent == "" && sourceAgent == "" {
		return guard.GuardResult{Allowed: true}, nil
	}

	// Blocked agent check.
	if g.blockedAgents[targetAgent] {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("handoff to blocked agent %q is not allowed", targetAgent),
			GuardName: g.Name(),
		}, nil
	}

	// Depth check.
	chain := toStringSlice(input.Metadata["handoff_chain"])
	// The chain after this handoff would include the target.
	effectiveDepth := len(chain) + 1
	if effectiveDepth > g.maxDepth {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("handoff chain depth %d exceeds maximum of %d", effectiveDepth, g.maxDepth),
			GuardName: g.Name(),
		}, nil
	}

	// Permission intersection check.
	if g.requireSubset && sourceAgent != "" && targetAgent != "" {
		if blocked, reason := g.checkPermissionSubset(sourceAgent, targetAgent); blocked {
			return guard.GuardResult{
				Allowed:   false,
				Reason:    reason,
				GuardName: g.Name(),
			}, nil
		}
	}

	return guard.GuardResult{Allowed: true}, nil
}

// checkPermissionSubset verifies that the target agent's permissions are a
// subset of the source agent's permissions. If either agent has no registered
// permissions and strict mode is disabled, the check passes (no restriction).
// In strict mode, missing permission metadata causes the handoff to be
// rejected so newly introduced agents cannot silently bypass protection.
func (g *PrivilegeEscalationGuard) checkPermissionSubset(source, target string) (bool, string) {
	sourcePerms, sourceOK := g.agentPerms[source]
	targetPerms, targetOK := g.agentPerms[target]

	if !sourceOK || !targetOK {
		if g.strictMode {
			missing := make([]string, 0, 2)
			if !sourceOK {
				missing = append(missing, source)
			}
			if !targetOK {
				missing = append(missing, target)
			}
			return true, fmt.Sprintf(
				"strict mode: no registered permissions for agent(s) [%s]",
				strings.Join(missing, ", "),
			)
		}
		// Cannot validate without permission data -- allow by default.
		return false, ""
	}

	var extra []string
	for p := range targetPerms {
		if !sourcePerms[p] {
			extra = append(extra, p)
		}
	}
	if len(extra) > 0 {
		return true, fmt.Sprintf(
			"target agent %q has permissions not held by source %q: [%s]",
			target, source, strings.Join(extra, ", "),
		)
	}
	return false, ""
}

// toStringSlice attempts to convert v to []string. It handles both
// []string and []any with string elements.
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func init() {
	guard.Register("privilege_escalation_guard", func(cfg map[string]any) (guard.Guard, error) {
		return NewPrivilegeEscalationGuard(), nil
	})
}
