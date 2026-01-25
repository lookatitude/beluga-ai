# Cross-Referencing Style Guide

**Purpose**: Define consistent cross-referencing patterns for Beluga AI documentation  
**Audience**: Documentation authors and contributors

## Overview

Effective cross-referencing helps users discover related content and navigate the documentation. This guide establishes patterns for consistent, helpful linking across all documentation types.

## Link Format Standards

### Relative Paths

Always use relative paths for internal links. This ensures links work in both the source files and the rendered website.

```markdown
<!-- Good: Relative path -->
[LLM Providers Guide](../guides/llm-providers.md)

<!-- Bad: Absolute URL -->
[LLM Providers Guide](https://lookatitude.github.io/beluga-ai/docs/guides/llm-providers)

<!-- Bad: Root-relative path -->
[LLM Providers Guide](/docs/guides/llm-providers.md)
```

### Descriptive Link Text

Use descriptive link text that tells users what to expect. Avoid generic text like "click here" or "this link".

```markdown
<!-- Good: Descriptive -->
Learn more about [configuring LLM providers](../guides/llm-providers.md).

<!-- Bad: Generic -->
[Click here](../guides/llm-providers.md) to learn about LLM providers.
```

### Brief Context

Add brief context in the surrounding text when linking to help users decide if the linked content is relevant.

```markdown
<!-- Good: With context -->
For production deployments, see the [observability guide](../guides/observability-tracing.md), 
which covers distributed tracing and metrics collection.

<!-- Acceptable but less helpful -->
See the [observability guide](../guides/observability-tracing.md).
```

## Related Resources Section Format

Every documentation resource should include a "Related Resources" section at the end. Use the following formats:

### For Guides

```markdown
## Related Resources

Now that you understand {topic}, explore these related resources:

- **[{Related Guide Title}](../guides/{guide}.md)** - {One sentence describing what users will learn}
- **[{Example Name}](https://github.com/lookatitude/beluga-ai/blob/main/examples/{path}/README.md)** - {What the example demonstrates}
- **[{Cookbook Recipe}](../cookbook/{recipe}.md)** - {Quick solution for specific task}
- **[{Use Case}](../use-cases/{use-case}.md)** - {Real-world scenario description}
```

### For Cookbook Recipes

```markdown
## Related Recipes

- **[{Recipe Title}](./another-recipe.md)** - Use this when {specific scenario}
- **[{Related Guide}](../guides/{guide}.md)** - Deeper dive into {topic}
```

### For Use Cases

```markdown
## Related Use Cases

If you're working on a similar project:

- **[{Use Case Title}](./related-use-case.md)** - Similar scenario with {different focus}
- **[{Guide}](../guides/{guide}.md)** - Detailed guide for {feature used}
- **[{Example}](https://github.com/lookatitude/beluga-ai/blob/main/examples/{path}/README.md)** - Code example for {feature}
```

### For Example READMEs

```markdown
## Related Examples

- **[{Example Name}](../{path}/README.md)** - {What it demonstrates}

## Learn More

- **[{Guide}](/docs/guides/{guide}.md)** - Comprehensive guide
- **[{Cookbook}](/docs/cookbook/{recipe}.md)** - Quick recipes
```

## Cross-Reference Categories

### When to Link to Each Resource Type

| Link to... | When the user needs... |
|------------|----------------------|
| **Guide** | In-depth understanding, step-by-step tutorial, conceptual background |
| **Cookbook** | Quick solution for a specific task, code snippet to copy-paste |
| **Use Case** | Real-world example, architecture inspiration, business context |
| **Example** | Complete, runnable code, testing patterns, implementation reference |
| **API Reference** | Function signatures, parameter details, return types |
| **Concept** | Foundational understanding, terminology, how things work |

### Linking Patterns by Context

**In a Tutorial Step:**
```markdown
We'll use the `NewLLM` function here. If you need to customize the configuration, 
see the [configuration options guide](../guides/implementing-providers.md).
```

**In an Error Handling Section:**
```markdown
This error often occurs when... For more error handling patterns, 
see our [LLM error handling recipe](../cookbook/llm-error-handling.md).
```

**In a Best Practices Section:**
```markdown
In production, you'll want proper observability. Our 
[batch processing use case](../use-cases/11-batch-processing.md) shows how we 
implemented monitoring for a high-volume system.
```

## Link Maintenance

### Preventing Broken Links

1. **Use consistent file naming**: Follow the patterns in this guide
2. **Update links when moving files**: Search for references before renaming
3. **Run link validation**: Use `npm run build` in the website directory to catch broken links

### When Renaming or Moving Files

1. Search for all references to the old path:
   ```bash
   grep -r "old-filename.md" docs/
   ```

2. Update all references before making the change

3. Consider adding a redirect if the page was linked externally

### Validation Script

Run periodic validation:

```bash
# In website directory
npm run build

# Check for broken links in output
# Docusaurus will warn about broken internal links
```

## Examples

### Complete Related Resources Section (Guide)

```markdown
## Related Resources

Now that you understand streaming LLM calls with tool calling, explore:

- **[Agent Types Guide](../guides/agent-types.md)** - Learn how agents orchestrate 
  multiple tool calls in a reasoning loop
- **[Streaming Example](https://github.com/lookatitude/beluga-ai/blob/main/examples/llms/streaming/README.md)** - Complete 
  implementation with tests and OTEL instrumentation
- **[LLM Error Handling](../cookbook/llm-error-handling.md)** - Handle rate limits 
  and API errors gracefully
- **[Batch Processing Use Case](../use-cases/11-batch-processing.md)** - See streaming 
  in action for high-volume processing
```

### Inline Cross-Reference (Tutorial)

```text
### Step 3: Configure the LLM Client

Now we'll create the LLM client. We're using the OpenAI provider here, but the
same pattern works for [Anthropic](./guides/llm-providers.md#anthropic) and
[Ollama](./guides/llm-providers.md#ollama).

(code block with Go example here)

> 💡 **Tip**: For local development, Ollama is a great option that doesn't
> require API keys.
```

## Checklist for Authors

Before submitting documentation:

- [ ] All internal links use relative paths
- [ ] Link text is descriptive
- [ ] Related Resources section is complete
- [ ] Links point to correct, existing files
- [ ] Context is provided for links where helpful
- [ ] No broken links (verify with `npm run build`)
