---
name: consult
description: Escape hatch for framework agents (or humans) to bounce a design question to a workspace specialist mid-implementation. Produces a citable consultation artifact in framework/docs/consultations/.
---

Consult: $ARGUMENTS

$ARGUMENTS must be: `<specialist-name> <question>` where `<specialist-name>` is one of: `ai-ml-expert`, `rag-expert`, `security-architect`, `systems-architect`, `devops-expert`, `observability-expert`. Anything after the first token is the question.

Example:

```
/consult observability-expert How should harness.Invoke emit spans to match gen_ai.* conventions when approval gates interpose between agent and LLM?
```

## Steps

### 1. Parse arguments

Split $ARGUMENTS on whitespace. First token = specialist name. Remainder = question. Reject (with a clear error) if:
- First token is not one of the six valid specialist names
- Question is empty or under 10 characters (likely missing)

### 2. Gather current context

Collect:
- Current git branch + most recent commit (via `git log -1 --oneline`)
- Current feature's Linear parent (if detectable — look at the branch name for `loo-NN` and fetch via Linear MCP if a sub-issue with that prefix is currently open)
- Current brief (if the Linear parent references one at `../research/briefs/<slug>.md`)
- File tree snapshot of the package being worked on (`git diff --name-only main...HEAD`)

### 3. Invoke the workspace specialist

The specialist agents live in the workspace repo at `../.claude/agents/<name>.md`. Invoke via subagent with a prompt that includes:

- The question from $ARGUMENTS
- The context gathered in step 2
- The instruction: "Produce a consultation response at framework/docs/consultations/<YYYY-MM-DD>-<feature-slug>-<specialist>.md. Use your standard output format (Question / Analysis / Recommendation / Tradeoffs / Risks / Sources)."

The specialist's normal output path is `research/briefs/<slug>/specialist-<name>.md`; consultations use a different path (`framework/docs/consultations/`) so they coexist with specialist design-time inputs without overwriting.

### 4. Verify the file was produced

```bash
ls framework/docs/consultations/<YYYY-MM-DD>-*-<specialist>.md
```

If not produced: report the specialist failed and include their output for manual capture.

### 5. Print the path + one-sentence summary

```
Consulted <specialist-name>. Response at framework/docs/consultations/<YYYY-MM-DD>-<slug>-<specialist>.md.
Summary: <first line of the Recommendation section>

Cite this in your PR description or commit message so reviewers can find it.
```

## Purpose and boundaries

`/consult` is an escape hatch for the framework team — architect, developer-go, security-reviewer, QA — when a question falls outside their core expertise and waiting for a full design round is overkill.

**Good uses:**
- Architect hits a question about OTel span design mid-planning; one /consult call beats a whole design iteration.
- Developer-go encounters a surprise layering tension during implementation; /consult systems-architect clarifies.
- Security-reviewer needs a threat model review for a code change; /consult security-architect produces it.

**Bad uses:**
- Using /consult as a substitute for thinking. The reviewer-qa agent may flag "this consultation could have been decided locally" as feedback.
- Bouncing every design question to consultation. If you're calling it more than twice per PR, the feature probably needs a full /design-feature (A3) round at the workspace instead.

## On consultation cleanup

Consultations persist in `framework/docs/consultations/`. They're citable history — do not clean them up after a feature ships. If the directory grows unwieldy later, a policy can be defined then.
