# Model Switching & Fallbacks

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement intelligent model switching strategies: cascading fallbacks for reliability and routing for cost optimization.

## Learning Objectives
- ✅ Implement retry-with-fallback logic
- ✅ Create a cost-optimizing router
- ✅ Handle rate limits dynamically

## Introduction
Welcome, colleague! Relying on a single LLM provider is a risky move for production apps. Let's look at how to build resilience and cost-efficiency by switching between models dynamically based on failure or complexity.

## Prerequisites

- [Multi-provider Chat Integration](./chatmodels-multi-provider.md)

## Pattern 1: Reliability Fallback

If the primary model fails (5xx error, rate limit), try the secondary.
```go
func GenerateWithFallback(ctx context.Context, primary, secondary chatmodels.ChatModel, input string) (string, error) {
    // Try primary
    res, err := primary.Generate(ctx, input)
    if err == nil {
        return res, nil
    }

    

    fmt.Printf("Primary failed: %v. Switching to secondary...\n", err)
    
    // Try secondary
    res, err = secondary.Generate(ctx, input)
    if err != nil {
        return "", fmt.Errorf("both models failed: %w", err)
    }
    
    return res, nil
}
```

## Pattern 2: Cost Router (Cascade)

Start with the cheapest model. If it refuses (e.g., "I can't do that") or has low confidence (if supported), try the stronger model. Since confidence is hard to measure, a common pattern is **Complexity Classification**.
```go
func Router(input string) chatmodels.ChatModel {
    // Use a tiny, fast model/heuristic to classify complexity
    complexity := classifier.Predict(input) 

    
    if complexity == "HARD" \{
        return gpt4
    } else {
        return gpt35
    }
}
```

## Pattern 3: The "Retry with Stronger Model"

If a tool call fails or code doesn't compile, switch to a smarter model for the fix.
res, err := gpt35.Generate(ctx, input)
```
if isCodeBug(res) {
    // Retry with GPT-4
    res, err = gpt4.Generate(ctx, input + "\nPrevious attempt failed.")
}

## Verification

1. Simulate a rate limit error on the primary model (mock it).
2. Verify the fallback triggers.
3. Classify "Write a recursive Fibonacci function" (Easy) vs "Design a microservices architecture for a bank" (Hard).

## Next Steps

- **[Orchestration Basics](../../getting-started/06-orchestration-basics.md)** - Use chains for this logic
- **[Building a Research Agent](./agents-research-agent.md)** - Apply to agents
