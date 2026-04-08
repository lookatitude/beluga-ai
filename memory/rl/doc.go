// Package rl provides RL-optimized memory operations for Beluga AI agents.
//
// This package implements the Memory-R1 approach: a reinforcement learning
// policy that decides which memory action (Add, Update, Delete, Noop) to
// take when an agent encounters new information. The policy is trained to
// maximize downstream task performance using GRPO (Group Relative Policy
// Optimization).
//
// # Architecture
//
// The rl package wraps any [memory.Memory] implementation with a policy-based
// decision layer:
//
//  1. When Save is called, [FeatureExtractor] computes [PolicyFeatures] from
//     the current memory state and the incoming message.
//  2. A [MemoryPolicy] decides which [MemoryAction] to take (Add, Update,
//     Delete, or Noop) along with a confidence score.
//  3. [PolicyMemory] routes the action to the underlying memory accordingly.
//
// # Built-in Policies
//
// [HeuristicPolicy] provides a rule-based baseline: add if the content is
// novel, update if a similar entry exists, delete if utility is low, and
// noop if the content is redundant. This works without any trained model
// and is suitable as a starting point or fallback.
//
// For trained policies, implement the [MemoryPolicy] interface with an ONNX
// or gRPC backend (see memory/rl/providers/ for provider packages).
//
// # Training Support
//
// [TrajectoryCollector] records decision episodes for offline training.
// Each episode contains a sequence of [Step] records (features + action +
// timestamp) plus a final outcome that is compared to ground truth by a
// [RewardFunc] to produce per-step rewards.
//
// # Usage
//
//	inner := memory.NewComposite(memory.WithCore(core), memory.WithRecall(recall))
//	policy := rl.NewHeuristicPolicy()
//	mem := rl.New(inner, policy)
//	err := mem.Save(ctx, input, output)  // policy decides action
//	msgs, err := mem.Load(ctx, "query")  // passthrough to inner
package rl
