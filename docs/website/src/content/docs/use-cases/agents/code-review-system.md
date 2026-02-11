---
title: Automated Code Review
description: Build an AI-powered code review agent with git integration, static analysis, and structured feedback using Beluga AI.
---

Code review is essential for maintaining quality but time-consuming and inconsistent. Human reviewers spend significant time on mechanical checks (style violations, lint errors, missing error handling) that leave less capacity for the high-value work: evaluating architecture decisions, spotting logic errors, and assessing security implications. Review quality also varies by reviewer fatigue, domain expertise, and time pressure.

An AI-powered code review agent automates the mechanical layer: running static analysis tools, identifying common bug patterns, checking for security issues, and generating structured feedback. Human reviewers then focus on architecture and design decisions — the work that requires understanding the broader system context. The agent uses a ReAct pattern because code review is inherently multi-step: fetch the diff, identify changed files, run appropriate linters per language, analyze the code, and synthesize findings into a structured report.

## Solution Architecture

The code review agent uses Beluga AI's tool system to interact with git repositories and static analysis tools. It fetches the diff, runs linters, analyzes the code with an LLM, and posts structured review comments. Safety guards ensure the agent's suggestions are constructive and accurate.

The tool-based approach (rather than a monolithic review script) allows the agent to reason about which tools to use based on the specific changes. A PR that only modifies Go files does not need ESLint; a PR that touches security-sensitive code triggers additional analysis. The ReAct loop handles this conditional logic naturally.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Webhook     │───▶│  Review      │───▶│  Git Tool    │
│  (PR opened) │    │  Agent       │    │  (Fetch diff,│
│              │    │  (ReAct)     │    │   file list) │
└──────────────┘    └──────┬───────┘    └──────────────┘
                           │
                    ┌──────┴───────┐
                    │              │
              ┌─────▼──────┐ ┌────▼───────┐
              │ Linter     │ │ LLM        │
              │ Tool       │ │ Analysis   │
              │ (go vet,   │ │ (Bugs,     │
              │  eslint)   │ │  security) │
              └─────┬──────┘ └────┬───────┘
                    │              │
                    └──────┬───────┘
                           ▼
                    ┌──────────────┐
                    │  Structured  │
                    │  Review      │
                    │  (Comments,  │
                    │   Score)     │
                    └──────────────┘
```

## Building the Review Agent

Define tools for git operations and code analysis, then create an agent that uses them to review pull requests. Each tool is created with `tool.NewFuncTool` using typed input structs, which auto-generates JSON schemas from struct tags. The agent's persona is set to "Senior Code Reviewer" with explicit instructions to be constructive — this prompt engineering is critical for producing useful review output rather than pedantic criticism.

```go
package main

