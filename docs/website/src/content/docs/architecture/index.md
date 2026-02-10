---
title: Architecture
description: "Design decisions, package layout, and full architecture reference for Beluga AI v2."
---

The architecture documentation covers the design philosophy, package organization, and technical decisions behind Beluga AI v2.

## Documents

| Document | Description |
|----------|-------------|
| [Concepts](./concepts/) | Architecture and design decisions — the "why" behind each choice |
| [Package Layout](./packages/) | Package structure and interfaces — the "what" each package provides |
| [Full Architecture](./architecture/) | Complete architecture with extensibility patterns — the "how" it all fits together |

## Key Design Principles

- **Streaming-first** — `iter.Seq2[T, error]` everywhere, not channels
- **Pluggable everything** — Registry + Middleware + Hooks pattern in every package
- **Zero global state** — Only `init()` registrations; no mutable globals
- **Context propagation** — `context.Context` as first parameter, no exceptions
- **Composition over inheritance** — Struct embedding, not interface hierarchies
- **Own your execution** — Built-in durable execution engine; Temporal is a provider option, not the default
