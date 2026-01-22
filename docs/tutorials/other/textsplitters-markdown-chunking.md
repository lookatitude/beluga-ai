# Markdown-aware Chunking

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll split Markdown documents into chunks based on their logical structure like headers, lists, and code blocks. You'll learn how to maintain context for RAG by ensuring headers stay with their content and code blocks aren't cut in half.

## Learning Objectives
- ✅ Understand why simple character splitting fails for Markdown
- ✅ Use the `MarkdownSplitter`
- ✅ Configure split levels (H1, H2, H3)
- ✅ Preserving code block integrity

## Introduction
Welcome, colleague! Splitting text by character count is a recipe for broken context. If a header is separated from its paragraph, your agent loses the "what" and "why." Let's build a structural splitter that understands Markdown and keeps our knowledge chunks coherent.

## Prerequisites

- Go 1.24+
- `pkg/textsplitters` package

## The Problem: Broken Context

If you split a Markdown file every 1000 characters:
- You might split in the middle of a code block.
- A header might be separated from its content.
- A list item might lose its parent bullet.

## Step 1: Initialize Markdown Splitter
```go
package main

import (
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/markdown"
)

func main() {
    // Configure to split primarily by H2 headers
    config := &markdown.Config{
        HeadersToSplitOn: []string{"#", "##", "###"},
        IncludeHeaders:   true, // Keep the header text in the chunk
    }
    
    splitter, _ := markdown.NewSplitter(config)
}
```

## Step 2: Splitting a Document
```go
    text := `
```
# My Project
## Installation
Run go get...
## Usage
Import the package...
    `
    
```go
    chunks, _ := splitter.SplitText(text)
    
    for i, chunk := range chunks {
        fmt.Printf("Chunk %d:\n%s\n---\n", i, chunk)
    }
```

## Step 3: Handling Code Blocks

The Markdown splitter ensures that fenced code blocks (<code> ```go ... ``` </code>) are not split internally unless they exceed the maximum chunk size.
```
    config.MaxChunkSize = 2000

## Step 4: Combining with Recursive Character Splitter

Use the Markdown splitter for structural breaks, then the recursive splitter for long sections.
    // This is often handled automatically by the high-level 
    // textsplitters.NewSplitter("markdown") factory.
```

## Verification

1. Take a complex Markdown file with multiple headers and code blocks.
2. Split it.
3. Verify that each chunk starts with a header and no code blocks are "cut in half".

## Next Steps

- **[Semantic Splitting](./textsplitters-semantic-splitting.md)** - Splitting by meaning, not just syntax.
- **[Directory & PDF Scraper](./docloaders-directory-pdf-scraper.md)** - Load files to split.
