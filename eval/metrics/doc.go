// Package metrics provides built-in evaluation metrics for the Beluga AI
// eval framework. Each metric implements the eval.Metric interface, returning
// a score in [0.0, 1.0] for a given EvalSample.
//
// # LLM-as-Judge Metrics
//
// These metrics use an LLM to evaluate AI-generated output quality:
//
//   - Faithfulness evaluates whether an answer is grounded in the provided
//     context documents. Requires an llm.ChatModel as judge.
//   - Relevance evaluates whether an answer adequately addresses the input
//     question. Requires an llm.ChatModel as judge.
//   - Hallucination detects fabricated facts by comparing answers against
//     context documents. Requires an llm.ChatModel as judge.
//
// # Keyword-Based Metrics
//
//   - Toxicity is a keyword heuristic (not a classifier). Keywords match
//     case-insensitively on whole-word boundaries. Returns 1.0 when no keyword
//     is found; any match scores strictly below the configurable threshold
//     (default 0.5), decreasing toward 0.0 as more distinct keywords match.
//     Use Toxicity.IsToxic to classify a score.
//
// # Metadata-Based Metrics
//
//   - Latency reads Metadata["latency_ms"] and returns a normalized score
//     where 1.0 is instantaneous and 0.0 is at or above a configurable
//     maximum threshold (default 10 seconds).
//   - Cost reads Metadata["input_tokens"], Metadata["output_tokens"], and
//     Metadata["model"] to calculate the dollar cost based on configurable
//     per-model pricing. Returns the raw dollar amount rather than a
//     normalized score.
//
// # Usage
//
//	// LLM-as-judge metric
//	faith := metrics.NewFaithfulness(judgeModel)
//	score, err := faith.Score(ctx, sample)
//
//	// Keyword-based metric
//	tox := metrics.NewToxicity()
//	score, err = tox.Score(ctx, sample)
//
//	// Metadata-based metric
//	lat := metrics.NewLatency(metrics.WithMaxLatencyMs(5000))
//	score, err = lat.Score(ctx, sample)
package metrics
