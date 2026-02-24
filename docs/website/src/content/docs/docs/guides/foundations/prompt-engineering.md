---
title: Prompt Management & Engineering
description: "Design production prompts with Beluga AI's PromptManager — template resolution, cache-optimal token ordering, few-shot selection, versioning, and A/B testing in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, prompt engineering, templates, versioning, cache optimization, A/B testing, few-shot"
---

Prompts are the programming interface to language models. The difference between a mediocre AI feature and a reliable one often comes down to prompt design — how you structure instructions, provide examples, and order content for cache efficiency. Beluga AI's `prompt` package provides three tools for production prompt management: `Template` for parameterized prompt rendering, `PromptManager` for versioned template storage and retrieval, and `Builder` for assembling message sequences in cache-optimal order.

## What You'll Learn

This guide covers:
- Designing effective prompts and system messages
- Using the `PromptManager` and template system
- Building cache-optimized message sequences with `Builder`
- Creating reusable persona libraries
- Few-shot learning patterns
- Prompt versioning and A/B testing

## When Prompt Engineering Matters

For simple, one-off interactions, inline string prompts work fine. Invest in the prompt management system when:
- **Consistency is critical** — multiple agents or endpoints share the same prompt logic, and a change must propagate everywhere
- **Non-technical teams** need to update AI behavior without modifying Go code
- **Testing variations** — you want to A/B test prompt versions with metrics tracking
- **Minimizing costs** — prompt caching can reduce token costs significantly, but requires careful content ordering
- **Audit and compliance** — you need to track which prompt version produced each response

## Prerequisites

Before starting this guide:
- Complete [Working with LLMs](/guides/foundations/working-with-llms/)
- Understand Go templates (`text/template`)
- Familiarity with JSON and YAML

## The PromptManager

The `PromptManager` interface separates prompt content from application code. Templates are stored in an external source (filesystem, database, or custom backend), loaded by name and version, and rendered with variables at runtime. This separation means prompt authors can iterate on wording without redeploying the application, and prompt changes are tracked independently of code changes.

The `prompt/providers/file` package implements `PromptManager` for filesystem-backed templates, making it easy to version prompts alongside your code in git.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/prompt"
    "github.com/lookatitude/beluga-ai/prompt/providers/file"
)

func main() {
    ctx := context.Background()

    // Load templates from a directory
    manager, err := file.New("./prompts")
    if err != nil {
        log.Fatal(err)
    }

    // Render a template by name with variables
    messages, err := manager.Render("greeting", map[string]any{
        "Name":    "Alice",
        "Service": "Beluga AI",
    })
    if err != nil {
        log.Fatal(err)
    }

    // messages is []schema.Message, ready to send to a ChatModel
    fmt.Println(messages)
}
```

## Template Syntax

Templates use Go's `text/template` syntax, which provides conditionals, loops, and variable substitution. This is a deliberate choice over custom template languages — Go developers already know the syntax, and the standard library template engine is well-tested and performant.

```go
// Template file: prompts/customer_support.tmpl
// Version: 1.0

// System message template with conditionals and loops
systemTemplate := `You are a customer support agent for {{.Company}}.

Guidelines:
- Be polite and empathetic
- Provide accurate information
- If unsure, escalate to human support
- Keep responses under 3 paragraphs

{{if .Policies}}Company policies:
{{range .Policies}}- {{.}}
{{end}}{{end}}`
```

## Building Cache-Optimized Prompts with Builder

LLM providers cache prompt prefixes to avoid reprocessing repeated content. The `prompt.Builder` enforces a message ordering that maximizes cache hit rates by placing the most static content first and the most dynamic content last. This ordering can significantly reduce both latency and cost for high-volume applications.

The builder organizes content into six ordered slots:

1. **System prompt** — most static, rarely changes across requests
2. **Tool definitions** — semi-static, change only when tools are added or removed
3. **Static context** — reference documents, retrieved knowledge, deployment-specific content
4. **Cache breakpoint** — explicit marker where the cached prefix ends
5. **Dynamic context** — conversation history, session-specific messages
6. **User input** — always changes, placed last

This ordering is not arbitrary. Each slot is ordered by decreasing stability: the system prompt is identical across all requests, tool definitions change only on code deployment, static context changes per deployment or retrieval cycle, and user input is unique per request. By placing stable content first, the LLM provider can cache and reuse the maximum prefix length.

```go
import (
    "github.com/lookatitude/beluga-ai/prompt"
    "github.com/lookatitude/beluga-ai/schema"
)

