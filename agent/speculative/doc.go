// Package speculative provides speculative execution for agents.
//
// Speculative execution runs a fast, cheap predictor in parallel with the
// ground-truth (slower, more expensive) agent. If the prediction matches the
// ground truth, the result is returned early with a speedup. If the prediction
// misses, the ground-truth result is used with minimal overhead.
//
// This pattern is inspired by speculative decoding in LLMs: a small draft
// model proposes tokens that a large verifier model accepts or rejects.
// Here, the same idea is applied at the agent level.
//
// # Architecture
//
// The [SpeculativeExecutor] implements [agent.Agent] and orchestrates:
//
//   - A [Predictor] that produces a fast prediction with a confidence score.
//   - A [Validator] that checks whether the prediction matches the ground truth.
//   - A ground-truth [agent.Agent] that produces the authoritative result.
//   - Thread-safe [Metrics] tracking hits, misses, speedup, and wasted tokens.
//
// # Usage
//
//	fast := speculative.NewLightModelPredictor(cheapModel)
//	validator := speculative.NewExactValidator()
//
//	executor := speculative.NewSpeculativeExecutor(
//	    groundTruthAgent,
//	    speculative.WithPredictor(fast),
//	    speculative.WithValidator(validator),
//	    speculative.WithConfidenceThreshold(0.7),
//	)
//
//	result, err := executor.Invoke(ctx, "What is 2+2?")
//
// # Streaming
//
// Stream pre-classifies via predictor confidence. If confidence exceeds the
// threshold, the fast agent's stream is used. Otherwise, the ground-truth
// agent's stream is used directly.
//
// # Hooks
//
// [Hooks] provide callbacks for prediction, validation, misprediction, and
// cancellation events.
package speculative
