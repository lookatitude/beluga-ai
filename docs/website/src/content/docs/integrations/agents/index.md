---
title: Agent & Tool Integrations
description: "Build custom tools, connect MCP servers, bridge OpenAI Assistants, and test agents deterministically with Beluga AI in Go."
sidebar:
  order: 0
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI agents, tool registry, MCP integration, OpenAI Assistants bridge, agent testing, Go AI tools, Model Context Protocol"
---

Agents are only as capable as the tools they can use. This section covers creating custom tools, exposing them over the Model Context Protocol (MCP) for IDE and bot consumption, bridging external APIs like OpenAI Assistants into Beluga's unified interface, and testing agent behavior deterministically without live API calls.

| Guide | Description |
|-------|-------------|
| [Custom Tools and Tool Registry](./agents-tools-registry/) | Build custom tools and manage them with the tool registry |
| [Agents and MCP Integration](./agents-mcp-integration/) | Expose agent tools over MCP for IDE clients and bots |
| [OpenAI Assistants Bridge](./openai-assistants-bridge/) | Bridge OpenAI's Assistants API with Beluga AI's ChatModel |
| [Mock ChatModel for UI Testing](./mock-ui-testing/) | Create mock ChatModel implementations for deterministic testing |
