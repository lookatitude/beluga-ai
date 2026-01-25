---
sidebar_position: 1
id: agent-design
title: Designing Agents
---

# Designing Agents

> "Code is cheap, context is expensive."

In Beluga AI, the difference between a "script" and an "agent" is the **Persona**. While the `agents` package provides the technical implementation (the *how*), this guide covers the **Agent Design** (the *what* and *why*).

## The 80/20 Rule

When building agentic applications, follow the **80/20 Rule**:

*   **80%** of your effort should go into designing the **Task**, **Role**, and **Backstory**.
*   **20%** of your effort should go into the actual coding (wiring up the `Agent` struct).

If an agent is failing, 9 times out of 10, it is a **definition** problem, not a code problem.

## The Role-Goal-Backstory (RGB) Framework

To interpret instructions correctly, an LLM needs 3 key pieces of context, known as the **RGB Framework**.

### 1. Role
**What job does this agent have?**
Be specific. Instead of "Assistant", use "Senior React Engineer" or "Financial Risk Auditor".
The role sets the **capability baseline**.

### 2. Goal
**What is the concrete outcome?**
The goal acts as the "North Star". It helps the agent prioritize tasks and know when it is finished.
*   *Bad*: "Analyze the data."
*   *Good*: "Analyze the Q3 sales data to identify at least 3 underperforming regions."

### 3. Backstory
**What creates the behavior?**
The backstory adds nuance, tone, and specific constraints. It tells the agent *how* to work.
*   *Example*: "You are a pragmatic engineer who prefers simple, proven solutions over complex, cutting-edge ones. You explain concepts using real-world analogies."

## Example: The Difference Design Makes

### ❌ Poor Design
```text
go
go
// Generic identity -> Generic results
agent := agents.NewBaseAgent(
    "helper",
    llm,
    tools,
    // Context: "You are a helpful assistant."
)
```

**Result**: The agent might give a vague, polite answer but fail to solve specific edge cases.

### ✅ Great Design
```text
go
go
// Specific identity -> High-quality results
agent := agents.NewBaseAgent(
    "compliance_officer",
    llm,
    tools,
    // Context: "You are a Senior Compliance Officer with 20 years of experience in GDPR.
    // Goal: Audit the provided database schema for PII leaks.
    // Backstory: You are meticulous and paranoid about data safety. You assume everything is a risk until proven otherwise."
)
```

**Result**: The agent will aggressively check for issues, flag potential risks, and explain *why* they matter, because its "paranoia" is encoded in the backstory.

## When to Split Agents

One "God Agent" is rarely the answer. Split tasks into multiple agents when:
1.  **Context Switching**: The task requires different "hats" (e.g., Creative Writing vs. Code Review).
2.  **Tool Overload**: An agent has too many tools (more than 5-7) and gets confused.
3.  **Process Stages**: The output of step A needs to be reviewed before step B (e.g., Writer -> Editor).

See [Orchestration Concepts](./orchestration.md) for how to make agents work together.