msgs := prompt.NewBuilder(
    prompt.WithSystemPrompt("You are an expert Go developer."),
    prompt.WithToolDefinitions(tools),
    prompt.WithStaticContext([]string{
        "Go style guide: use gofmt, prefer short variable names...",
        "Project conventions: all errors must be wrapped with fmt.Errorf...",
    }),
    prompt.WithCacheBreakpoint(),
    prompt.WithDynamicContext(conversationHistory),
    prompt.WithUserInput(schema.NewHumanMessage("Review this function for bugs.")),
).Build()

// msgs is []schema.Message in cache-optimal order
resp, err := model.Generate(ctx, msgs)
```

## Creating Reusable Personas

When multiple agents share similar behavioral patterns, define personas as reusable templates. This approach centralizes behavior definitions so changes propagate consistently. The Persona struct in the `agent` package uses a Role-Goal-Backstory framework, but for prompt-level personas you can use templates with variables for customization points.

The example below defines a persona library as a map of templates. Each persona specifies not just the system prompt text but also recommended generation parameters (temperature, max tokens) that match the persona's purpose — a code reviewer needs low temperature for precision, while a creative writer needs high temperature for variety.

```go
type Persona struct {
    Name         string
    Description  string
    SystemPrompt string
    Temperature  float64
    MaxTokens    int
}

// Define persona library
var PersonaLibrary = map[string]Persona{
    "code_reviewer": {
        Name:        "Code Reviewer",
        Description: "Senior engineer reviewing code for best practices",
        SystemPrompt: `You are a senior software engineer reviewing code.

Focus areas:
- Code quality and readability
- Security vulnerabilities
- Performance issues
- Best practices for {{.Language}}

Output format:
1. Summary (2-3 sentences)
2. Issues found (list)
3. Recommendations (list)

Be constructive and specific.`,
        Temperature: 0.3,
        MaxTokens:   1000,
    },

    "creative_writer": {
        Name:        "Creative Writer",
        Description: "Novelist creating engaging narratives",
        SystemPrompt: `You are a creative writer specializing in {{.Genre}}.

Style guidelines:
- Use vivid, descriptive language
- Show, don't tell
- Develop complex characters
- Create engaging dialogue
- Maintain consistent tone

Target audience: {{.Audience}}`,
        Temperature: 0.9,
        MaxTokens:   2000,
    },

    "data_analyst": {
        Name:        "Data Analyst",
        Description: "Analyst extracting insights from data",
        SystemPrompt: `You are a data analyst specializing in {{.Domain}}.

Analysis approach:
- State clear hypotheses
- Use statistical reasoning
- Identify patterns and trends
- Provide actionable insights
- Visualize findings when helpful

Always cite specific data points.`,
        Temperature: 0.2,
        MaxTokens:   1500,
    },
}

// Render persona template with variables
func GetPersonaMessage(personaName string, variables map[string]any) (schema.Message, error) {
    persona, ok := PersonaLibrary[personaName]
    if !ok {
        return nil, fmt.Errorf("unknown persona: %s", personaName)
    }

    tmpl, err := template.New(persona.Name).Parse(persona.SystemPrompt)
    if err != nil {
        return nil, err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, variables); err != nil {
        return nil, err
    }

    return schema.NewSystemMessage(buf.String()), nil
}
```

## Few-Shot Learning

Few-shot prompts teach the model by example rather than by instruction. This is effective when the desired behavior is hard to describe in words but easy to demonstrate through input-output pairs. The model learns the pattern from the examples and applies it to the new input. For classification tasks, 3-5 examples typically provide sufficient signal. For more complex generation tasks, you may need more examples or a combination of instructions and examples.

In Beluga's message-based API, few-shot examples are naturally represented as alternating `HumanMessage` and `AIMessage` pairs in the conversation history, placed between the system prompt and the actual user input.

```go
type FewShotExample struct {
    Input  string
    Output string
}

