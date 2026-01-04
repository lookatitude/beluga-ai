# Quickstart: Real-Time Voice Agent Support

**Date**: 2025-01-27  
**Feature**: Real-Time Voice Agent Support

## Overview

This quickstart demonstrates how to create a voice session with a streaming agent that processes user speech in real-time and responds with ultra-low latency (< 500ms).

## Prerequisites

- Go 1.24.1+
- Beluga AI Framework packages: `pkg/agents`, `pkg/voice/session`, `pkg/llms`, `pkg/voice/stt`, `pkg/voice/tts`, `pkg/voice/transport`
- LLM provider with streaming support (OpenAI, Anthropic, etc.)
- STT provider (Azure, Deepgram, Google, OpenAI)
- TTS provider (Azure, ElevenLabs, Google, OpenAI)
- Transport provider (WebSocket or WebRTC)

## Basic Example

### Step 1: Create a Streaming Agent

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/iface"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

func main() {
    ctx := context.Background()
    
    // 1. Create LLM with streaming support
    llm, err := llms.NewChatModel(ctx, "openai", llms.Config{
        APIKey: "your-api-key",
        Model:  "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. Create agent with streaming capabilities
    agent, err := agents.NewBaseAgent("voice-agent", llm, nil,
        agents.WithStreaming(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Create STT provider
    sttProvider, err := stt.NewProvider(ctx, "azure", stt.Config{
        APIKey: "your-stt-key",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 4. Create TTS provider
    ttsProvider, err := tts.NewProvider(ctx, "azure", tts.Config{
        APIKey: "your-tts-key",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 5. Create transport provider
    transportProvider, err := transport.NewProvider(ctx, "websocket", transport.Config{
        URL: "wss://your-websocket-server",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 6. Create voice session with agent instance
    voiceSession, err := session.NewVoiceSession(ctx,
        session.WithSTTProvider(sttProvider),
        session.WithTTSProvider(ttsProvider),
        session.WithTransport(transportProvider),
        session.WithAgentInstance(agent, &agents.AgentConfig{
            EnableStreaming: true,
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 7. Start session
    err = voiceSession.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // 8. Process audio (in real application, this would come from transport)
    audio := []byte{...} // Your audio data
    err = voiceSession.ProcessAudio(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    // 9. Wait for session to complete or handle events
    // (In real application, you'd handle this via callbacks or event loop)
}
```

## Advanced Example: With Tools

```go
// Create agent with tools
tools := []tools.Tool{
    tools.NewCalculatorTool(),
    tools.NewEchoTool(),
}

agent, err := agents.NewBaseAgent("voice-agent-with-tools", llm, tools,
    agents.WithStreaming(true),
    agents.WithMaxRetries(3),
)

// Agent will automatically use tools during voice calls
// Tool execution results will be included in agent responses
```

## Advanced Example: Custom Streaming Configuration

```go
agent, err := agents.NewBaseAgent("voice-agent", llm, nil,
    agents.WithStreaming(true),
    agents.WithStreamingConfig(agents.StreamingConfig{
        ChunkBufferSize:     20,
        SentenceBoundary:   true,  // Wait for sentence boundaries before TTS
        InterruptOnNewInput: true, // Allow interruption on new input
        MaxStreamDuration:  30 * time.Second,
    }),
)
```

## Testing the Implementation

### Unit Test Example

```go
func TestStreamingAgent(t *testing.T) {
    ctx := context.Background()
    
    // Create mock LLM with streaming
    mockLLM := llms.NewMockChatModel()
    mockLLM.OnStreamChat = func(ctx context.Context, messages []schema.Message) (<-chan iface.AIMessageChunk, error) {
        ch := make(chan iface.AIMessageChunk, 1)
        go func() {
            defer close(ch)
            ch <- iface.AIMessageChunk{Content: "Hello"}
            ch <- iface.AIMessageChunk{Content: " world"}
        }()
        return ch, nil
    }
    
    // Create agent
    agent, err := agents.NewBaseAgent("test-agent", mockLLM, nil,
        agents.WithStreaming(true),
    )
    require.NoError(t, err)
    
    // Test streaming execution
    chunkChan, err := agent.StreamExecute(ctx, map[string]any{"input": "test"})
    require.NoError(t, err)
    
    var chunks []iface.AgentStreamChunk
    for chunk := range chunkChan {
        chunks = append(chunks, chunk)
    }
    
    assert.Greater(t, len(chunks), 0)
}
```

### Integration Test Example

```go
func TestVoiceSessionWithAgent(t *testing.T) {
    ctx := context.Background()
    
    // Create real providers (or use mocks)
    llm := createTestLLM(t)
    stt := createTestSTT(t)
    tts := createTestTTS(t)
    transport := createTestTransport(t)
    
    // Create agent
    agent, err := agents.NewBaseAgent("test-agent", llm, nil,
        agents.WithStreaming(true),
    )
    require.NoError(t, err)
    
    // Create voice session
    session, err := session.NewVoiceSession(ctx,
        session.WithSTTProvider(stt),
        session.WithTTSProvider(tts),
        session.WithTransport(transport),
        session.WithAgentInstance(agent, &agents.AgentConfig{}),
    )
    require.NoError(t, err)
    
    // Test end-to-end
    err = session.Start(ctx)
    require.NoError(t, err)
    
    // Send audio
    audio := generateTestAudio(t)
    err = session.ProcessAudio(ctx, audio)
    require.NoError(t, err)
    
    // Verify response (check transport output or use callbacks)
    // ...
}
```

## Performance Validation

### Latency Measurement

```go
func TestEndToEndLatency(t *testing.T) {
    start := time.Now()
    
    // Process audio through full pipeline
    err := session.ProcessAudio(ctx, audio)
    require.NoError(t, err)
    
    // Wait for response
    // (In real test, you'd wait for TTS output or use callbacks)
    
    latency := time.Since(start)
    assert.Less(t, latency, 500*time.Millisecond, "End-to-end latency must be < 500ms")
}
```

## Common Patterns

### Handling Interruptions

```go
// Interruptions are handled automatically by the session
// When new user input arrives during agent response:
// 1. Current agent stream is cancelled
// 2. TTS generation stops
// 3. Conversation context is preserved
// 4. New input is processed immediately
```

### Monitoring Performance

```go
// OTEL metrics are automatically recorded:
// - agent.streaming.latency: Time from input to first chunk
// - agent.streaming.duration: Total streaming duration
// - agent.tool.execution.time: Tool execution time
// - voice.session.end_to_end_latency: End-to-end latency
```

### Error Handling

```go
// Errors are handled gracefully:
// - Agent errors don't crash the session
// - Errors are logged and reported via metrics
// - Session can recover from errors
// - User receives appropriate error messages
```

## Next Steps

1. **Read the full documentation**: See `pkg/agents/README.md` and `pkg/voice/session/README.md`
2. **Explore examples**: Check `examples/voice/` directory
3. **Run integration tests**: `go test ./tests/integration/...`
4. **Monitor metrics**: Use OTEL to monitor performance and errors

## Troubleshooting

### High Latency
- Check LLM provider response time
- Verify network latency to providers
- Check TTS provider performance
- Review conversation history size (may impact LLM processing)

### Interruptions Not Working
- Verify `InterruptOnNewInput` is enabled
- Check context cancellation is working
- Review interruption detection logic

### Memory Issues
- Check chunk buffer sizes
- Review conversation history limits
- Monitor memory usage via OTEL metrics

## Validation Checklist

After implementing, verify:

- [ ] Voice session accepts agent instance
- [ ] Agent streams responses in real-time
- [ ] End-to-end latency < 500ms
- [ ] Interruptions work correctly
- [ ] Conversation context preserved
- [ ] Tool execution works during voice calls
- [ ] Multiple concurrent sessions work
- [ ] Error handling is graceful
- [ ] OTEL metrics are recorded
- [ ] Tests pass with 90%+ coverage

