// Package cognitive implements a System 1 / System 2 dual-process cognitive
// architecture for agents. Inspired by Kahneman's dual-process theory, SOFAI
// (Software Architecture for AI), and DeepMind's Talker-Reasoner pattern,
// this package provides a DualProcessAgent that routes between a fast,
// heuristic System 1 agent and a deliberative System 2 agent.
//
// The routing decision is driven by a pluggable ComplexityScorer that
// classifies input complexity. Two built-in scorers are provided:
//
//   - HeuristicScorer: zero-cost keyword and token-count analysis
//   - LLMScorer: LLM-based classification for higher accuracy
//
// For synchronous (Invoke) calls, the agent uses a cascading strategy:
// System 1 runs first, and if the scorer rates the output as insufficient,
// it escalates to System 2. For streaming (Stream) calls, the agent
// pre-classifies the input and routes directly to avoid buffering.
//
// All scorers follow the registry pattern (RegisterScorer / NewScorer /
// ListScorers) for extensibility. The package also tracks RoutingMetrics
// for cost-savings analysis.
package cognitive
