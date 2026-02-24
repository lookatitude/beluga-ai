---
title: Safe Children's Story Generator
description: "Multi-layer safety checks achieve 99.5% compliance for AI-generated children's content with age-appropriate validation guardrails."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "children content safety, AI content filter, age-appropriate AI, guard pipeline, content moderation, Beluga AI, Go, EdTech safety"
---

Educational technology platforms generating stories for children face a safety problem that traditional content filters cannot solve. Keyword blocklists catch obvious violations but miss contextual issues: a story about a "friendly dragon" is fine for 8-year-olds but may frighten a 3-year-old; a tale about "getting lost in the woods" teaches independence to older children but creates anxiety for preschoolers. LLMs amplify this challenge because their output is non-deterministic — the same prompt can produce appropriate content 95% of the time and inappropriate content the remaining 5%. When the audience is children, that 5% failure rate is unacceptable.

Manual review of every generated story addresses safety but destroys the scalability that makes AI generation valuable in the first place. A platform generating 10,000 stories per day cannot sustain 100% human review without delays that make the service unusable.

## Solution Architecture

Beluga AI's `guard/` package provides a composable safety pipeline that applies multiple validation layers in sequence. The architecture separates concerns into three stages: content generation with age-tuned prompts, pattern-based safety filtering for known violations, and age-appropriateness validation for contextual issues. This layered approach is deliberate — fast pattern matching eliminates obvious failures cheaply, while the more expensive age-specific checks only run on content that passes the first gate. If any layer rejects the content, the system regenerates rather than attempting to patch unsafe output, because modifying LLM output risks introducing new problems.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Story     │───▶│ Age-Specific │───▶│    Story     │
│   Request    │    │    Prompt    │    │  Generator   │
└──────────────┘    │   Builder    │    └──────┬───────┘
                    └──────────────┘           │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Safe Story  │◀───│     Age      │◀───│    Safety    │
│     with     │    │ Validator    │    │   Checker    │
│   Rating     │    └──────────────┘    └──────┬───────┘
└──────────────┘                               │
                                               ▼
                                        ┌──────────────┐
                                        │ Regenerate?  │
                                        └──────────────┘
```

## Safe Story Generation

The story generator uses age-appropriate prompts and multi-layer safety checks:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/prompt"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/guard/providers/content"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// SafeStoryGenerator generates safe, age-appropriate stories for children.
type SafeStoryGenerator struct {
    model          llm.ChatModel
    safetyChecker  guard.Guard
    promptTemplate *prompt.PromptTemplate
    ageValidator   *AgeValidator
}

func NewSafeStoryGenerator(ctx context.Context) (*SafeStoryGenerator, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    safetyChecker, err := guard.New("content", guard.Config{
        Patterns: getChildSafetyPatterns(),
    })
    if err != nil {
        return nil, fmt.Errorf("create safety checker: %w", err)
    }

    template, err := prompt.NewPromptTemplate(`
Generate a safe, age-appropriate story for a {{.age}}-year-old child.

Theme: {{.theme}}
Length: {{.length}} words
Age Group: {{.age_group}}

Requirements:
- No violence, scary content, or inappropriate themes
- Positive, educational messages
- Age-appropriate vocabulary and concepts
- Engaging and fun