import (
    "context"
    "fmt"
    "os/exec"
    "strings"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// GitDiffInput specifies which PR diff to fetch.
type GitDiffInput struct {
    Repo     string `json:"repo" jsonschema:"description=Repository in owner/repo format"`
    PRNumber int    `json:"pr_number" jsonschema:"description=Pull request number"`
}

// LintInput specifies what to lint.
type LintInput struct {
    FilePath string `json:"file_path" jsonschema:"description=Path to the file to lint"`
    Language string `json:"language" jsonschema:"enum=go,python,javascript,typescript"`
}

func createReviewAgent(ctx context.Context) (agent.Agent, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    // Git diff tool — fetch the PR changes
    diffTool := tool.NewFuncTool[GitDiffInput](
        "get_pr_diff",
        "Fetch the diff for a pull request",
        func(ctx context.Context, input GitDiffInput) (*tool.Result, error) {
            cmd := exec.CommandContext(ctx, "gh", "pr", "diff",
                fmt.Sprintf("%d", input.PRNumber),
                "--repo", input.Repo,
            )
            output, err := cmd.Output()
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(string(output)), nil
        },
    )

    // File list tool — get the list of changed files
    fileListTool := tool.NewFuncTool[GitDiffInput](
        "get_changed_files",
        "List files changed in a pull request",
        func(ctx context.Context, input GitDiffInput) (*tool.Result, error) {
            cmd := exec.CommandContext(ctx, "gh", "pr", "view",
                fmt.Sprintf("%d", input.PRNumber),
                "--repo", input.Repo,
                "--json", "files",
                "--jq", ".files[].path",
            )
            output, err := cmd.Output()
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(string(output)), nil
        },
    )

    // Linter tool — run static analysis
    lintTool := tool.NewFuncTool[LintInput](
        "run_linter",
        "Run static analysis on a file",
        func(ctx context.Context, input LintInput) (*tool.Result, error) {
            var cmd *exec.Cmd
            switch input.Language {
            case "go":
                cmd = exec.CommandContext(ctx, "golangci-lint", "run", input.FilePath)
            case "python":
                cmd = exec.CommandContext(ctx, "ruff", "check", input.FilePath)
            case "javascript", "typescript":
                cmd = exec.CommandContext(ctx, "eslint", input.FilePath)
            default:
                return tool.ErrorResult(fmt.Errorf("unsupported language: %s", input.Language)), nil
            }

            output, err := cmd.CombinedOutput()
            if err != nil {
                // Linters return non-zero on findings — that's expected
                return tool.TextResult(string(output)), nil
            }
            return tool.TextResult("No issues found"), nil
        },
    )

    reviewAgent, err := agent.New(
        agent.WithID("code-reviewer"),
        agent.WithPersona(agent.Persona{
            Role: "Senior Code Reviewer",
            Goal: "Identify bugs, security issues, and quality improvements in code changes",
            Backstory: "You are a thorough code reviewer with expertise in multiple languages. " +
                "Focus on correctness, security, performance, and maintainability. " +
                "Be constructive — explain why something is an issue and suggest a fix. " +
                "Do not flag style preferences as bugs.",
        }),
        agent.WithModel(model),
        agent.WithTools(diffTool, fileListTool, lintTool),
    )
    if err != nil {
        return nil, fmt.Errorf("create review agent: %w", err)
    }

    return reviewAgent, nil
}
```

## Structured Review Output

Use structured output to produce machine-readable review comments. The two-step approach — first run the agent for free-form analysis, then use `llm.NewStructured[ReviewReport]` to format the output — separates the reasoning step from the structuring step. This produces better results than asking the agent to both analyze and format simultaneously, because the LLM can focus entirely on code analysis in the first step and entirely on structuring in the second.

```go
type ReviewReport struct {
    Summary    string          `json:"summary"`
    Score      int             `json:"score" jsonschema:"description=Overall quality score 1-10"`
    Issues     []ReviewIssue   `json:"issues"`
    Strengths  []string        `json:"strengths"`
    Suggestion string          `json:"suggestion"`
}

type ReviewIssue struct {
    File       string `json:"file"`
    Line       int    `json:"line"`
    Severity   string `json:"severity" jsonschema:"enum=critical,warning,info"`
    Category   string `json:"category" jsonschema:"enum=bug,security,performance,style,maintainability"`
    Message    string `json:"message"`
    Suggestion string `json:"suggestion"`
}

func reviewPR(ctx context.Context, reviewAgent agent.Agent, model llm.ChatModel, repo string, prNumber int) (ReviewReport, error) {
    // Step 1: Agent analyzes the PR using tools
    analysis, err := reviewAgent.Invoke(ctx,
        fmt.Sprintf("Review pull request #%d in %s. "+
            "Fetch the diff, run linters on changed files, "+
            "and provide a detailed analysis of issues and quality.", prNumber, repo),
    )
    if err != nil {
        return ReviewReport{}, fmt.Errorf("review: %w", err)
    }

    // Step 2: Structure the analysis into a formal report
    structured := llm.NewStructured[ReviewReport](model)

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Convert this code review analysis into a structured report. " +
                "Be precise about file paths and line numbers."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: analysis},
        }},
    }

    report, err := structured.Generate(ctx, msgs)
    if err != nil {
        return ReviewReport{}, fmt.Errorf("structure report: %w", err)
    }

    return report, nil
}
```

## Webhook Handler

Trigger reviews automatically when pull requests are opened or updated:

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    var event struct {
        Action      string `json:"action"`
        Number      int    `json:"number"`
        Repository  struct {
            FullName string `json:"full_name"`
        } `json:"repository"`
    }

    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "invalid payload", http.StatusBadRequest)
        return
    }

    if event.Action != "opened" && event.Action != "synchronize" {
        w.WriteHeader(http.StatusOK)
        return
    }

    // Run review asynchronously
    go func() {
        ctx := context.Background()
        report, err := reviewPR(ctx, reviewAgent, model, event.Repository.FullName, event.Number)
        if err != nil {
            log.Printf("Review failed: %v", err)
            return
        }

        if err := postReviewComment(ctx, event.Repository.FullName, event.Number, report); err != nil {
            log.Printf("Post comment failed: %v", err)
        }
    }()

    w.WriteHeader(http.StatusAccepted)
}
```

## Posting Review Comments

Format the structured report and post it as a PR comment:

