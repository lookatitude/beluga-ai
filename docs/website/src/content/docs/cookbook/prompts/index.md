---
title: Prompt Recipes
description: Battle-tested prompt patterns for dynamic templates, few-shot learning, and context management.
sidebar:
  order: 0
---

Prompt engineering in LLM applications is more than writing good instructions. It involves structuring message sequences, managing template variables, selecting relevant examples, optimizing for provider caching, and adapting prompts to runtime conditions. These recipes provide production patterns for each of these concerns, with implementations that handle the edge cases that emerge at scale: missing variables, unbounded example lists, cache invalidation, and conditional logic.

Each recipe is self-contained with full imports and error handling, designed to be composed with Beluga AI's `schema.Message` types and LLM interfaces.

## Dynamic Message Chain Templates

LLM APIs accept message arrays, not single strings, and the composition of that array determines the model's behavior. A static template works for simple cases, but production applications need to build message chains dynamically: adding few-shot examples based on input similarity, injecting retrieved documents as context, and conditioning system instructions on user attributes. The `MessageChainBuilder` pattern provides a fluent API for constructing these chains programmatically, with the chain builder pattern ensuring messages are added in the correct order (system first, then examples, then context, then the user query).

```go
package main

import (
    "context"
    "fmt"
    "text/template"
    "bytes"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/llm"
)

type MessageChainBuilder struct {
    messages []schema.Message
}

func NewMessageChainBuilder() *MessageChainBuilder {
    return &MessageChainBuilder{
        messages: make([]schema.Message, 0),
    }
}

func (mcb *MessageChainBuilder) AddSystem(template string, data interface{}) *MessageChainBuilder {
    content := renderTemplate(template, data)
    mcb.messages = append(mcb.messages, schema.NewSystemMessage(content))
    return mcb
}

func (mcb *MessageChainBuilder) AddHuman(template string, data interface{}) *MessageChainBuilder {
    content := renderTemplate(template, data)
    mcb.messages = append(mcb.messages, schema.NewHumanMessage(content))
    return mcb
}

func (mcb *MessageChainBuilder) AddAI(template string, data interface{}) *MessageChainBuilder {
    content := renderTemplate(template, data)
    mcb.messages = append(mcb.messages, schema.NewAIMessage(content))
    return mcb
}

func (mcb *MessageChainBuilder) AddFewShot(examples []FewShotExample) *MessageChainBuilder {
    for _, ex := range examples {
        mcb.messages = append(mcb.messages,
            schema.NewHumanMessage(ex.Input),
            schema.NewAIMessage(ex.Output),
        )
    }
    return mcb
}

func (mcb *MessageChainBuilder) AddContext(documents []string, prefix string) *MessageChainBuilder {
    var contextBuf bytes.Buffer
    contextBuf.WriteString(prefix + "\n\n")

    for i, doc := range documents {
        contextBuf.WriteString(fmt.Sprintf("[Document %d]\n%s\n\n", i+1, doc))
    }

    mcb.messages = append(mcb.messages, schema.NewSystemMessage(contextBuf.String()))
    return mcb
}

func (mcb *MessageChainBuilder) Build() []schema.Message {
    return mcb.messages
}

type FewShotExample struct {
    Input  string
    Output string
}

func renderTemplate(tmpl string, data interface{}) string {
    t, err := template.New("msg").Parse(tmpl)
    if err != nil {
        return tmpl // Fallback to raw template
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        return tmpl
    }

    return buf.String()
}

// Usage example: SQL query generator
func GenerateSQLQuery(ctx context.Context, model llm.ChatModel, userQuery string, dbSchema string) (string, error) {
    builder := NewMessageChainBuilder().
        AddSystem(`You are an expert SQL developer.
Database schema:
{{.Schema}}

Rules:
- Use PostgreSQL syntax
- Include comments
- Use proper indentation
- Handle edge cases`, map[string]interface{}{
            "Schema": dbSchema,
        }).
        AddFewShot([]FewShotExample{
            {
                Input: "Get all users who signed up last month",
                Output: `-- Get users from previous month
SELECT *
FROM users
WHERE created_at >= DATE_TRUNC('month', NOW() - INTERVAL '1 month')
  AND created_at < DATE_TRUNC('month', NOW());`,
            },
            {
                Input: "Count orders by status",
                Output: `-- Order counts grouped by status