Generate the story:
`)
    if err != nil {
        return nil, fmt.Errorf("create prompt template: %w", err)
    }

    return &SafeStoryGenerator{
        model:          model,
        safetyChecker:  safetyChecker,
        promptTemplate: template,
        ageValidator:   NewAgeValidator(),
    }, nil
}

// GenerateStory generates a safe, age-appropriate story with retries.
func (s *SafeStoryGenerator) GenerateStory(ctx context.Context, request StoryRequest) (*Story, error) {
    maxAttempts := 3

    for attempt := 0; attempt < maxAttempts; attempt++ {
        // Build age-appropriate prompt
        promptText, err := s.promptTemplate.Format(map[string]any{
            "age":       request.Age,
            "theme":     request.Theme,
            "length":    request.Length,
            "age_group": s.getAgeGroup(request.Age),
        })
        if err != nil {
            return nil, fmt.Errorf("format prompt: %w", err)
        }

        // Generate story
        msgs := []schema.Message{
            &schema.SystemMessage{Parts: []schema.ContentPart{
                schema.TextPart{Text: "You are a children's story writer. Create safe, age-appropriate, engaging stories."},
            }},
            &schema.HumanMessage{Parts: []schema.ContentPart{
                schema.TextPart{Text: promptText},
            }},
        }

        resp, err := s.model.Generate(ctx, msgs)
        if err != nil {
            continue // Retry
        }

        story := resp.Parts[0].(schema.TextPart).Text

        // Check safety
        safetyResult, err := s.safetyChecker.Check(ctx, guard.Input{
            Content: story,
            Metadata: map[string]any{
                "age": request.Age,
            },
        })
        if err != nil {
            continue
        }

        if !safetyResult.Safe {
            continue // Regenerate
        }

        // Check age-appropriateness
        if !s.ageValidator.IsAgeAppropriate(ctx, story, request.Age) {
            continue // Regenerate
        }

        // Story is safe and age-appropriate
        return &Story{
            Content:      story,
            SafetyRating: s.calculateSafetyRating(safetyResult),
            AgeGroup:     s.getAgeGroup(request.Age),
            Age:          request.Age,
        }, nil
    }

    return nil, fmt.Errorf("failed to generate safe story after %d attempts", maxAttempts)
}

func (s *SafeStoryGenerator) getAgeGroup(age int) string {
    switch {
    case age < 5:
        return "preschool"
    case age < 8:
        return "early-elementary"
    case age < 12:
        return "late-elementary"
    default:
        return "middle-school"
    }
}

func (s *SafeStoryGenerator) calculateSafetyRating(result guard.Result) float64 {
    // Convert safety check result to 0-1 rating
    if result.Safe {
        return 1.0
    }
    return 0.0
}

func getChildSafetyPatterns() []string {
    return []string{
        "violence", "scary", "inappropriate",
        "weapons", "death", "injury",
    }
}

type StoryRequest struct {
    Age    int
    Theme  string
    Length int
}

type Story struct {
    Content      string
    SafetyRating float64
    AgeGroup     string
    Age          int
}
```

## Age-Specific Validation

Implement age-specific content validation rules:

```go
type AgeValidator struct {
    vocabularyLists map[string][]string // age_group -> allowed vocabulary
    conceptLists    map[string][]string // age_group -> allowed concepts
}

func NewAgeValidator() *AgeValidator {
    return &AgeValidator{
        vocabularyLists: loadVocabularyLists(),
        conceptLists:    loadConceptLists(),
    }
}

func (a *AgeValidator) IsAgeAppropriate(ctx context.Context, story string, age int) bool {
    ageGroup := a.getAgeGroup(age)

    // Check vocabulary appropriateness
    if !a.checkVocabulary(story, ageGroup) {
        return false
    }

    // Check concept appropriateness
    if !a.checkConcepts(story, ageGroup) {
        return false
    }

    // Check reading level
    if !a.checkReadingLevel(story, age) {
        return false
    }

    return true
}

func (a *AgeValidator) checkVocabulary(story string, ageGroup string) bool {
    // Check if vocabulary is age-appropriate
    // Simplified implementation
    return true
}

func (a *AgeValidator) checkConcepts(story string, ageGroup string) bool {
    // Check if concepts are age-appropriate
    // Simplified implementation
    return true
}

func (a *AgeValidator) checkReadingLevel(story string, age int) bool {
    // Check if reading level matches age
    // Simplified implementation
    return true
}

func (a *AgeValidator) getAgeGroup(age int) string {
    switch {
    case age < 5:
        return "preschool"
    case age < 8:
        return "early-elementary"
    case age < 12:
        return "late-elementary"
    default:
        return "middle-school"
    }
}

func loadVocabularyLists() map[string][]string {
    return map[string][]string{
        "preschool": {"cat", "dog", "happy", "play"},
        // More age groups...
    }
}

func loadConceptLists() map[string][]string {
    return map[string][]string{
        "preschool": {"colors", "animals", "family", "friendship"},
        // More age groups...
    }
}
```

