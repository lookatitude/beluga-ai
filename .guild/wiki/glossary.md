---
schema_version: guild.glossary.v1
title: Glossary
importance: medium
description: Guild operating terms. Add this project's own terms below; Guild never overwrites this file.
recall_importance: 3
---

# Glossary

Terms the Guild machinery uses. The context-manager attaches matching entries to a
specialist bundle on demand — this file is never loaded whole into the always-on
prefix (KTD70).

## Guild terms

- **root** — a directory with a `.guild/` tree. Guild has exactly two levels: an
  umbrella workspace root and its immediate project roots.
- **durable** — content under `.guild/` that you would lose by deleting it. Caches,
  scratch and runtime state are NOT durable and live off the repo.
- **run** — one lifecycle execution, recorded under `.guild/runs/<run-id>/`.
- **lane** — one specialist's slice of a plan, dispatched as its own task cell.
- **handoff** — the compact envelope a specialist returns. The parent reads the
  envelope, never the specialist's transcript.
- **tier** — `cheap` | `mid` | `powerful`. A selector, never a model name; the
  host adapter maps a tier to a model at dispatch.
- **session binding** — the host and models resolved for ONE run. Session state,
  never durable config.
- **harvest** — the automatic capture of a decision into this root's wiki. The
  only unattended writer there is; everything else goes through a human.
- **class** — which of the five graphs a run follows: `product`, `research`,
  `debug`, `ops`, `init`. Bound once at intake, by a typed verb, or by
  `--class=`.
- **cursor** — where a run currently sits on its class graph. Read on crash resume.
- **working set** — the small card a phase reads first: pinned decision ids, open
  questions, and a fingerprint that says whether any of it can have changed.
- **lane bundle** — the ≤1200-token citation list a parent sees. Parents get
  citations; only the specialist reads the sources.
- **recall** — looking something up before doing it. Cheap and mandatory; research
  is what happens when recall misses.
- **redirect** — an operator correction that made the orchestrator change course.
  Three on the same agent and topic in one run fire a harvest.
- **layout version** — the integer in `.guild/storage-layout.json` saying which
  storage layout this root is on. Guild upgrades a root on activation.

## Project terms

Add yours here. This file is yours once it exists.
