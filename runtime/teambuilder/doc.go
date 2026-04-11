// Package teambuilder provides role-based dynamic team formation for
// multi-agent systems. It enables automatic selection of agents from a pool
// based on task requirements, using pluggable selection strategies.
//
// The core workflow is:
//  1. Register agents with capabilities into an AgentPool.
//  2. Configure a TeamBuilder with a Selector strategy and constraints.
//  3. Call Build(ctx, task) to dynamically form a runtime.Team.
//
// Built-in selectors include KeywordSelector (fast keyword overlap) and
// LLMSelector (LLM-based structured selection for higher accuracy).
// Custom selectors implement the Selector interface and are registered
// via RegisterSelector for discovery.
//
// Example:
//
//	pool := teambuilder.NewAgentPool()
//	pool.Register(codeAgent, "golang", "testing", "refactoring")
//	pool.Register(docAgent, "documentation", "markdown", "tutorials")
//
//	builder := teambuilder.NewTeamBuilder(pool,
//	    teambuilder.WithSelector(teambuilder.NewKeywordSelector()),
//	    teambuilder.WithMaxAgents(3),
//	)
//
//	team, err := builder.Build(ctx, "write unit tests for the auth package")
package teambuilder