func CreateFewShotPrompt(task string, examples []FewShotExample, newInput string) string {
    var prompt strings.Builder

    // Task description
    prompt.WriteString(fmt.Sprintf("Task: %s\n\n", task))

    // Examples
    prompt.WriteString("Examples:\n")
    for i, ex := range examples {
        prompt.WriteString(fmt.Sprintf("\nExample %d:\n", i+1))
        prompt.WriteString(fmt.Sprintf("Input: %s\n", ex.Input))
        prompt.WriteString(fmt.Sprintf("Output: %s\n", ex.Output))
    }

    // New input
    prompt.WriteString(fmt.Sprintf("\nNow complete this:\nInput: %s\nOutput:", newInput))

    return prompt.String()
}

// Example: Sentiment classification with few-shot examples
func ClassifySentimentFewShot(ctx context.Context, model llm.ChatModel, text string) (string, error) {
    examples := []FewShotExample{
        {
            Input:  "This product is amazing! Best purchase ever.",
            Output: "Positive",
        },
        {
            Input:  "Terrible quality. Waste of money.",
            Output: "Negative",
        },
        {
            Input:  "It's okay, nothing special.",
            Output: "Neutral",
        },
    }

    prompt := CreateFewShotPrompt(
        "Classify the sentiment of the following text as Positive, Negative, or Neutral.",
        examples,
        text,
    )

    messages := []schema.Message{
        schema.NewHumanMessage(prompt),
    }

    resp, err := model.Generate(ctx, messages,
        llm.WithTemperature(0.0),
        llm.WithMaxTokens(10),
    )
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(resp.Text()), nil
}
```

## Dynamic Prompt Building

When prompts need to be assembled from multiple sources at runtime — system instructions, constraints, context documents, few-shot examples, and user input — a builder pattern keeps the assembly logic organized and composable. Each `With` method adds content to a specific section, and `Build` assembles them into a properly ordered message sequence. This pattern is complementary to `prompt.Builder`: use `prompt.Builder` when you need cache-optimal slot ordering, and a custom builder when you need flexible content assembly.

```go
type PromptBuilder struct {
    systemInstructions []string
    fewShotExamples    []FewShotExample
    contextDocs        []string
    constraints        []string
    outputFormat       string
}

func NewPromptBuilder() *PromptBuilder {
    return &PromptBuilder{}
}

func (pb *PromptBuilder) WithSystemInstruction(instruction string) *PromptBuilder {
    pb.systemInstructions = append(pb.systemInstructions, instruction)
    return pb
}

func (pb *PromptBuilder) WithFewShotExample(input, output string) *PromptBuilder {
    pb.fewShotExamples = append(pb.fewShotExamples, FewShotExample{
        Input:  input,
        Output: output,
    })
    return pb
}

func (pb *PromptBuilder) WithContext(doc string) *PromptBuilder {
    pb.contextDocs = append(pb.contextDocs, doc)
    return pb
}

func (pb *PromptBuilder) WithConstraint(constraint string) *PromptBuilder {
    pb.constraints = append(pb.constraints, constraint)
    return pb
}

func (pb *PromptBuilder) WithOutputFormat(format string) *PromptBuilder {
    pb.outputFormat = format
    return pb
}

