// Package metacognitive provides self-model improvement capabilities for agents.
//
// It implements cross-session learning inspired by ExpeL (Experience Learning)
// and ERL (Experience Reinforcement Learning) research. Agents accumulate
// heuristics from their successes and failures, persist them across sessions,
// and retrieve relevant ones to guide future behavior.
//
// Key components:
//
//   - SelfModel: persistent typed self-knowledge (heuristics + capability scores)
//   - SelfModelStore: persistence interface with in-memory implementation
//   - Monitor: hooks-based signal collection from agent execution
//   - HeuristicExtractor: extracts learnings from execution signals
//   - MetacognitivePlugin: runtime.Plugin that injects and persists learnings
//
// Usage:
//
//	store := metacognitive.NewInMemoryStore()
//	plugin := metacognitive.NewPlugin(store)
//	runner := runtime.NewRunner(myAgent, runtime.WithPlugins(plugin))
package metacognitive