SELECT
    status,
    COUNT(*) as order_count
FROM orders
GROUP BY status
ORDER BY order_count DESC;`,
            },
        }).
        AddHuman("{{.Query}}", map[string]interface{}{
            "Query": userQuery,
        })

    messages := builder.Build()

    response, err := model.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.GetContent(), nil
}
```

## Partial Variable Substitution

In many applications, prompt variables become available at different times: the user role is known at session start, the retrieved documents arrive after a search, and the user query comes last. Rather than blocking until all variables are available, partial substitution lets you fill in variables incrementally and inspect the prompt's current state at any point. This is particularly useful for debugging (you can see exactly what the prompt looks like with only some variables filled) and for streaming scenarios where data arrives progressively.

```go
type PartialTemplate struct {
    template string
    variables map[string]interface{}
}

func NewPartialTemplate(tmpl string) *PartialTemplate {
    return &PartialTemplate{
        template:  tmpl,
        variables: make(map[string]interface{}),
    }
}

func (pt *PartialTemplate) Set(key string, value interface{}) *PartialTemplate {
    pt.variables[key] = value
    return pt
}

func (pt *PartialTemplate) SetMultiple(vars map[string]interface{}) *PartialTemplate {
    for k, v := range vars {
        pt.variables[k] = v
    }
    return pt
}

func (pt *PartialTemplate) Render() (string, error) {
    tmpl, err := template.New("partial").
        Option("missingkey=zero"). // Don't error on missing keys
        Parse(pt.template)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, pt.variables); err != nil {
        return "", err
    }

    return buf.String(), nil
}

func (pt *PartialTemplate) GetMissingVariables() []string {
    tmpl, err := template.New("check").Parse(pt.template)
    if err != nil {
        return nil
    }

    // Extract all variable names from template
    var missing []string
    for _, node := range tmpl.Root.Nodes {
        // This is simplified - in production use proper AST parsing
        varName := extractVarName(node.String())
        if varName != "" {
            if _, exists := pt.variables[varName]; !exists {
                missing = append(missing, varName)
            }
        }
    }

    return missing
}

// Usage: Progressive prompt building
func BuildProgressivePrompt(ctx context.Context) (string, error) {
    prompt := NewPartialTemplate(`You are a {{.Role}} helping with {{.Task}}.

Context:
{{.Context}}

Requirements:
{{range .Requirements}}- {{.}}
{{end}}

Question: {{.Question}}`)

    // Set variables progressively
    prompt.Set("Role", "senior developer")
    prompt.Set("Task", "code review")

    // Check what's still needed
    missing := prompt.GetMissingVariables()
    fmt.Printf("Still need: %v\n", missing) // [Context, Requirements, Question]

    // Add more variables as they become available
    prompt.Set("Context", "Production API handling 10K RPS")
    prompt.SetMultiple(map[string]interface{}{
        "Requirements": []string{
            "Check for security issues",
            "Verify error handling",
            "Look for performance problems",
        },
        "Question": "Review this authentication middleware",
    })

    return prompt.Render()
}
```

## Few-Shot Learning with Dynamic Examples

Static few-shot examples work when your use case is narrow, but they break down when the input domain is broad. A SQL generation prompt with examples about date queries won't help much when the user asks about aggregations. Dynamic example selection solves this by embedding both the user query and example inputs into vector space, then selecting the most similar examples. This ensures the model sees examples that are structurally similar to the current task, improving output quality without increasing the total number of examples (and therefore tokens) in the prompt.

