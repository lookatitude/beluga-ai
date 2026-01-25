# Beluga AI Framework - Cookbook

Quick recipes and code snippets for common tasks.

## Table of Contents

1. [RAG Recipes](./rag-recipes.md) - RAG pipeline patterns
2. [Agent Recipes](./agent-recipes.md) - Agent patterns
3. [Tool Recipes](./tool-recipes.md) - Tool integration
4. [Memory Recipes](./memory-recipes.md) - Memory management
5. [Integration Recipes](./integration-recipes.md) - External integrations
6. [Quick Solutions](./quick-solutions.md) - Common problems

## Quick Reference

### Basic LLM Call

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-3.5-turbo"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)
factory := llms.NewFactory()
provider, _ := factory.CreateProvider("openai", config)
response, _ := provider.Generate(ctx, messages)
```

### Simple RAG

```go
// Add documents
store.AddDocuments(ctx, documents, vectorstores.WithEmbedder(embedder))

// Search
docs, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder)

// Generate
response, _ := llm.Generate(ctx, messagesWithContext)
```

### Agent with Tools

```go
tools := []tools.Tool{tools.NewCalculatorTool()}
agent, _ := agents.NewBaseAgent("assistant", llm, tools)
result, _ := agent.Invoke(ctx, input)
```

## Contributing

Add your own recipes! See [Contributing Guide](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md).

---

**Browse Recipes:** Start with [RAG Recipes](./rag-recipes.md) or [Agent Recipes](./agent-recipes.md)

