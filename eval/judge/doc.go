// Package judge provides LLM-as-Judge evaluation capabilities.
//
// It implements eval.Metric using an LLM to score samples against structured
// rubrics. BatchJudge enables concurrent evaluation with rate limiting, and
// ConsistencyChecker validates scoring reliability through repeated evaluation
// and cross-model agreement analysis.
//
// Key types:
//   - JudgeMetric: Evaluates samples using an LLM judge and a rubric
//   - Rubric: Defines scoring criteria with levels and weights
//   - BatchJudge: Concurrent evaluation with bounded parallelism
//   - ConsistencyChecker: Repeated eval + cross-model agreement
package judge