```go
import (
    "github.com/lookatitude/beluga-ai/rag/embedding"
)

type ExampleSelector struct {
    examples  []FewShotExample
    embedder  embedding.Embedder
    cache     map[string][]float32
}

func NewExampleSelector(embedder embedding.Embedder) *ExampleSelector {
    return &ExampleSelector{
        examples: make([]FewShotExample, 0),
        embedder: embedder,
        cache:    make(map[string][]float32),
    }
}

func (es *ExampleSelector) AddExample(input, output string) {
    es.examples = append(es.examples, FewShotExample{
        Input:  input,
        Output: output,
    })
}

func (es *ExampleSelector) SelectRelevant(ctx context.Context, query string, k int) ([]FewShotExample, error) {
    // Embed query
    queryEmbed, err := es.embedder.EmbedText(ctx, query)
    if err != nil {
        return nil, err
    }

    // Embed all examples (with caching)
    type scoredExample struct {
        example FewShotExample
        score   float64
    }

    scored := make([]scoredExample, 0, len(es.examples))

    for _, ex := range es.examples {
        // Check cache
        exEmbed, ok := es.cache[ex.Input]
        if !ok {
            exEmbed, err = es.embedder.EmbedText(ctx, ex.Input)
            if err != nil {
                return nil, err
            }
            es.cache[ex.Input] = exEmbed
        }

        // Calculate similarity
        similarity := cosineSimilarity(queryEmbed, exEmbed)
        scored = append(scored, scoredExample{
            example: ex,
            score:   similarity,
        })
    }

    // Sort by similarity
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    // Return top k
    if k > len(scored) {
        k = len(scored)
    }

    result := make([]FewShotExample, k)
    for i := 0; i < k; i++ {
        result[i] = scored[i].example
    }

    return result, nil
}

func cosineSimilarity(a, b []float32) float64 {
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i] * b[i])
        normA += float64(a[i] * a[i])
        normB += float64(b[i] * b[i])
    }
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Usage: Dynamic SQL query generation
func GenerateSQLWithDynamicExamples(
    ctx context.Context,
    model llm.ChatModel,
    embedder embedding.Embedder,
    query string,
) (string, error) {
    selector := NewExampleSelector(embedder)

    // Add example library
    selector.AddExample(
        "Get users who haven't logged in for 30 days",
        "SELECT * FROM users WHERE last_login < NOW() - INTERVAL '30 days'",
    )
    selector.AddExample(
        "Count orders by month",
        "SELECT DATE_TRUNC('month', created_at), COUNT(*) FROM orders GROUP BY 1",
    )
    selector.AddExample(
        "Find top customers by revenue",
        "SELECT customer_id, SUM(total) FROM orders GROUP BY customer_id ORDER BY SUM(total) DESC LIMIT 10",
    )

    // Select 2 most relevant examples
    examples, err := selector.SelectRelevant(ctx, query, 2)
    if err != nil {
        return "", err
    }

    // Build prompt with relevant examples
    builder := NewMessageChainBuilder().
        AddSystem("You are an SQL expert. Generate PostgreSQL queries based on natural language.").
        AddFewShot(examples).
        AddHuman(query)

    messages := builder.Build()

    response, err := model.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.GetContent(), nil
}
```

## Prompt Caching Optimization

LLM providers like Anthropic and OpenAI offer prompt caching, where repeated prefix content is served from cache at reduced cost and latency. The key to maximizing cache hits is message ordering: static content (system instructions, tool definitions) must come first, semi-dynamic content (retrieved documents, session context) in the middle, and fully dynamic content (the user query) last. If you interleave static and dynamic content, the cache breaks at the first dynamic token and everything after it is recomputed. The `CachedPromptBuilder` enforces this ordering structurally, making cache-optimal prompts the default rather than requiring discipline from every caller.

```go
type CachedPromptBuilder struct {
    staticPrefix   string // Fully cacheable
    dynamicPrefix  string // Partially cacheable
    userQuery      string // Never cached
}

func NewCachedPromptBuilder() *CachedPromptBuilder {
    return &CachedPromptBuilder{}
}

func (cpb *CachedPromptBuilder) WithStaticPrefix(instructions string) *CachedPromptBuilder {
    cpb.staticPrefix = instructions
    return cpb
}

func (cpb *CachedPromptBuilder) WithDynamicPrefix(context string) *CachedPromptBuilder {
    cpb.dynamicPrefix = context
    return cpb
}

func (cpb *CachedPromptBuilder) WithQuery(query string) *CachedPromptBuilder {
    cpb.userQuery = query
    return cpb
}

func (cpb *CachedPromptBuilder) Build() []schema.Message {
    messages := make([]schema.Message, 0, 3)

    // Static content first - fully cacheable
    if cpb.staticPrefix != "" {
        messages = append(messages, schema.NewSystemMessage(cpb.staticPrefix))
    }

    // Dynamic but slowly changing - partially cacheable
    if cpb.dynamicPrefix != "" {
        messages = append(messages, schema.NewSystemMessage(cpb.dynamicPrefix))
    }

    // User query - never cached
    if cpb.userQuery != "" {
        messages = append(messages, schema.NewHumanMessage(cpb.userQuery))
    }

    return messages
}

// Usage: RAG with optimal caching
func RAGWithCaching(
    ctx context.Context,
    model llm.ChatModel,
    systemInstructions string, // Static - cache forever
    retrievedDocs []string,    // Dynamic - cache for session
    userQuery string,          // Dynamic - no cache
) (string, error) {
    // Build context string
    var contextBuf bytes.Buffer
    contextBuf.WriteString("Relevant documents:\n\n")
    for i, doc := range retrievedDocs {
        contextBuf.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, doc))
    }

    // Build optimized prompt
    builder := NewCachedPromptBuilder().
        WithStaticPrefix(systemInstructions).
        WithDynamicPrefix(contextBuf.String()).
        WithQuery(userQuery)

    messages := builder.Build()

    response, err := model.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.GetContent(), nil
}
```

