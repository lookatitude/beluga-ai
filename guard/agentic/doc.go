// Package agentic provides guards aligned with the OWASP Top 10 for Agentic
// Applications. It extends the guard package with specialised validators that
// address risks unique to multi-agent and tool-using AI systems.
//
// The ten OWASP agentic risks covered are:
//
//  1. Prompt Injection (AG01) -- handled by guard.PromptInjectionDetector
//  2. Insecure Output Handling (AG02) -- handled by guard.ContentFilter
//  3. Tool Misuse (AG03) -- ToolMisuseGuard
//  4. Privilege Escalation (AG04) -- PrivilegeEscalationGuard
//  5. Memory Poisoning (AG05) -- future: MemoryPoisoningGuard
//  6. Data Exfiltration (AG06) -- DataExfiltrationGuard
//  7. Supply Chain (AG07) -- future: SupplyChainGuard
//  8. Cascading Failure (AG08) -- CascadeGuard
//  9. Insufficient Logging (AG09) -- addressed via o11y integration
//  10. Excessive Autonomy (AG10) -- CascadeGuard depth/iteration limits
//
// Usage:
//
//	pipeline := agentic.NewAgenticPipeline(
//	    agentic.WithToolMisuseGuard(),
//	    agentic.WithEscalationGuard(),
//	    agentic.WithExfiltrationGuard(),
//	    agentic.WithCascadeGuard(),
//	)
//	result, err := pipeline.Validate(ctx, input)
package agentic
