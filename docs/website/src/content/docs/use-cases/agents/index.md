---
title: AI Agent Use Cases
description: "Build intelligent agents with tool use, multi-agent handoffs, memory, and autonomous reasoning. Production-ready Go examples."
sidebar:
  order: 0
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AI agents, multi-agent systems, conversational AI, autonomous agents, Go AI agents, Beluga AI, agent handoffs, ReAct agents"
---

Build intelligent agents with tool use, multi-agent handoffs, memory, and autonomous reasoning using Beluga AI's agent runtime. These use cases demonstrate core agent patterns: ReAct for autonomous reasoning-and-action loops, handoffs-as-tools for multi-agent coordination, MemGPT 3-tier memory for persistent context, and the guard pipeline for safety. Each agent uses the registry pattern (`agent.New()`) and functional options for configuration.

| Use Case | Description |
|----------|-------------|
| [Multi-Agent Customer Support](./multi-agent-support/) | Build intelligent support with specialized agents, handoffs, and human escalation. |
| [Autonomous Customer Support](./autonomous-support/) | Build self-service support with ReAct agents, tool integration, and intelligent escalation. |
| [Conversational AI Assistant](./conversational-ai/) | Build a personalized conversational AI with persistent 3-tier memory. |
| [Automated Code Review](./code-review-system/) | Build an AI-powered code review agent with git integration and structured feedback. |
| [Few-Shot SQL Generation](./few-shot-sql/) | Few-shot learning for natural language to SQL query generation. |
| [Context-aware IDE Extension](./memory-ide-extension/) | Build an IDE extension with project-specific memory that learns from code patterns. |
| [Dynamic Tool Injection](./dynamic-tool-injection/) | Runtime tool injection for context-aware agent tool selection. |
