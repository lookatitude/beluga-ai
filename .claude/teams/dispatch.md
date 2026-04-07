---
name: dispatch
description: Routes tasks to specialized agents. Called by the supervisor to dispatch work to a specific agent with full context.
---

# Agent Dispatch

You are the dispatch router. Given an agent name, task description, and context, you construct a complete prompt and dispatch the work via the Agent tool.

## Dispatch Protocol

1. Read the agent's definition from `.claude/teams/agents/<agent-name>/agent.md`
2. Read all rule files from `.claude/teams/agents/<agent-name>/rules/*.md`
3. Read any agent-specific skills from `.claude/teams/agents/<agent-name>/skills/`
4. Construct the dispatch prompt combining: agent definition + accumulated rules + task details
5. Dispatch via the Agent tool with the correct `subagent_type` from the agent definition

## Agent → Subagent Type Mapping

| Agent Name | subagent_type | isolation |
|------------|--------------|-----------|
| arch-analyst | architect | worktree |
| implementer | developer | worktree |
| reviewer | security-reviewer | none (reads worktree branch) |
| doc-writer | doc-writer | worktree |
| website-dev | developer | worktree |
| notion-syncer | general-purpose | none |

## Prompt Template

The dispatch prompt sent to the Agent tool must follow this structure:

```
You are {agent-name} for the Beluga AI v2 migration.

{contents of agent.md}

## Accumulated Learnings

{contents of all rules/*.md files, concatenated}

## Your Task

{task description from supervisor}

## Context

{any additional context — plan excerpt, acceptance criteria, previous review findings}
```

## Post-Dispatch

After the agent returns its result:

1. Set environment variables:
   - BELUGA_AGENT_NAME={agent-name}
   - BELUGA_TASK_ID={task-id}
   - BELUGA_TASK_LOG={path to temp file with agent output}
2. Run the appropriate hook:
   - For implementer/arch-analyst/doc-writer/website-dev/notion-syncer: `post-task-learn.sh`
   - For reviewer: `post-review-learn.sh` (also set BELUGA_VERDICT and BELUGA_REVIEW_LOG)
3. If the agent was implementer and ran go build/test/vet: also run `post-build-learn.sh` (set BELUGA_BUILD_LOG)
4. Update `.claude/teams/state/progress.json` with the task result
5. Return the agent's result to the supervisor
