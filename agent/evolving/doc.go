// Package evolving provides experience-learning support for agents. An
// EvolvingAgent wraps an agent, records its interactions, and uses an
// ExperienceDistiller and a MetaOptimizer to produce improvement suggestions
// from accumulated experience.
//
// EvolvingAgent is observation only: it does not modify the wrapped agent.
// Read the suggestions with EvolvingAgent.Suggestions and apply them (for
// example by updating the persona or prompt) in application code.
package evolving
