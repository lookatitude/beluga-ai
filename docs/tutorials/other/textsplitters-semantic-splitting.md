# Semantic Splitting for better Embeddings

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use semantic splitting to create chunks based on the meaning of the text. You'll learn how to use an embedder to find transition points where topics change, ensuring each chunk is semantically coherent and focused.

## Learning Objectives
- ✅ Understand Semantic vs. Recursive splitting
- ✅ Use an Embedder to find transition points
- ✅ Balance chunk size vs. semantic coherence
- ✅ Implement a Semantic Splitter in Beluga AI

## Introduction
Welcome, colleague! Even structural splitting can be arbitrary if a single paragraph covers two distinct topics. Let's look at how to use embeddings to "see" where one topic ends and another begins, creating chunks that are truly optimized for semantic retrieval.

## Prerequisites

- [Simple RAG](../../getting-started/02-simple-rag.md)
- OpenAI or Ollama Embedder configured

## The Problem: Arbitrary Breaks

Even `RecursiveCharacterSplitter` is arbitrary. It doesn't know when a topic actually changes. 
**Semantic Splitting** looks for "breaks" where the similarity between consecutive sentences drops significantly.

## Step 1: The Semantic Logic

1. Split text into sentences.
2. Embed every sentence.
3. Calculate cosine similarity between sentence `i` and `i+1`.
4. If similarity < threshold, start a new chunk.

## Step 2: Implementation with Beluga AI
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    // 1. Needs an embedder to "understand" the text
    embedder := setupEmbedder() 

    config := &textsplitters.SemanticConfig{
        Embedder:           embedder,
        BreakpointThreshold: 0.85, // Adjust based on model
        BufferSize:         1,    // Look-ahead/behind sentences
    }
    
    splitter, _ := textsplitters.NewSemanticSplitter(config)
}
```

## Step 3: Tuning the Threshold

- **High Threshold (0.95)**: Many small, highly specific chunks.
- **Low Threshold (0.70)**: Fewer, larger chunks (more context, but noisier).

## Step 4: Practical Example
```go
    text := `The sun is a star. It is at the center of the Solar System. 
             Cats are feline mammals. They are often kept as pets.`
             
    // Semantic splitter should break after "Solar System." 
    // because the topic changes from Astronomy to Biology.
    chunks, _ := splitter.SplitText(text)
```

## Verification

1. Run the semantic splitter on a document that changes topics clearly.
2. Verify that chunks don't contain mixed topics.
3. Measure retrieval accuracy (Hit Rate) vs. standard splitting.

## Next Steps

- **[Markdown-aware Chunking](./textsplitters-markdown-chunking.md)** - Combine structural and semantic cues.
- **[Fine-tuning Embedding Strategies](../providers/embeddings-finetuning-strategies.md)** - Optimize the underlying embeddings.