```go
func postReviewComment(ctx context.Context, repo string, prNumber int, report ReviewReport) error {
    var comment strings.Builder
    comment.WriteString(fmt.Sprintf("## AI Code Review (Score: %d/10)\n\n", report.Score))
    comment.WriteString(report.Summary + "\n\n")

    if len(report.Strengths) > 0 {
        comment.WriteString("### Strengths\n")
        for _, s := range report.Strengths {
            comment.WriteString(fmt.Sprintf("- %s\n", s))
        }
        comment.WriteString("\n")
    }

    if len(report.Issues) > 0 {
        comment.WriteString("### Issues Found\n\n")
        for _, issue := range report.Issues {
            icon := map[string]string{"critical": "!!!", "warning": "!", "info": "i"}[issue.Severity]
            comment.WriteString(fmt.Sprintf("**[%s] %s** — `%s:%d`\n", icon, issue.Category, issue.File, issue.Line))
            comment.WriteString(issue.Message + "\n")
            if issue.Suggestion != "" {
                comment.WriteString(fmt.Sprintf("> Suggestion: %s\n", issue.Suggestion))
            }
            comment.WriteString("\n")
        }
    }

    if report.Suggestion != "" {
        comment.WriteString("### Overall Suggestion\n" + report.Suggestion + "\n")
    }

    // Post via GitHub CLI
    cmd := exec.CommandContext(ctx, "gh", "pr", "comment",
        fmt.Sprintf("%d", prNumber),
        "--repo", repo,
        "--body", comment.String(),
    )
    return cmd.Run()
}
```

## Safety Guards for Code Review

Guard the agent's output to ensure reviews are constructive and accurate. Beluga AI's guard pipeline provides output guards that screen the agent's responses before they are posted. The tone guard catches harsh or personal language, and the security guard prevents the agent from suggesting code patterns that could introduce vulnerabilities. These guards implement the output stage of Beluga AI's 3-stage guard pipeline (Input, Output, Tool).

```go
import "github.com/lookatitude/beluga-ai/guard"

// Guard against overly harsh or personal criticism
toneGuard := guard.GuardFunc("constructive-tone", func(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
    upper := strings.ToUpper(input.Content)
    harshPhrases := []string{"TERRIBLE CODE", "INCOMPETENT", "NEVER DO THIS", "AWFUL"}
    for _, phrase := range harshPhrases {
        if strings.Contains(upper, phrase) {
            return guard.GuardResult{
                Allowed: false,
                Reason:  "Review comments must be constructive",
            }, nil
        }
    }
    return guard.GuardResult{Allowed: true}, nil
})

// Guard against suggesting code changes that could introduce vulnerabilities
securityGuard := guard.GuardFunc("no-insecure-suggestions", func(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
    insecurePatterns := []string{"eval(", "exec(", "os.system(", "innerHTML ="}
    for _, pattern := range insecurePatterns {
        if strings.Contains(input.Content, pattern) {
            return guard.GuardResult{
                Allowed: false,
                Reason:  "Suggestion contains potentially insecure pattern: " + pattern,
            }, nil
        }
    }
    return guard.GuardResult{Allowed: true}, nil
})
```

## Production Considerations

### Observability

Track review metrics for quality monitoring:

```go
span.SetAttributes(
    attribute.String("review.repo", repo),
    attribute.Int("review.pr_number", prNumber),
    attribute.Int("review.score", report.Score),
    attribute.Int("review.issues_found", len(report.Issues)),
    attribute.Int("review.critical_count", countBySeverity(report.Issues, "critical")),
)
```

### Resilience

- Retry GitHub API calls with exponential backoff (rate limits are common)
- Set timeouts on linter executions to prevent hangs on large files
- Use circuit breakers for the LLM provider — if the provider is down, skip AI review and fall back to linter-only output

### Security

- Run linters in sandboxed environments (containers) to prevent code execution from untrusted repositories
- Validate repository and PR parameters to prevent injection attacks
- Use read-only GitHub tokens — the agent should comment but never modify code
- Limit the diff size the agent processes to prevent excessive token usage

### Scaling

- Process reviews asynchronously via a message queue (one worker per review)
- Cache linter results for files that haven't changed between pushes
- Rate-limit reviews per repository to avoid overwhelming the GitHub API
- For monorepos, split the diff by directory and review in parallel

## Related Resources

- [Tools & MCP](/guides/tools-and-mcp/) for building custom tools
- [LLM Recipes](/cookbook/llm-recipes/) for typed LLM responses
- [Production Agent Platform](/use-cases/production-platform/) for deployment patterns
