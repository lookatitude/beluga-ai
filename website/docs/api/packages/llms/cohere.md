<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# cohere

```go
import "github.com/lookatitude/beluga-ai/llms/cohere"
```

Package cohere provides an implementation of the llms.ChatModel interface using the Cohere API.

## Index

- [type CohereChat](<#CohereChat>)
  - [func NewCohereChat\(options ...CohereOption\) \(\*CohereChat, error\)](<#NewCohereChat>)
  - [func \(cc \*CohereChat\) Batch\(ctx context.Context, inputs \[\]any, options ...core.Option\) \(\[\]any, error\)](<#CohereChat.Batch>)
  - [func \(cc \*CohereChat\) BindTools\(toolsToBind \[\]tools.Tool\) llms.ChatModel](<#CohereChat.BindTools>)
  - [func \(cc \*CohereChat\) Generate\(ctx context.Context, messages \[\]schema.Message, options ...core.Option\) \(schema.Message, error\)](<#CohereChat.Generate>)
  - [func \(cc \*CohereChat\) GetBoundTools\(\) \[\]\*cohere.Tool](<#CohereChat.GetBoundTools>)
  - [func \(cc \*CohereChat\) GetModelName\(\) string](<#CohereChat.GetModelName>)
  - [func \(cc \*CohereChat\) Invoke\(ctx context.Context, input any, options ...core.Option\) \(any, error\)](<#CohereChat.Invoke>)
  - [func \(cc \*CohereChat\) Stream\(ctx context.Context, input any, options ...core.Option\) \(\<\-chan core.Chunk, error\)](<#CohereChat.Stream>)
  - [func \(cc \*CohereChat\) StreamChat\(ctx context.Context, messages \[\]schema.Message, options ...core.Option\) \(\<\-chan llms.AIMessageChunk, error\)](<#CohereChat.StreamChat>)
- [type CohereOption](<#CohereOption>)
  - [func WithCohereDefaultMaxTokens\(maxTokens int\) CohereOption](<#WithCohereDefaultMaxTokens>)
  - [func WithCohereDefaultTemperature\(temp float64\) CohereOption](<#WithCohereDefaultTemperature>)
  - [func WithCohereMaxConcurrentBatches\(n int\) CohereOption](<#WithCohereMaxConcurrentBatches>)


<a name="CohereChat"></a>
## type CohereChat

CohereChat represents a chat model client for Cohere.

```go
type CohereChat struct {
    // contains filtered or unexported fields
}
```

<a name="NewCohereChat"></a>
### func NewCohereChat

```go
func NewCohereChat(options ...CohereOption) (*CohereChat, error)
```

NewCohereChat creates a new Cohere chat client.

<a name="CohereChat.Batch"></a>
### func \(\*CohereChat\) Batch

```go
func (cc *CohereChat) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
```

Batch implements the core.Runnable interface.

<a name="CohereChat.BindTools"></a>
### func \(\*CohereChat\) BindTools

```go
func (cc *CohereChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel
```

BindTools implements the llms.ChatModel interface.

<a name="CohereChat.Generate"></a>
### func \(\*CohereChat\) Generate

```go
func (cc *CohereChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
```

Generate implements the llms.ChatModel interface.

<a name="CohereChat.GetBoundTools"></a>
### func \(\*CohereChat\) GetBoundTools

```go
func (cc *CohereChat) GetBoundTools() []*cohere.Tool
```

GetBoundTools returns the tools bound to the client.

<a name="CohereChat.GetModelName"></a>
### func \(\*CohereChat\) GetModelName

```go
func (cc *CohereChat) GetModelName() string
```

GetModelName returns the model name used by the client.

<a name="CohereChat.Invoke"></a>
### func \(\*CohereChat\) Invoke

```go
func (cc *CohereChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error)
```

Invoke implements the core.Runnable interface.

<a name="CohereChat.Stream"></a>
### func \(\*CohereChat\) Stream

```go
func (cc *CohereChat) Stream(ctx context.Context, input any, options ...core.Option) (<-chan core.Chunk, error)
```

Stream implements the core.Runnable interface.

<a name="CohereChat.StreamChat"></a>
### func \(\*CohereChat\) StreamChat

```go
func (cc *CohereChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error)
```

StreamChat implements the llms.ChatModel interface.

<a name="CohereOption"></a>
## type CohereOption

CohereOption is a function type for setting options on the CohereChat client.

```go
type CohereOption func(*CohereChat)
```

<a name="WithCohereDefaultMaxTokens"></a>
### func WithCohereDefaultMaxTokens

```go
func WithCohereDefaultMaxTokens(maxTokens int) CohereOption
```

WithCohereDefaultMaxTokens sets the default max tokens.

<a name="WithCohereDefaultTemperature"></a>
### func WithCohereDefaultTemperature

```go
func WithCohereDefaultTemperature(temp float64) CohereOption
```

WithCohereDefaultTemperature sets the default temperature.

<a name="WithCohereMaxConcurrentBatches"></a>
### func WithCohereMaxConcurrentBatches

```go
func WithCohereMaxConcurrentBatches(n int) CohereOption
```

WithCohereMaxConcurrentBatches sets the concurrency limit for Batch.

Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
