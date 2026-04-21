// Package degradation provides graceful degradation of agent autonomy in
// response to security events. It monitors anomaly signals from the guard
// pipeline, computes a severity score, and applies runtime restrictions to
// agent capabilities based on configurable policies.
//
// # Autonomy Levels
//
// The package defines four autonomy levels that progressively restrict agent
// capabilities:
//
//   - Full: All tools and capabilities are available. Normal operation.
//   - Restricted: Only explicitly allowlisted tools may be executed.
//   - ReadOnly: No tool calls or write operations are permitted.
//   - Sequestered: The agent is fully isolated; all actions are logged but
//     not executed.
//
// # Security Monitor
//
// SecurityMonitor tracks anomaly signals from guard events within a sliding
// time window. Each recorded SecurityEvent contributes to an aggregate
// severity score that decays as events age out of the window.
//
// # Degradation Policy
//
// PolicyEvaluator maps severity scores to autonomy levels. The built-in
// ThresholdPolicy uses configurable severity thresholds for each level
// transition.
//
// # Runtime Degrader
//
// RuntimeDegrader is an agent.Middleware that intercepts agent invocations
// and enforces the current autonomy level. It queries the SecurityMonitor
// for the current severity, evaluates the PolicyEvaluator, and applies
// the appropriate restrictions before delegating to the wrapped agent.
//
// # Usage
//
//	monitor := degradation.NewSecurityMonitor(
//	    degradation.WithWindowSize(5 * time.Minute),
//	)
//	policy := degradation.NewThresholdPolicy()
//	degrader := degradation.NewRuntimeDegrader(monitor, policy,
//	    degradation.WithToolAllowlist("search", "read_file"),
//	)
//	wrapped := agent.ApplyMiddleware(myAgent, degrader.Middleware())
package degradation
