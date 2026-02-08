package auth

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Effect specifies whether a rule allows or denies access.
type Effect string

const (
	// EffectAllow grants access when the rule matches.
	EffectAllow Effect = "allow"
	// EffectDeny denies access when the rule matches.
	EffectDeny Effect = "deny"
)

// Condition is a predicate evaluated during ABAC authorization. All conditions
// in a rule must return true for the rule to match.
type Condition func(ctx context.Context, subject string, permission Permission, resource string) bool

// Rule defines an ABAC rule with an effect, conditions, and priority. Rules
// with higher priority are evaluated first. The first matching rule determines
// the authorization outcome.
type Rule struct {
	// Name identifies this rule for logging and debugging.
	Name string

	// Effect specifies whether this rule allows or denies access.
	Effect Effect

	// Conditions are predicates that must ALL return true for this rule to
	// match. An empty conditions slice means the rule always matches.
	Conditions []Condition

	// Priority determines evaluation order. Higher values are evaluated first.
	Priority int
}

// ABACPolicy implements attribute-based access control. Rules are evaluated in
// priority order (highest first); the first matching rule determines the
// outcome. If no rule matches, access is denied (default deny).
//
// ABACPolicy is safe for concurrent use.
type ABACPolicy struct {
	name string

	mu    sync.RWMutex
	rules []Rule
}

// NewABACPolicy creates a new ABAC policy with the given name.
func NewABACPolicy(name string) *ABACPolicy {
	return &ABACPolicy{
		name: name,
	}
}

// Name returns the policy name.
func (p *ABACPolicy) Name() string { return p.name }

// AddRule adds a rule to the policy. Returns an error if the rule name is
// empty or a rule with the same name already exists.
func (p *ABACPolicy) AddRule(rule Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("auth/abac: rule name must not be empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, r := range p.rules {
		if r.Name == rule.Name {
			return fmt.Errorf("auth/abac: rule %q already exists", rule.Name)
		}
	}

	p.rules = append(p.rules, rule)
	return nil
}

// Authorize evaluates rules in priority order (highest first). The first rule
// whose conditions all match determines the result. Returns false if no rule
// matches (default deny).
func (p *ABACPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	p.mu.RLock()
	// Copy rules under lock to sort without holding the lock during evaluation.
	sorted := make([]Rule, len(p.rules))
	copy(sorted, p.rules)
	p.mu.RUnlock()

	// Sort by priority descending.
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	for _, rule := range sorted {
		if matchesAll(ctx, rule.Conditions, subject, permission, resource) {
			return rule.Effect == EffectAllow, nil
		}
	}

	// Default deny.
	return false, nil
}

// matchesAll returns true if all conditions evaluate to true, or if there are
// no conditions.
func matchesAll(ctx context.Context, conditions []Condition, subject string, permission Permission, resource string) bool {
	for _, cond := range conditions {
		if !cond(ctx, subject, permission, resource) {
			return false
		}
	}
	return true
}

// Ensure ABACPolicy implements Policy at compile time.
var _ Policy = (*ABACPolicy)(nil)
