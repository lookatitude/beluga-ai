# {Title}

<!--
Template Guidelines for Cookbook Recipes:
- Be concise but thorough
- Focus on solving ONE specific problem
- Assume reader understands Go basics but needs Beluga AI guidance
- Write like a helpful colleague sharing a quick tip
- Keep code examples focused (50-100 lines max)
-->

## Problem

<!--
One clear sentence describing the problem.
Be specific: "You need to handle rate limit errors from the OpenAI API" 
not "Error handling".
-->

You need to {specific problem description}.

## Solution

<!--
Brief explanation (2-3 sentences) of the approach.
Explain the pattern, not just the code.
-->

The solution is to {approach overview}. This works because {brief explanation of why this pattern is effective}.

## Code Example

<!--
Focused, runnable code snippet (50-100 lines max).
Include comments explaining key decisions.
Show production-ready patterns (error handling, OTEL).
-->

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/{package}"
)

func main() {
    ctx := context.Background()
    
    // {Comment explaining key decision}
    // ...
    
    if err != nil {
        // {Comment explaining error handling approach}
        log.Fatalf("failed: %v", err)
    }
    
    fmt.Printf("Result: %v\n", result)
}
```

## Explanation

<!--
Walk through the code step-by-step.
Explain why each part matters.
Use a teacher's voice: "Notice how we..." rather than "The code does..."
-->

Let's break down what's happening:

1. **{First key point}** - Notice how we {explanation}. This is important because {reason}.

2. **{Second key point}** - We {action} here to {benefit}. Without this, you might encounter {potential issue}.

3. **{Third key point}** - The {element} ensures {outcome}.

**Key insight:** {Most important takeaway from this solution}

## Testing

<!--
Show how to verify the solution works.
Include a simple test or verification steps.
-->

Here's how to test this solution:

```go
func TestSolution(t *testing.T) {
    // Arrange
    ctx := context.Background()
    // ... setup
    
    // Act
    result, err := // ... execute
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

Or verify manually:

```bash
# Run the example
go run main.go

# Expected output:
# {expected output}
```

## Variations

<!--
Optional: Show common variations of this pattern
-->

### {Variation 1: Different scenario}

If you need to {alternative scenario}, modify the approach:

```go
// {Brief code showing variation}
```

## Related Recipes

<!--
Link to related cookbook entries with context about when to use each.
Keep descriptions brief but helpful.
-->

- **[{Related Recipe 1}](./related-recipe.md)** - Use this when you need to {specific scenario}
- **[{Related Recipe 2}](./another-recipe.md)** - Helpful if you're also working with {feature}
- **[{Related Guide}](../guides/related-guide.md)** - For a deeper understanding of {topic}
