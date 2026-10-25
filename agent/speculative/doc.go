// Package speculative provides speculative execution for agents.
//
// Speculative execution runs a fast, cheap predictor in parallel with the
// ground-truth (slower, more expensive) agent.
//
// By default Invoke waits for both and returns the prediction only after the
// validator has accepted it against the ground truth, so the answer is always
// validated but Invoke is never faster than the ground truth alone (and costs
// the predictor call on top). With [WithEarlyReturn], a confident prediction
// is returned as soon as it is ready and validated in the background;
// mispredictions are reported through [Hooks].OnMisprediction so the caller
// can correct them. Only the early-return mode actually reduces latency.
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
