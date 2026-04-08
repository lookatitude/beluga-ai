// Package debate implements multi-agent debate and generator-evaluator
// patterns for the Beluga AI orchestration framework.
//
// # Debate Orchestrator
//
// [DebateOrchestrator] enables multiple agents to engage in structured
// debate. Agents contribute responses each round, and a [ConvergenceDetector]
// determines when the debate has reached a consensus or should stop.
//
// Different debate styles are supported via the [DebateProtocol] interface:
//   - [RoundRobinProtocol]: All agents speak each round with full context.
//   - [AdversarialProtocol]: Agents are assigned pro/con roles.
//   - [JudgedProtocol]: One agent acts as judge evaluating others.
//
// Example:
//
//	d := debate.NewDebateOrchestrator(agents,
//	    debate.WithMaxRounds(5),
//	    debate.WithProtocol(debate.NewRoundRobinProtocol()),
//	    debate.WithConvergenceDetector(debate.NewStabilityDetector(0.9)),
//	)
//	result, err := d.Invoke(ctx, "Should we use microservices?")
//
// # Generator-Evaluator
//
// [GeneratorEvaluator] implements the generate-evaluate-refine loop. A
// generator agent produces responses that are scored by evaluator functions.
// The loop continues until all evaluators approve or max iterations is reached.
//
// Example:
//
//	ge := debate.NewGeneratorEvaluator(generator,
//	    []debate.EvaluatorFunc{qualityCheck, safetyCheck},
//	    debate.WithMaxIterations(3),
//	)
//	result, err := ge.Invoke(ctx, "Write a product description")
//
// Both patterns implement [core.Runnable] and can be composed with the rest
// of the orchestration framework. They also support streaming via iter.Seq2.
package debate
