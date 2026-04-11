// Package redteam provides automated red teaming for AI agents. It generates
// adversarial prompts across multiple attack categories (prompt injection,
// jailbreak, obfuscation, tool misuse, data exfiltration, role play, and
// multi-turn), runs them against a target agent, and scores the agent's
// defenses.
//
// The package follows the registry pattern: attack patterns are registered
// via RegisterPattern and discovered via ListPatterns. Built-in patterns
// cover prompt injection, jailbreak, and obfuscation (Base64, ROT13,
// leetspeak). Custom patterns can be registered to extend the attack surface.
//
// An AttackGenerator can use an LLM (llm.ChatModel) to dynamically create
// novel adversarial prompts beyond the static built-in patterns.
//
// The RedTeamRunner orchestrates the full red team exercise: it collects
// attack prompts from patterns and the generator, runs them against the
// target agent.Agent, and produces a RedTeamReport with per-category scores
// and an overall defense score.
//
// Usage:
//
//	runner := redteam.NewRunner(
//	    redteam.WithTarget(myAgent),
//	    redteam.WithPatterns("prompt_injection", "jailbreak", "obfuscation"),
//	    redteam.WithParallel(4),
//	    redteam.WithTimeout(30 * time.Second),
//	)
//	report, err := runner.Run(ctx)
package redteam
