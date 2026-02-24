---
title: Prompts & Schema Integrations
description: "Load prompt templates, validate schemas, and bridge data between Go and Python in Beluga AI with LangChain Hub and JSON Schema."
sidebar:
  order: 0
head:
  - tag: meta
    attrs:
      name: keywords
      content: "prompt templates, JSON Schema, Beluga AI, LangChain Hub, schema validation, Go Python bridge, prompt management"
---

Prompts are the primary interface between your application and language models. Managing them as hardcoded strings works for prototypes, but production systems need versioning, validation, and reuse. These guides cover loading templates from files and community hubs, validating structured data at application boundaries, and bridging data between Go and Python services.

| Guide | Description |
|-------|-------------|
| [LangChain Hub Prompt Loading](./langchain-hub/) | Load community prompt templates from LangChain Hub |
| [Filesystem Template Store](./filesystem-templates/) | Manage prompt templates as files with caching and version control |
| [JSON Schema Validation](./json-schema-validation/) | Validate message and document structures using JSON Schema |
| [Pydantic / Go Struct Bridge](./go-struct-bridge/) | Bridge data exchange between Python Pydantic models and Go structs |