## Conditional Prompt Sections

Production prompts often need to adapt to runtime conditions: premium customers get different instructions than free-tier users, peak hours call for concise responses, and active support issues require empathy language. Hardcoding these variations as separate prompt templates leads to a combinatorial explosion (2 tiers x 2 issue states x 2 time states = 8 templates). The `ConditionalPromptBuilder` avoids this by treating each variation as an independent section with a condition function and a priority. Sections are included only when their condition is true and ordered by priority, composing the final prompt dynamically from its constituent parts.

```go
type ConditionalPromptBuilder struct {
    sections []PromptSection
}

type PromptSection struct {
    Content   string
    Condition func() bool
    Priority  int // Lower = earlier in prompt
}

func NewConditionalPromptBuilder() *ConditionalPromptBuilder {
    return &ConditionalPromptBuilder{
        sections: make([]PromptSection, 0),
    }
}

func (cpb *ConditionalPromptBuilder) AddSection(content string, priority int, condition func() bool) *ConditionalPromptBuilder {
    cpb.sections = append(cpb.sections, PromptSection{
        Content:   content,
        Condition: condition,
        Priority:  priority,
    })
    return cpb
}

func (cpb *ConditionalPromptBuilder) Build() string {
    // Filter sections by condition
    active := make([]PromptSection, 0)
    for _, section := range cpb.sections {
        if section.Condition() {
            active = append(active, section)
        }
    }

    // Sort by priority
    sort.Slice(active, func(i, j int) bool {
        return active[i].Priority < active[j].Priority
    })

    // Combine sections
    var result strings.Builder
    for i, section := range active {
        result.WriteString(section.Content)
        if i < len(active)-1 {
            result.WriteString("\n\n")
        }
    }

    return result.String()
}

// Usage: Adaptive customer support
func AdaptiveSupport(
    ctx context.Context,
    model llm.ChatModel,
    customerTier string,
    hasActiveIssue bool,
    isPeakHours bool,
) (string, error) {
    builder := NewConditionalPromptBuilder()

    // Base instructions (always included)
    builder.AddSection(
        "You are a customer support agent. Be helpful and professional.",
        0,
        func() bool { return true },
    )

    // Premium customer handling
    builder.AddSection(
        "This is a PREMIUM customer. Provide white-glove service and expedited solutions.",
        1,
        func() bool { return customerTier == "premium" },
    )

    // Active issue handling
    builder.AddSection(
        "Customer has an active critical issue. Prioritize resolution and show empathy.",
        2,
        func() bool { return hasActiveIssue },
    )

    // Peak hours handling
    builder.AddSection(
        "Peak hours: Keep responses concise. Offer callback options for complex issues.",
        3,
        func() bool { return isPeakHours },
    )

    systemPrompt := builder.Build()

    messages := []schema.Message{
        schema.NewSystemMessage(systemPrompt),
        schema.NewHumanMessage("How do I reset my password?"),
    }

    response, err := model.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.GetContent(), nil
}
```

## Prompt Template Library

