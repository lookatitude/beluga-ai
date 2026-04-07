// Package webhooks provides Kubernetes admission webhook handlers for the
// Beluga AI operator.
//
// This package implements validation and mutation webhooks for [operator.AgentResource]
// and [operator.TeamResource] custom resources. It is intentionally free of any
// Kubernetes library dependencies — all logic operates on plain Go structs from
// the operator package and returns simple [ValidationResult] values.
//
// # Validation Webhooks
//
// [ValidateAgent] checks an [operator.AgentResource] for correctness:
//   - ModelRef must be non-empty.
//   - Planner must be a recognised value ("react", "openai-functions", "plan-and-execute").
//   - MaxIterations must be > 0.
//   - Persona.Role must be non-empty.
//
// [ValidateTeam] checks a [operator.TeamResource] for correctness:
//   - Pattern must be a recognised value ("sequential", "parallel", "supervisor").
//   - Members (agentRefs) must be non-empty.
//   - No duplicate member names are allowed.
//
// # Mutation Webhooks
//
// [MutateAgent] applies sensible defaults to an [operator.AgentResource]:
//   - Sets MaxIterations to 10 when it is zero.
//   - Adds the standard label beluga.ai/component=agent.
//
// [MutateTeam] applies sensible defaults to a [operator.TeamResource]:
//   - Adds the standard label beluga.ai/component=team.
//
// # Usage
//
//	// Validate before persisting.
//	result := webhooks.ValidateAgent(agentResource)
//	if !result.Allowed {
//	    return errors.New(result.Reason)
//	}
//
//	// Apply defaults before persisting.
//	agentResource = webhooks.MutateAgent(agentResource)
package webhooks
