# LangChain Hub Prompt Loading

Welcome, colleague! In this integration guide, we're going to integrate LangChain Hub for loading prompt templates with Beluga AI's prompts package. LangChain Hub provides a repository of community-contributed prompts that you can use in your applications.

## What you will build

You will create a system that loads prompt templates from LangChain Hub and uses them with Beluga AI, enabling you to leverage community prompts and share your own prompts.

## Learning Objectives

- ✅ Load prompts from LangChain Hub
- ✅ Convert LangChain prompts to Beluga AI format
- ✅ Use LangChain Hub prompts in your applications
- ✅ Understand prompt format conversion

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- LangChain Hub account (optional, for private prompts)
- Understanding of prompt templates

## Step 1: Setup and Installation

Install HTTP client for LangChain Hub:
bash
```bash
go get net/http
```

LangChain Hub API endpoint: `https://api.hub.langchain.com`

## Step 2: Load Prompt from LangChain Hub

Create a LangChain Hub loader:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "go.opentelemetry.io/otel"
)

type LangChainHubPrompt struct {
    Messages []struct {
        Role    string `json:"role"`
        Content string `json:"content"`
    } `json:"messages"`
    InputVariables []string `json:"input_variables"`
}

type LangChainHubLoader struct {
    baseURL    string
    httpClient *http.Client
}

func NewLangChainHubLoader() *LangChainHubLoader {
    return &LangChainHubLoader{
        baseURL: "https://api.hub.langchain.com",
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (l *LangChainHubLoader) LoadPrompt(ctx context.Context, owner, repo, promptID string) (*LangChainHubPrompt, error) {
    url := fmt.Sprintf("%s/repos/%s/%s/prompts/%s", l.baseURL, owner, repo, promptID)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    resp, err := l.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch prompt: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    var prompt LangChainHubPrompt
    if err := json.Unmarshal(body, &prompt); err != nil {
        return nil, fmt.Errorf("failed to parse prompt: %w", err)
    }

    
    return &prompt, nil
}
```

## Step 3: Convert to Beluga AI Format

Convert LangChain Hub prompt to Beluga AI template:
```go
func (l *LangChainHubLoader) ConvertToBelugaTemplate(ctx context.Context, hubPrompt *LangChainHubPrompt) (prompts.PromptTemplate, error) {
    // Create manager
    manager, err := prompts.NewPromptManager()
    if err != nil {
        return nil, fmt.Errorf("failed to create manager: %w", err)
    }
    
    // Convert messages to template
    var templateStr string
    for _, msg := range hubPrompt.Messages {
        templateStr += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
    }
    
    // Create Beluga AI template
    template, err := manager.NewStringTemplate(templateStr, hubPrompt.InputVariables)
    if err != nil {
        return nil, fmt.Errorf("failed to create template: %w", err)
    }

    
    return template, nil
}
```

## Step 4: Use Loaded Prompt

Use the loaded prompt in your application:
```go
func main() {
    ctx := context.Background()
    
    // Load prompt from LangChain Hub
    loader := NewLangChainHubLoader()
    hubPrompt, err := loader.LoadPrompt(ctx, "langchain-ai", "prompt-hub", "prompts/rag")
    if err != nil {
        log.Fatalf("Failed to load prompt: %v", err)
    }
    
    // Convert to Beluga AI format
    template, err := loader.ConvertToBelugaTemplate(ctx, hubPrompt)
    if err != nil {
        log.Fatalf("Failed to convert prompt: %v", err)
    }
    
    // Use template
    result, err := template.Format(ctx, map[string]any{
        "context": "Machine learning is...",
        "question": "What is ML?",
    })
    if err != nil {
        log.Fatalf("Failed to format: %v", err)
    }

    
    fmt.Printf("Formatted prompt: %s\n", result.ToString())
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type LangChainHubLoader struct {
    baseURL    string
    httpClient *http.Client
    tracer     trace.Tracer
}

func NewLangChainHubLoader() *LangChainHubLoader {
    return &LangChainHubLoader{
        baseURL: "https://api.hub.langchain.com",
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        tracer: otel.Tracer("beluga.prompts.langchain_hub"),
    }
}

func (l *LangChainHubLoader) LoadAndConvert(ctx context.Context, owner, repo, promptID string) (prompts.PromptTemplate, error) {
    ctx, span := l.tracer.Start(ctx, "langchain_hub.load",
        trace.WithAttributes(
            attribute.String("owner", owner),
            attribute.String("repo", repo),
            attribute.String("prompt_id", promptID),
        ),
    )
    defer span.End()
    
    // Load from Hub
    url := fmt.Sprintf("%s/repos/%s/%s/prompts/%s", l.baseURL, owner, repo, promptID)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    resp, err := l.httpClient.Do(req)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to fetch: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
        span.RecordError(err)
        return nil, err
    }
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to read: %w", err)
    }
    
    var hubPrompt struct {
        Messages       []map[string]interface{} `json:"messages"`
        InputVariables []string                  `json:"input_variables"`
    }
    if err := json.Unmarshal(body, &hubPrompt); err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to parse: %w", err)
    }
    
    // Convert to Beluga AI template
    manager, err := prompts.NewPromptManager()
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to create manager: %w", err)
    }
    
    // Build template string
    templateStr := ""
    for _, msg := range hubPrompt.Messages {
        if content, ok := msg["content"].(string); ok {
            templateStr += content + "\n"
        }
    }
    
    template, err := manager.NewStringTemplate(templateStr, hubPrompt.InputVariables)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to create template: %w", err)
    }
    
    span.SetAttributes(attribute.Int("input_variables", len(hubPrompt.InputVariables)))
    return template, nil
}

func main() {
    ctx := context.Background()
    
    loader := NewLangChainHubLoader()
    template, err := loader.LoadAndConvert(ctx, "langchain-ai", "prompt-hub", "prompts/rag")
    if err != nil {
        log.Fatalf("Failed to load prompt: %v", err)
    }
    
    result, err := template.Format(ctx, map[string]any{
        "context": "AI is transforming technology.",
        "question": "What is AI?",
    })
    if err != nil {
        log.Fatalf("Failed to format: %v", err)
    }

    
    fmt.Printf("Prompt: %s\n", result.ToString())
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `BaseURL` | LangChain Hub API URL | `https://api.hub.langchain.com` | No |
| `Timeout` | HTTP request timeout | `30s` | No |
| `APIKey` | API key for private prompts | - | No |

## Common Issues

### "Prompt not found"

**Problem**: Invalid owner, repo, or prompt ID.

**Solution**: Verify prompt path:curl https://api.hub.langchain.com/repos/langchain-ai/prompt-hub/prompts/rag
```

### "Format conversion failed"

**Problem**: LangChain format incompatible.

**Solution**: Handle format differences:// Convert LangChain format to Beluga AI format
// Handle different message types and structures
```

## Production Considerations

When using LangChain Hub in production:

- **Cache prompts**: Cache loaded prompts to reduce API calls
- **Error handling**: Handle network failures gracefully
- **Versioning**: Pin prompt versions for stability
- **Validation**: Validate loaded prompts before use
- **Fallbacks**: Have local fallback prompts

## Next Steps

Congratulations! You've integrated LangChain Hub with Beluga AI. Next, learn how to:

- **[Local Filesystem Template Store](./local-filesystem-template-store.md)** - Local prompt storage
- **[Prompts Package Documentation](../../api-docs/packages/prompts.md)** - Deep dive into prompts package
- **[Prompt Templates Guide](../../guides/llm-providers.md)** - Advanced prompt patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