As your application grows, prompts accumulate across features, teams, and deployment environments. Without a central registry, prompts are duplicated, versioning happens informally (if at all), and A/B testing requires code changes. The `PromptLibrary` provides a registry pattern (following Beluga AI's `Register()` + `Get()` convention) with version support, variable validation, and template rendering. Versioning is explicit (e.g., "code_review@2.0"), making it safe to evolve prompts without breaking existing consumers that pin to a specific version.

```go
type PromptTemplate struct {
    Name        string
    Version     string
    Template    string
    Variables   []string
    Description string
    Examples    []map[string]string
}

type PromptLibrary struct {
    templates map[string]map[string]PromptTemplate // name -> version -> template
    mu        sync.RWMutex
}

func NewPromptLibrary() *PromptLibrary {
    return &PromptLibrary{
        templates: make(map[string]map[string]PromptTemplate),
    }
}

func (pl *PromptLibrary) Register(tmpl PromptTemplate) {
    pl.mu.Lock()
    defer pl.mu.Unlock()

    if _, ok := pl.templates[tmpl.Name]; !ok {
        pl.templates[tmpl.Name] = make(map[string]PromptTemplate)
    }

    pl.templates[tmpl.Name][tmpl.Version] = tmpl
}

func (pl *PromptLibrary) Get(name, version string) (PromptTemplate, error) {
    pl.mu.RLock()
    defer pl.mu.RUnlock()

    versions, ok := pl.templates[name]
    if !ok {
        return PromptTemplate{}, fmt.Errorf("template not found: %s", name)
    }

    // Get specific version or latest
    if version == "latest" || version == "" {
        var latest PromptTemplate
        var latestVer string

        for v, tmpl := range versions {
            if v > latestVer {
                latest = tmpl
                latestVer = v
            }
        }

        if latestVer == "" {
            return PromptTemplate{}, fmt.Errorf("no versions found for: %s", name)
        }

        return latest, nil
    }

    tmpl, ok := versions[version]
    if !ok {
        return PromptTemplate{}, fmt.Errorf("version not found: %s@%s", name, version)
    }

    return tmpl, nil
}

func (pl *PromptLibrary) Render(name, version string, vars map[string]interface{}) (string, error) {
    tmpl, err := pl.Get(name, version)
    if err != nil {
        return "", err
    }

    // Check required variables
    for _, varName := range tmpl.Variables {
        if _, ok := vars[varName]; !ok {
            return "", fmt.Errorf("missing required variable: %s", varName)
        }
    }

    // Render template
    t, err := template.New(tmpl.Name).Parse(tmpl.Template)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, vars); err != nil {
        return "", err
    }

    return buf.String(), nil
}

// Setup library
func SetupPromptLibrary() *PromptLibrary {
    lib := NewPromptLibrary()

    // Register templates
    lib.Register(PromptTemplate{
        Name:    "code_review",
        Version: "1.0",
        Template: `Review this {{.Language}} code for:
- Security issues
- Performance problems
- Best practices

Code:
{{.Code}}`,
        Variables: []string{"Language", "Code"},
        Description: "General code review template",
    })

    lib.Register(PromptTemplate{
        Name:    "code_review",
        Version: "2.0",
        Template: `You are an expert {{.Language}} developer.

Review this code focusing on:
{{range .FocusAreas}}- {{.}}
{{end}}

Code:
{{.Code}}

Provide:
1. Summary
2. Issues (with line numbers)
3. Recommendations`,
        Variables: []string{"Language", "Code", "FocusAreas"},
        Description: "Enhanced code review with focus areas",
    })

    return lib
}
```

## Best Practices

When engineering prompts:

1. **Structure for caching** - static content first, dynamic last
2. **Use templates** - separate logic from data
3. **Version prompts** - track changes and A/B test
4. **Select examples dynamically** - based on input similarity
5. **Validate variables** - check required fields before rendering
6. **Cache template renders** - avoid re-parsing
7. **Build conditionally** - include sections based on context
8. **Document templates** - describe purpose and variables
9. **Test with edge cases** - empty inputs, missing variables
10. **Monitor performance** - track cache hit rates and costs

## Next Steps

- Learn about [Prompt Engineering](/guides/foundations/prompt-engineering) for production patterns
- Explore [LLM Recipes](/cookbook/llm) for advanced generation
- Read [Agent Recipes](/cookbook/agents) for agentic prompts
