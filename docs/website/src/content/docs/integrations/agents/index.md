---
title: Agents & Tools
description: Integration guides for agent tools, MCP integration, API bridges, and testing patterns.
sidebar:
  order: 0
---

Agents are only as capable as the tools they can use. This section covers creating custom tools, exposing them over the Model Context Protocol (MCP) for IDE and bot consumption, bridging external APIs like OpenAI Assistants into Beluga's unified interface, and testing agent behavior deterministically without live API calls.

| Guide | Description |
|-------|-------------|
| [Custom Tools and Tool Registry](./agents-tools-registry/) | Build custom tools and manage them with the tool registry |
| [Agents and MCP Integration](./agents-mcp-integration/) | Expose agent tools over MCP for IDE clients and bots |
| [OpenAI Assistants Bridge](./openai-assistants-bridge/) | Bridge OpenAI's Assistants API with Beluga AI's ChatModel |
| [Mock ChatModel for UI Testing](./mock-ui-testing/) | Create mock ChatModel implementations for deterministic testing |