func (pb *PromptBuilder) Build(userQuery string) []schema.Message {
    var systemPrompt strings.Builder

    // System instructions
    if len(pb.systemInstructions) > 0 {
        systemPrompt.WriteString("Instructions:\n")
        for _, inst := range pb.systemInstructions {
            systemPrompt.WriteString(fmt.Sprintf("- %s\n", inst))
        }
        systemPrompt.WriteString("\n")
    }

    // Constraints
    if len(pb.constraints) > 0 {
        systemPrompt.WriteString("Constraints:\n")
        for _, constraint := range pb.constraints {
            systemPrompt.WriteString(fmt.Sprintf("- %s\n", constraint))
        }
        systemPrompt.WriteString("\n")
    }

    // Output format
    if pb.outputFormat != "" {
        systemPrompt.WriteString(fmt.Sprintf("Output format: %s\n\n", pb.outputFormat))
    }

    // Build messages
    messages := []schema.Message{
        schema.NewSystemMessage(systemPrompt.String()),
    }

    // Add few-shot examples as conversation history
    for _, ex := range pb.fewShotExamples {
        messages = append(messages,
            schema.NewHumanMessage(ex.Input),
            schema.NewAIMessage(ex.Output),
        )
    }

    // Add context documents
    if len(pb.contextDocs) > 0 {
        var contextMsg strings.Builder
        contextMsg.WriteString("Relevant context:\n\n")
        for i, doc := range pb.contextDocs {
            contextMsg.WriteString(fmt.Sprintf("[Document %d]\n%s\n\n", i+1, doc))
        }
        messages = append(messages, schema.NewSystemMessage(contextMsg.String()))
    }

    // Add user query
    messages = append(messages, schema.NewHumanMessage(userQuery))

    return messages
}

// Usage
func main() {
    builder := NewPromptBuilder().
        WithSystemInstruction("You are a helpful SQL assistant").
        WithSystemInstruction("Generate valid PostgreSQL syntax only").
        WithConstraint("Do not use deprecated functions").
        WithConstraint("Include comments explaining complex logic").
        WithOutputFormat("SQL code block with explanation").
        WithFewShotExample(
            "Get all users who signed up last month",
            "SELECT * FROM users WHERE created_at >= DATE_TRUNC('month', NOW() - INTERVAL '1 month')",
        ).
        WithContext("Schema: users (id, name, email, created_at)")

    messages := builder.Build("Find users who haven't logged in for 30 days")
    // ... send to LLM
}
```

## Prompt Versioning

Versioning prompts is essential for reproducibility and safe iteration. When a prompt change degrades output quality, you need to know which version caused the regression and roll back to the previous one. The pattern below shows a thread-safe prompt registry that stores multiple versions of each prompt and supports "latest" lookups. In production, back this with a database or the filesystem-based `PromptManager` rather than an in-memory map.

```go
type VersionedPrompt struct {
    ID          string
    Version     string
    Name        string
    Template    string
    CreatedAt   time.Time
    Performance *PromptMetrics
}

type PromptMetrics struct {
    AvgLatency     time.Duration
    SuccessRate    float64
    AvgTokens      int
    UserSatisfaction float64
}

type PromptRegistry struct {
    prompts map[string]map[string]VersionedPrompt // name -> version -> prompt
    mu      sync.RWMutex
}

func NewPromptRegistry() *PromptRegistry {
    return &PromptRegistry{
        prompts: make(map[string]map[string]VersionedPrompt),
    }
}

func (pr *PromptRegistry) Register(prompt VersionedPrompt) {
    pr.mu.Lock()
    defer pr.mu.Unlock()

    if _, ok := pr.prompts[prompt.Name]; !ok {
        pr.prompts[prompt.Name] = make(map[string]VersionedPrompt)
    }

    pr.prompts[prompt.Name][prompt.Version] = prompt
}

func (pr *PromptRegistry) Get(name, version string) (VersionedPrompt, error) {
    pr.mu.RLock()
    defer pr.mu.RUnlock()

    versions, ok := pr.prompts[name]
    if !ok {
        return VersionedPrompt{}, fmt.Errorf("prompt not found: %s", name)
    }

    // If version is "latest", get the newest
    if version == "latest" {
        var latest VersionedPrompt
        var latestTime time.Time

        for _, p := range versions {
            if p.CreatedAt.After(latestTime) {
                latest = p
                latestTime = p.CreatedAt
            }
        }

        if latestTime.IsZero() {
            return VersionedPrompt{}, fmt.Errorf("no versions found for: %s", name)
        }

        return latest, nil
    }

    prompt, ok := versions[version]
    if !ok {
        return VersionedPrompt{}, fmt.Errorf("version not found: %s@%s", name, version)
    }

    return prompt, nil
}

