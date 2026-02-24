---
title: Foundation Guides
description: "Learn the core abstractions powering Beluga AI — the ChatModel interface, agent runtime, tool system, streaming with iter.Seq2, and the registry pattern in Go."
sidebar:
  order: 0
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, foundations, AI agents, LLM, streaming, registry pattern, middleware"
---

The Foundations guides introduce the core building blocks of Beluga AI v2. Each guide builds on the previous one, starting with the agent abstraction that ties everything together, then drilling into the LLM layer that powers reasoning, and finally covering the supporting systems for structured data extraction and prompt management. These guides assume familiarity with Go but no prior experience with AI frameworks. By the end, you will understand Beluga's key design patterns — the registry pattern for pluggable providers, `iter.Seq2` for streaming, functional options for configuration, and middleware for composable behavior — and how they work together to form a production-ready agentic system.

We recommend reading the guides in order:

| Guide | Description |
|-------|-------------|
| [Building Your First Agent](./first-agent/) | Create a complete AI agent with tools, streaming, and reasoning. Introduces the agent runtime, the ReAct loop, and handoffs between specialized agents. |
| [Working with LLMs](./working-with-llms/) | Configure language models through the unified `ChatModel` interface. Covers provider setup, streaming with `iter.Seq2`, middleware composition, hooks, and multi-provider routing. |
| [Structured Output](./structured-output/) | Extract typed Go structs from LLM responses. Covers schema generation, retry strategies, classification patterns, and production validation. |
| [Prompt Engineering](./prompt-engineering/) | Design, template, version, and optimize prompts. Covers the `PromptManager`, `Builder` for cache-optimal ordering, few-shot patterns, and A/B testing. |
