# Mock Package

The mock package provides centralized mock implementations for testing across the Beluga AI Framework. It offers easy-to-use mock factories for all major interfaces, reducing boilerplate in test code.

## Features

- **LLM Mock**: Configurable LLM mock with response control
- **Embedder Mock**: Mock embeddings with configurable dimensions
- **STT Mock**: Mock speech-to-text with configurable transcriptions
- **TTS Mock**: Mock text-to-speech with configurable audio output
- **Tool Mock**: Mock tool with configurable results

## Usage

### Mock LLM

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/mock"

// Create a mock LLM with a custom response
llm := mock.NewLLM(mock.WithResponse("Hello, world!"))

// Use in tests
result, err := llm.Invoke(ctx, "test input")
fmt.Println(result) // "Hello, world!"

// Check call count
fmt.Println(llm.CallCount()) // 1

// Configure error responses
errorLLM := mock.NewLLM(mock.WithError(errors.New("test error")))
```

### Mock Embedder

```go
// Create a mock embedder with custom dimension
embedder := mock.NewEmbedder(mock.WithDimension(768))

// Embed documents
embeddings, err := embedder.EmbedDocuments(ctx, []string{"text1", "text2"})
// Returns [][]float32 with dimension 768

// Embed single query
embedding, err := embedder.EmbedQuery(ctx, "query text")
```

### Mock STT

```go
// Create a mock speech-to-text
stt := mock.NewSTT(mock.WithTranscription("hello world"))

// Transcribe audio
text, err := stt.Transcribe(ctx, audioBytes)
fmt.Println(text) // "hello world"
```

### Mock TTS

```go
// Create a mock text-to-speech
tts := mock.NewTTS(mock.WithAudioData([]byte("audio")))

// Synthesize speech
audio, err := tts.Synthesize(ctx, "hello")
```

### Mock Tool

```go
// Create a mock tool
tool := mock.NewTool("calculator", "Performs calculations",
    mock.WithResult("42"))

// Execute tool
result, err := tool.Execute(ctx, "2+2")
fmt.Println(result) // "42"
```

## Options

### LLM Options
- `WithResponse(string)` - Set the mock response
- `WithError(error)` - Set an error to return
- `WithModelName(string)` - Set the model name

### Embedder Options
- `WithDimension(int)` - Set embedding dimension
- `WithEmbedderError(error)` - Set an error to return

### STT Options
- `WithTranscription(string)` - Set the mock transcription
- `WithSTTError(error)` - Set an error to return

### TTS Options
- `WithAudioData([]byte)` - Set the mock audio data
- `WithTTSError(error)` - Set an error to return

### Tool Options
- `WithResult(any)` - Set the mock result
- `WithToolError(error)` - Set an error to return

## Testing Utilities

All mocks include:
- `CallCount()` - Returns the number of calls made
- `Reset()` - Resets the call count