func (pr *PromptRegistry) ListVersions(name string) []string {
    pr.mu.RLock()
    defer pr.mu.RUnlock()

    versions, ok := pr.prompts[name]
    if !ok {
        return nil
    }

    result := make([]string, 0, len(versions))
    for v := range versions {
        result = append(result, v)
    }

    sort.Strings(result)
    return result
}
```

Prompts can be loaded from YAML files, making them easy to version in git alongside your code:

```yaml
- id: code-review-v1
  version: "1.0"
  name: code_reviewer
  template: |
    You are a code reviewer. Review the following code for bugs and best practices.
    Code: {{.Code}}
  created_at: 2025-01-01T00:00:00Z

- id: code-review-v2
  version: "2.0"
  name: code_reviewer
  template: |
    You are an expert code reviewer. Analyze this code for:
    - Security vulnerabilities
    - Performance issues
    - Code style violations
    - Best practices

    Code: {{.Code}}

    Provide specific line numbers and recommendations.
  created_at: 2025-02-01T00:00:00Z
```

## A/B Testing Prompts

Prompt A/B testing lets you compare two versions quantitatively before committing to a rollout. The pattern below randomly assigns each request to variant A or B, tracks metrics for each variant, and provides a simple scoring function to determine the winner. In production, integrate this with your observability system to collect metrics automatically via LLM hooks.

```go
type ABTest struct {
    Name          string
    VariantA      VersionedPrompt
    VariantB      VersionedPrompt
    SplitRatio    float64 // 0.5 = 50/50 split
    MetricsA      *PromptMetrics
    MetricsB      *PromptMetrics
}

func (test *ABTest) GetVariant() VersionedPrompt {
    if rand.Float64() < test.SplitRatio {
        return test.VariantA
    }
    return test.VariantB
}

func (test *ABTest) RecordMetrics(variant string, latency time.Duration, success bool, tokens int) {
    metrics := test.MetricsA
    if variant == "B" {
        metrics = test.MetricsB
    }

    // Update metrics (use proper aggregation in production)
    metrics.AvgLatency = (metrics.AvgLatency + latency) / 2
    if success {
        metrics.SuccessRate = (metrics.SuccessRate + 1.0) / 2
    } else {
        metrics.SuccessRate = metrics.SuccessRate / 2
    }
    metrics.AvgTokens = (metrics.AvgTokens + tokens) / 2
}

func (test *ABTest) GetWinner() string {
    // Simple scoring: higher success rate and lower latency wins
    scoreA := test.MetricsA.SuccessRate / float64(test.MetricsA.AvgLatency.Seconds())
    scoreB := test.MetricsB.SuccessRate / float64(test.MetricsB.AvgLatency.Seconds())

    if scoreA > scoreB {
        return "A"
    }
    return "B"
}
```

## Production Best Practices

When engineering prompts for production:

1. **Separate prompts from code** — use `PromptManager` or YAML files so prompt changes do not require code deploys
2. **Version all prompts** — track which version produced each response for debugging and compliance
3. **Use `prompt.Builder` for cache optimization** — place static content first, dynamic content last
4. **Test systematically** — use the `eval` package to measure prompt quality across diverse inputs
5. **Monitor token usage** — track input and output tokens per prompt version to detect bloat
6. **Validate template variables** — ensure all required variables are provided before rendering to avoid template errors at runtime
7. **Document prompt intent** — describe what each prompt is designed to achieve and what assumptions it makes
8. **Implement gradual rollouts** — use A/B testing to validate prompt changes before full deployment
9. **Keep prompts focused** — one prompt per task; avoid multi-purpose prompts that are hard to optimize
10. **Store prompts in version control** — treat prompts as code artifacts with the same review and deployment rigor

## Next Steps

Now that you understand prompt engineering:
- Learn about [Structured Output](/guides/foundations/structured-output/) for typed responses
- Explore [Working with LLMs](/guides/foundations/working-with-llms/) for model configuration
- Read [Building Your First Agent](/guides/foundations/first-agent/) for using prompts with agents
