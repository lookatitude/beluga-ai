# Three-Level Example Organization

Organize examples by package, learning path, AND use case.

## Structure in Meta-README

```markdown
## Example Categories

### Core Packages
- **[config](config/basic/)** - Configuration management
- **[embeddings](embeddings/basic/)** - Text embedding generation
...

## Learning Path

### Beginner
1. Start with **[agents/basic](agents/basic/)**
2. Try **[rag/simple](rag/simple/)**
3. Explore **[agents/with_tools](agents/with_tools/)**

### Intermediate
1. Learn **[agents/react](agents/react/)**
2. Try **[rag/with_memory](rag/with_memory/)**
...

### Advanced
1. Study **[multi-agent/collaboration](multi-agent/collaboration/)**
...

## Example Selection Guide

### By Use Case

**Building a Chatbot:**
- Start with `agents/basic`
- Add `agents/with_memory` for conversation history
- Use `rag/simple` for knowledge base integration
```

## Why Three Dimensions
- **By Package**: Quick reference for API exploration
- **By Learning Path**: Progressive skill building
- **By Use Case**: Solution-oriented discovery

## Directory Convention
- Each example in its own directory: `{category}/{name}/`
- Required files: `main.go` + `README.md`
- Optional: `*_test.go`, `data/` directory