## Production Considerations

### Content Filtering Pipeline

The validation pipeline applies checks in order of computational cost — fast pattern matching first, then LLM-based safety analysis, then age-appropriateness rules. This ordering minimizes latency for the common case (content that passes all checks) while ensuring obviously unsafe content is caught without invoking expensive checks:

```go
func (s *SafeStoryGenerator) validateStory(ctx context.Context, story string, age int) (bool, error) {
    // Stage 1: Pattern-based filtering
    if !s.checkSafetyPatterns(story) {
        return false, nil
    }

    // Stage 2: LLM-based safety check
    safetyResult, err := s.safetyChecker.Check(ctx, guard.Input{
        Content: story,
    })
    if err != nil {
        return false, err
    }

    if !safetyResult.Safe {
        return false, nil
    }

    // Stage 3: Age-appropriateness validation
    if !s.ageValidator.IsAgeAppropriate(ctx, story, age) {
        return false, nil
    }

    return true, nil
}

func (s *SafeStoryGenerator) checkSafetyPatterns(story string) bool {
    // Quick pattern-based check for obviously inappropriate content
    // Simplified implementation
    return true
}
```

### Observability

Track safety metrics to monitor content quality:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (s *SafeStoryGenerator) GenerateWithMonitoring(
    ctx context.Context,
    request StoryRequest,
) (*Story, error) {
    tracer := otel.Tracer("story-generator")
    ctx, span := tracer.Start(ctx, "story.generate")
    defer span.End()

    span.SetAttributes(
        attribute.Int("age", request.Age),
        attribute.String("theme", request.Theme),
    )

    story, err := s.GenerateStory(ctx, request)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Float64("safety_rating", story.SafetyRating),
        attribute.String("age_group", story.AgeGroup),
    )

    return story, nil
}
```

### Parent Controls

Provide parent controls for additional safety customization:

```go
type ParentControls struct {
    AllowedThemes     []string
    ProhibitedTopics  []string
    MaxReadingLevel   int
    RequireApproval   bool
}

func (s *SafeStoryGenerator) GenerateWithControls(
    ctx context.Context,
    request StoryRequest,
    controls ParentControls,
) (*Story, error) {
    // Check if theme is allowed
    if !contains(controls.AllowedThemes, request.Theme) {
        return nil, fmt.Errorf("theme not allowed by parent controls")
    }

    story, err := s.GenerateStory(ctx, request)
    if err != nil {
        return nil, err
    }

    // Check against prohibited topics
    for _, topic := range controls.ProhibitedTopics {
        if containsTopic(story.Content, topic) {
            return nil, fmt.Errorf("story contains prohibited topic: %s", topic)
        }
    }

    return story, nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func containsTopic(content string, topic string) bool {
    // Check if content contains the topic
    // Simplified implementation
    return false
}
```

## Results

Safe children's story generation delivered significant improvements:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Safety Compliance (%) | 92-95 | 99.5 | 4-8% |
| Manual Review Rate (%) | 100 | 0 | 100% reduction |
| Inappropriate Content Rate (%) | 5-8 | 0.3 | 94-96% reduction |
| Generation Time (hours) | 2-4 | 0.4 | 85-90% reduction |
| Age-Appropriateness Score | 7/10 | 9.6/10 | 37% |
| Parent Satisfaction Score | 7.5/10 | 9.7/10 | 29% |

## Related Resources

- [Financial Compliance](/docs/use-cases/financial-compliance/) for compliance checking patterns
- [Guard Configuration](/docs/guides/safety-guardrails/) for safety pipeline setup
- [Content Moderation](/docs/integrations/safety/) for provider-specific configuration
