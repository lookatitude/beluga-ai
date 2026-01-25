# Multi-query Retrieval Chains

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll improve RAG performance by generating multiple search queries from a single user question. We'll build a chain that uses an LLM to expand a user's query into several variations to capture more relevant context.

## Learning Objectives
- ✅ Understand the "vocabulary mismatch" problem
- ✅ Use LLMs to generate query variations
- ✅ Deduplicate and fuse results

## Introduction
Welcome, colleague! Users don't always use the same terminology as your documentation. If they ask "How do I add a helper?" but your docs use "Middleware Implementation," vector search might fail. Let's use an LLM to bridge that gap by generating better search queries.

## Prerequisites

- [Simple RAG](../../getting-started/02-simple-rag.md)

## The Problem

User asks: "How do I add a helper?"
Docs say: "Middleware Implementation."
Vector search might miss this connection.

## Step 1: Query Generation Chain

Create a chain specifically to expand the query.
```go
const template = `You are a helpful assistant.
Generate 3 different search queries for the user question.
Focus on different perspectives and synonyms.
Question: {{.Input}}
Output (newline separated):`

generator := chatmodels.NewChatModel(llm)
// ... bind prompt ...

## Step 2: Execution
func generateQueries(question string) []string {
    response, _ := generator.Generate(ctx, question)
    // Split by newline
    return strings.Split(response, "\n")
}
```

Example Output:
1. "How to add a helper"
2. "Implementing middleware patterns"
3. "Extending functionality with wrappers"

## Step 3: Retrieval and Fusion
```go
func retrieveAll(queries []string) []schema.Document {
    var allDocs []schema.Document
    for _, q := range queries {
        docs, _ := retriever.GetRelevantDocuments(ctx, q)
        allDocs = append(allDocs, docs...)
    }
    
    return deduplicate(allDocs)
}

func deduplicate(docs []schema.Document) []schema.Document {
    // Dedupe based on content or ID
    seen := make(map[string]bool)
    var unique []schema.Document
    
    for _, d := range docs {
        if !seen[d.Content] {
            seen[d.Content] = true
            unique = append(unique, d)
        }
    }
    return unique
}
```

## Step 4: Putting it together

Wrap this entire logic into a custom `Runnable` or `Retriever`.
```go
type MultiQueryRetriever struct {
    LLM      llmsiface.ChatModel
    Retriever core.Retriever
}

func (m *MultiQueryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    queries := m.generateQueries(query)
    return m.retrieveAll(queries), nil
}
```

## Verification

1. Ask a vague question.
2. Log the generated queries.
3. Verify that at least one query matches the terminology in your documents.

## Next Steps

- **[Hybrid Search](./retrievers-hybrid-search.md)** - Combine with keyword search
- **[RAG Multimodal](../../guides/rag-multimodal.md)** - Retrieve images too
