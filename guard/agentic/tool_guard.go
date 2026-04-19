package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/guard"
)

// Compile-time check.
var _ guard.Guard = (*ToolMisuseGuard)(nil)

// ToolSchema defines the expected JSON schema for a tool's arguments. When a
// schema is registered for a tool, the guard validates that tool invocations
// conform to it.
type ToolSchema struct {
	// RequiredFields lists the field names that must be present.
	RequiredFields []string

	// MaxFieldCount is the maximum number of fields allowed. Zero means no
	// limit.
	MaxFieldCount int

	// ForbiddenFields lists field names that must not appear.
	ForbiddenFields []string
}

// toolRateState tracks invocation timestamps for a single tool.
type toolRateState struct {
	mu          sync.Mutex
	invocations []time.Time
}

// ToolMisuseGuard validates tool call arguments against registered schemas,
// enforces per-tool rate limits, and restricts tool access to an allowed set
// (capability scoping). It addresses OWASP AG03 (Tool Misuse).
type ToolMisuseGuard struct {
	schemas      map[string]ToolSchema
	allowedTools map[string]bool
	rateLimit    int           // max invocations per window per tool; 0 = unlimited
	rateWindow   time.Duration // sliding window duration

	rateMu     sync.RWMutex
	rateStates map[string]*toolRateState
}

// ToolMisuseOption configures a ToolMisuseGuard.
type ToolMisuseOption func(*ToolMisuseGuard)

// WithToolSchema registers a schema for the named tool.
func WithToolSchema(toolName string, schema ToolSchema) ToolMisuseOption {
	return func(g *ToolMisuseGuard) {
		g.schemas[toolName] = schema
	}
}

// WithAllowedTools restricts tool execution to the named set. An empty set
// means all tools are allowed.
func WithAllowedTools(names ...string) ToolMisuseOption {
	return func(g *ToolMisuseGuard) {
		for _, n := range names {
			g.allowedTools[n] = true
		}
	}
}

// WithToolRateLimit sets the maximum number of invocations per tool within
// the given window. A limit of zero disables rate limiting.
func WithToolRateLimit(limit int, window time.Duration) ToolMisuseOption {
	return func(g *ToolMisuseGuard) {
		g.rateLimit = limit
		g.rateWindow = window
	}
}

// NewToolMisuseGuard creates a ToolMisuseGuard with the given options.
func NewToolMisuseGuard(opts ...ToolMisuseOption) *ToolMisuseGuard {
	g := &ToolMisuseGuard{
		schemas:      make(map[string]ToolSchema),
		allowedTools: make(map[string]bool),
		rateStates:   make(map[string]*toolRateState),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Name returns "tool_misuse_guard".
func (g *ToolMisuseGuard) Name() string {
	return "tool_misuse_guard"
}

// Validate checks the tool call carried in input. The tool name is expected
// in input.Metadata["tool_name"]. The input.Content is treated as JSON
// arguments.
func (g *ToolMisuseGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	select {
	case <-ctx.Done():
		return guard.GuardResult{}, ctx.Err()
	default:
	}

	toolName, _ := input.Metadata["tool_name"].(string)
	if toolName == "" {
		// No tool context -- nothing to validate.
		return guard.GuardResult{Allowed: true}, nil
	}

	// Capability scoping: reject tools not in the allowed set.
	if len(g.allowedTools) > 0 && !g.allowedTools[toolName] {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("tool %q is not in the allowed set", toolName),
			GuardName: g.Name(),
		}, nil
	}

	// Rate limiting.
	if g.rateLimit > 0 {
		if blocked, reason := g.checkRateLimit(toolName); blocked {
			return guard.GuardResult{
				Allowed:   false,
				Reason:    reason,
				GuardName: g.Name(),
			}, nil
		}
	}

	// Schema validation.
	schema, hasSchema := g.schemas[toolName]
	if hasSchema {
		if blocked, reason := validateToolArgs(input.Content, schema); blocked {
			return guard.GuardResult{
				Allowed:   false,
				Reason:    reason,
				GuardName: g.Name(),
			}, nil
		}
	}

	return guard.GuardResult{Allowed: true}, nil
}

// checkRateLimit returns true and a reason if the tool has exceeded its rate
// limit.
func (g *ToolMisuseGuard) checkRateLimit(toolName string) (bool, string) {
	g.rateMu.Lock()
	state, ok := g.rateStates[toolName]
	if !ok {
		state = &toolRateState{}
		g.rateStates[toolName] = state
	}
	g.rateMu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-g.rateWindow)

	// Prune old invocations.
	valid := state.invocations[:0]
	for _, t := range state.invocations {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	state.invocations = valid

	if len(state.invocations) >= g.rateLimit {
		return true, fmt.Sprintf("tool %q exceeded rate limit of %d calls per %v", toolName, g.rateLimit, g.rateWindow)
	}

	state.invocations = append(state.invocations, now)
	return false, ""
}

// validateToolArgs parses content as JSON and checks it against the schema.
func validateToolArgs(content string, schema ToolSchema) (bool, string) {
	content = strings.TrimSpace(content)
	if content == "" {
		if len(schema.RequiredFields) > 0 {
			return true, fmt.Sprintf("tool arguments missing required fields: %v", schema.RequiredFields)
		}
		return false, ""
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(content), &args); err != nil {
		return true, "tool arguments are not valid JSON: " + err.Error()
	}

	// Required fields.
	var missing []string
	for _, f := range schema.RequiredFields {
		if _, ok := args[f]; !ok {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return true, fmt.Sprintf("tool arguments missing required fields: %v", missing)
	}

	// Forbidden fields.
	var forbidden []string
	for _, f := range schema.ForbiddenFields {
		if _, ok := args[f]; ok {
			forbidden = append(forbidden, f)
		}
	}
	if len(forbidden) > 0 {
		return true, fmt.Sprintf("tool arguments contain forbidden fields: %v", forbidden)
	}

	// Max field count.
	if schema.MaxFieldCount > 0 && len(args) > schema.MaxFieldCount {
		return true, fmt.Sprintf("tool arguments have %d fields, maximum is %d", len(args), schema.MaxFieldCount)
	}

	return false, ""
}

func init() {
	guard.Register("tool_misuse_guard", func(cfg map[string]any) (guard.Guard, error) {
		return NewToolMisuseGuard(), nil
	})
}
