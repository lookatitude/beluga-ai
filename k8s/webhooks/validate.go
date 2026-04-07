package webhooks

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/k8s/operator"
)

// validPlanners is the set of recognised planner values for an Agent CRD.
var validPlanners = map[string]bool{
	"react":            true,
	"openai-functions": true,
	"plan-and-execute": true,
}

// validPatterns is the set of recognised pattern values for a Team CRD.
var validPatterns = map[string]bool{
	"sequential": true,
	"parallel":   true,
	"supervisor": true,
}

// ValidationResult contains the admission webhook decision.
type ValidationResult struct {
	// Allowed is true when the resource passes all validation checks.
	Allowed bool

	// Reason is a human-readable explanation when Allowed is false.
	// It is empty when Allowed is true.
	Reason string
}

// allowed is the singleton result for a successful validation.
var allowed = ValidationResult{Allowed: true}

// deny constructs a ValidationResult that rejects the resource.
func deny(reason string) ValidationResult {
	return ValidationResult{Allowed: false, Reason: reason}
}

// ValidateAgent checks an AgentResource for correctness and returns an
// admission decision. The following invariants are enforced:
//   - Spec.ModelRef must be non-empty.
//   - Spec.Planner must be one of "react", "openai-functions", "plan-and-execute".
//   - Spec.MaxIterations must be > 0.
//   - Spec.Persona.Role must be non-empty.
func ValidateAgent(agent *operator.AgentResource) ValidationResult {
	if agent == nil {
		return deny("agent must not be nil")
	}

	if strings.TrimSpace(agent.Spec.ModelRef) == "" {
		return deny("spec.modelRef is required")
	}

	if !validPlanners[agent.Spec.Planner] {
		return deny(fmt.Sprintf(
			"spec.planner %q is not valid; must be one of %s",
			agent.Spec.Planner, plannerList(),
		))
	}

	if agent.Spec.MaxIterations <= 0 {
		return deny(fmt.Sprintf(
			"spec.maxIterations must be > 0, got %d",
			agent.Spec.MaxIterations,
		))
	}

	if strings.TrimSpace(agent.Spec.Persona.Role) == "" {
		return deny("spec.persona.role is required")
	}

	return allowed
}

// ValidateTeam checks a TeamResource for correctness and returns an admission
// decision. The following invariants are enforced:
//   - Spec.Pattern must be one of "sequential", "parallel", "supervisor".
//   - Spec.Members must be non-empty.
//   - Spec.Members must not contain duplicate names.
func ValidateTeam(team *operator.TeamResource) ValidationResult {
	if team == nil {
		return deny("team must not be nil")
	}

	if !validPatterns[team.Spec.Pattern] {
		return deny(fmt.Sprintf(
			"spec.pattern %q is not valid; must be one of %s",
			team.Spec.Pattern, patternList(),
		))
	}

	if len(team.Spec.Members) == 0 {
		return deny("spec.members (agentRefs) must not be empty")
	}

	seen := make(map[string]bool, len(team.Spec.Members))
	for _, m := range team.Spec.Members {
		if seen[m.Name] {
			return deny(fmt.Sprintf("spec.members contains duplicate name %q", m.Name))
		}
		seen[m.Name] = true
	}

	return allowed
}

// plannerList returns the sorted list of valid planner names for error messages.
func plannerList() string {
	names := make([]string, 0, len(validPlanners))
	for k := range validPlanners {
		names = append(names, fmt.Sprintf("%q", k))
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// patternList returns the sorted list of valid pattern names for error messages.
func patternList() string {
	names := make([]string, 0, len(validPatterns))
	for k := range validPatterns {
		names = append(names, fmt.Sprintf("%q", k))
	}
	return "[" + strings.Join(names, ", ") + "]"
}
