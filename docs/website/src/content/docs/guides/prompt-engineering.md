---
title: Prompt Management & Engineering
description: Master prompt design, templating, versioning, and optimization for production AI systems.
---

Prompts are the programming interface to language models. Well-engineered prompts produce consistent, high-quality responses. Beluga AI's prompt system provides templates, versioning, and cache optimization for production deployments.

## What You'll Learn

This guide covers:
- Designing effective prompts and system messages
- Using the PromptManager and template system
- Creating reusable persona libraries
- Few-shot learning patterns
- Prompt versioning and A/B testing
- Cache optimization strategies

## When Prompt Engineering Matters

Invest in prompt engineering when:
- **Consistency is critical** across many requests
- **Multiple agents** need to share prompts
- **Non-technical teams** need to update AI behavior
- **Testing variations** to optimize quality
- **Minimizing costs** through prompt caching

## Prerequisites

Before starting this guide:
- Complete [Working with LLMs](/guides/working-with-llms)
- Understand Go templates (`text/template`)
- Familiarity with JSON and YAML

## The PromptManager

Beluga AI's prompt system separates prompt logic from data through templates.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    ctx := context.Background()

    // Create a prompt manager
    manager, err := prompts.NewPromptManager(
        prompts.WithConfig(prompts.DefaultConfig()),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create a template
    template, err := manager.CreateStringPrompt(
        "greeting",
        "Hello {{.Name}}! Welcome to {{.Service}}.",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Format with variables
    result, err := template.Format(ctx, map[string]interface{}{
        "Name":    "Alice",
        "Service": "Beluga AI",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.ToString())
    // Output: Hello Alice! Welcome to Beluga AI.
}
```

## Building Message Templates

Most LLMs expect structured conversations with system, user, and assistant messages.

```go
// Create a chat template
func CreateCustomerSupportTemplate() (*prompts.ChatPrompt, error) {
    manager, _ := prompts.NewPromptManager()

    return manager.CreateChatPrompt(
        "customer_support",
        prompts.WithSystemTemplate(`You are a customer support agent for {{.Company}}.

Guidelines:
- Be polite and empathetic
- Provide accurate information
- If unsure, escalate to human support
- Keep responses under 3 paragraphs

Company policies:
{{range .Policies}}- {{.}}
{{end}}`),
        prompts.WithUserTemplate("Customer inquiry: {{.Question}}"),
    )
}

func HandleSupportRequest(ctx context.Context, question string) ([]schema.Message, error) {
    template, err := CreateCustomerSupportTemplate()
    if err != nil {
        return nil, err
    }

    variables := map[string]interface{}{
        "Company": "Beluga AI",
        "Policies": []string{
            "30-day refund policy",
            "24/7 support availability",
            "Data privacy guaranteed",
        },
        "Question": question,
    }

    messages, err := template.Format(ctx, variables)
    if err != nil {
        return nil, err
    }

    return messages, nil
}
```

## Creating Reusable Personas

Personas define consistent agent behavior across your application.

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

// Get persona and create system message
func GetPersonaMessage(personaName string, variables map[string]interface{}) (schema.Message, error) {
    persona, ok := PersonaLibrary[personaName]
    if !ok {
        return nil, fmt.Errorf("unknown persona: %s", personaName)
    }

    // Apply template
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

// Usage
func main() {
    ctx := context.Background()

    // Get code reviewer persona
    sysMsg, err := GetPersonaMessage("code_reviewer", map[string]interface{}{
        "Language": "Go",
    })
    if err != nil {
        log.Fatal(err)
    }

    messages := []schema.Message{
        sysMsg,
        schema.NewHumanMessage("Review this code: func add(a, b int) { return a + b }"),
    }

    // Generate with persona
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithTemperature(PersonaLibrary["code_reviewer"].Temperature),
        llms.WithMaxTokens(PersonaLibrary["code_reviewer"].MaxTokens),
    )
    // ... generate response
}
```

## Few-Shot Learning

Few-shot prompts teach models by example.

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

// Example: Sentiment classification
func ClassifySentimentFewShot(ctx context.Context, llm llms.LLM, text string) (string, error) {
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

    response, err := llm.Generate(ctx, messages,
        llms.WithTemperature(0.0),
        llms.WithMaxTokens(10),
    )
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(response.Content), nil
}
```

## Dynamic Prompt Building

Build prompts dynamically based on context.

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

Version prompts for tracking and A/B testing.

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

// Load prompts from YAML file
func LoadPromptsFromYAML(filename string) (*PromptRegistry, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    var prompts []VersionedPrompt
    if err := yaml.Unmarshal(data, &prompts); err != nil {
        return nil, err
    }

    registry := NewPromptRegistry()
    for _, prompt := range prompts {
        registry.Register(prompt)
    }

    return registry, nil
}
```

Example `prompts.yaml`:

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

## Prompt Caching Optimization

Order prompts to maximize cache hits and reduce costs.

```go
// Optimal prompt structure for caching:
// 1. Static system instructions (cached)
// 2. Static examples (cached)
// 3. Dynamic context (partially cached)
// 4. User query (not cached)

func BuildCacheOptimizedPrompt(staticContext, dynamicContext, userQuery string) []schema.Message {
    return []schema.Message{
        // Static content first - fully cacheable
        schema.NewSystemMessage(`You are an expert assistant.

Guidelines:
- Be precise and helpful
- Cite sources when possible
- Format code in markdown blocks`),

        // Few-shot examples - static and cacheable
        schema.NewHumanMessage("What is Go's error handling pattern?"),
        schema.NewAIMessage("Go uses explicit error returns: if err != nil { return err }"),

        // Dynamic but slowly changing context - partially cacheable
        schema.NewSystemMessage(fmt.Sprintf("Current context:\n%s", dynamicContext)),

        // User query - never cached
        schema.NewHumanMessage(userQuery),
    }
}
```

## A/B Testing Prompts

Compare prompt versions to optimize quality.

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

1. **Separate prompts from code** using templates and registries
2. **Version all prompts** to track behavior changes
3. **Test systematically** with diverse inputs
4. **Monitor metrics** (latency, tokens, success rate)
5. **Use static content first** for cache optimization
6. **Validate inputs** before template substitution
7. **Document prompt intent** and design decisions
8. **Implement gradual rollouts** for prompt changes
9. **Track prompt lineage** for debugging
10. **Store prompts in version control** or a database

## Next Steps

Now that you understand prompt engineering:
- Learn about [Structured Output](/guides/structured-output) for typed responses
- Explore [Multi-Agent Systems](/guides/multi-agent-systems) for persona coordination
- Read [RAG Pipeline](/guides/rag-pipeline) for context-aware prompts
- Check out [Prompt Recipes](/cookbook/prompt-recipes) for advanced patterns
