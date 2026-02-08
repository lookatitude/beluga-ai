// Package orchestration provides workflow composition patterns for the Beluga AI
// framework including chains, directed graphs, routers, scatter-gather, supervisors,
// and blackboard architectures.
//
// All patterns implement core.Runnable, allowing seamless composition with the
// rest of the framework. Hooks and middleware provide extensibility for logging,
// tracing, and custom cross-cutting concerns.
//
// Usage:
//
//	// Chain steps sequentially
//	pipeline := orchestration.Chain(step1, step2, step3)
//	result, err := pipeline.Invoke(ctx, input)
//
//	// Fan-out to workers, aggregate results
//	sg := orchestration.NewScatterGather(aggregator, worker1, worker2)
//	result, err := sg.Invoke(ctx, input)
//
//	// Route based on classification
//	router := orchestration.NewRouter(classifier).
//	    AddRoute("math", mathAgent).
//	    AddRoute("code", codeAgent)
//	result, err := router.Invoke(ctx, input)
package orchestration
