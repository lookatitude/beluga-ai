# S2S Provider Implementation Roadmap

This document provides a detailed roadmap for implementing Speech-to-Speech (S2S) providers in the Beluga AI Framework.

## Table of Contents

1. [Overview](#overview)
2. [Current Status](#current-status)
3. [Implementation Requirements](#implementation-requirements)
4. [Provider-Specific Implementation Guides](#provider-specific-implementation-guides)
5. [Testing Strategy](#testing-strategy)
6. [Integration Points](#integration-points)
7. [Timeline](#timeline)

---

## Overview

S2S (Speech-to-Speech) providers enable end-to-end speech conversations without explicit intermediate text steps. All four S2S providers currently have placeholder implementations that need to be completed.

### Providers to Implement

1. **Amazon Nova 2 Sonic** - AWS Bedrock
2. **OpenAI Realtime** - GPT Realtime API
3. **Gemini 2.5 Flash Native Audio** - Google Gemini
4. **Grok Voice Agent** - xAI Grok

---

## Current Status

### Implementation Status Summary

| Provider | Process() | Streaming Output | Streaming Input (SendAudio) | Status |
|----------|-----------|------------------|----------------------------|--------|
| Amazon Nova | ✅ Complete | ✅ Complete | ⚠️ Partial (buffered, not sent) | Functional |
| Gemini | ✅ Complete | ✅ Complete | ⚠️ Partial (buffered, not sent) | Functional |
| Grok | ✅ Complete | ✅ Complete | ⚠️ Partial (buffered, not sent) | Functional |
| OpenAI Realtime | ✅ Complete | ✅ Complete | ✅ Complete | Fully Functional |

### Implementation Details

All four providers have:
- ✅ Configuration structures - Complete
- ✅ Provider structures - Complete
- ✅ Auto-registration - Complete
- ✅ `Process()` method - **COMPLETE** (makes real API calls)
- ✅ Streaming output - **COMPLETE** (receives streaming responses)
- ⚠️ Streaming input (`SendAudio()`) - **PARTIAL** for Nova/Gemini/Grok

**Note**: Amazon Nova, Gemini, and Grok use one-way streaming APIs (server-to-client only). Their `SendAudio()` methods currently buffer audio but don't send it to the API during an active stream. For bidirectional streaming, use OpenAI Realtime or the non-streaming `Process()` method.

### Files Requiring Implementation

```
pkg/voice/s2s/providers/
├── amazon_nova/
│   ├── provider.go      # Line 105: TODO - API implementation
│   └── streaming.go     # Line 33: TODO - Streaming connection
├── openai_realtime/
│   ├── provider.go      # Line 97: TODO - API implementation
│   └── streaming.go     # Line 32: TODO - Streaming connection
├── gemini/
│   ├── provider.go      # Line 100: TODO - API implementation
│   └── streaming.go     # Line 32: TODO - Streaming connection
└── grok/
    ├── provider.go      # Line 97: TODO - API implementation
    └── streaming.go      # Line 32: TODO - Streaming connection
```

---

## Implementation Requirements

### Core Requirements

Each provider must implement:

1. **Non-Streaming Processing**
   - Accept audio input
   - Process through provider API
   - Return audio output
   - Handle errors and retries

2. **Streaming Support** (if supported by provider)
   - Bidirectional streaming connection
   - Real-time audio input/output
   - Connection lifecycle management
   - Error recovery

3. **Observability**
   - OTEL metrics for all operations
   - Distributed tracing
   - Structured logging

4. **Error Handling**
   - Custom error types with codes
   - Retry logic with exponential backoff
   - Rate limiting handling

### Common Implementation Pattern

```go
// Process implements S2SProvider interface
func (p *{Provider}Provider) Process(
    ctx context.Context,
    input *internal.AudioInput,
    convCtx *internal.ConversationContext,
    opts ...internal.STSOption,
) (*internal.AudioOutput, error) {
    // 1. Start tracing
    ctx, span := s2s.StartProcessSpan(ctx, p.providerName, p.config.Model, input.Language)
    defer span.End()
    
    // 2. Validate input
    if err := internal.ValidateAudioInput(input); err != nil {
        s2s.RecordSpanError(span, err)
        return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidInput, err)
    }
    
    // 3. Apply options
    stsOpts := &internal.STSOptions{}
    for _, opt := range opts {
        opt(stsOpts)
    }
    
    // 4. Prepare API request
    request := p.prepareRequest(input, convCtx, stsOpts)
    
    // 5. Call provider API with retry
    start := time.Now()
    var response *{Provider}Response
    var err error
    
    retryErr := common.RetryWithBackoff(ctx, p.retryConfig, "{provider}.process", func() error {
        response, err = p.client.ProcessAudio(ctx, request)
        return err
    })
    
    if retryErr != nil {
        duration := time.Since(start)
        s2s.RecordSpanError(span, retryErr)
        s2s.GetMetrics().RecordError(ctx, p.providerName, s2s.GetS2SErrorCode(retryErr), duration)
        return nil, retryErr
    }
    
    // 6. Process response
    output := p.processResponse(response, input)
    
    // 7. Record metrics
    duration := time.Since(start)
    s2s.RecordSpanLatency(span, duration)
    s2s.GetMetrics().RecordRequest(ctx, p.providerName, duration)
    
    return output, nil
}
```

---

## Provider-Specific Implementation Guides

### 1. Amazon Nova 2 Sonic

**API Documentation:** AWS Bedrock Runtime API  
**Model ID:** `amazon.nova-2-sonic-v1:0`  
**Endpoint:** Bedrock Runtime API

#### Implementation Steps

1. **Prepare Request**
   ```go
   type NovaRequest struct {
       ModelID    string
       InputAudio []byte
       VoiceID    string
       Language   string
   }
   ```

2. **Call Bedrock Runtime API**
   ```go
   import "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
   
   request := bedrockruntime.InvokeModelInput{
       ModelId: aws.String("amazon.nova-2-sonic-v1:0"),
       Body:    requestBody,
   }
   
   response, err := p.client.InvokeModel(ctx, &request)
   ```

3. **Process Response**
   - Extract audio data from response
   - Parse metadata (voice characteristics, latency)
   - Construct `AudioOutput`

4. **Streaming Implementation**
   - Use Bedrock Streaming API if available
   - Establish WebSocket connection
   - Handle bidirectional audio streams

#### Resources
- AWS Bedrock Runtime API docs
- Nova 2 Sonic model documentation
- AWS SDK Go v2 examples

---

### 2. OpenAI Realtime

**API Documentation:** OpenAI Realtime API  
**Endpoint:** `https://api.openai.com/v1/realtime`  
**Protocol:** WebSocket-based

#### Implementation Steps

1. **Establish WebSocket Connection**
   ```go
   import "github.com/gorilla/websocket"
   
   url := "wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview"
   conn, _, err := websocket.DefaultDialer.Dial(url, headers)
   ```

2. **Send Audio Events**
   ```json
   {
     "type": "input_audio_buffer.append",
     "audio": "<base64_encoded_audio>"
   }
   ```

3. **Receive Audio Events**
   ```json
   {
     "type": "conversation.item.input_audio_transcription.completed",
     "item_id": "...",
     "transcript": "..."
   }
   {
     "type": "response.audio_transcript.delta",
     "delta": "..."
   }
   {
     "type": "response.audio.delta",
     "delta": "<base64_encoded_audio>"
   }
   ```

4. **Handle Session Lifecycle**
   - Session creation
   - Session updates
   - Session completion

#### Resources
- OpenAI Realtime API documentation
- OpenAI Realtime SDK examples
- WebSocket protocol specification

---

### 3. Gemini 2.5 Flash Native Audio

**API Documentation:** Google Gemini API  
**Endpoint:** `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent`  
**Protocol:** REST API with streaming support

#### Implementation Steps

1. **Prepare Request**
   ```go
   type GeminiRequest struct {
       Contents []Content `json:"contents"`
       GenerationConfig  GenerationConfig `json:"generationConfig"`
   }
   
   type Content struct {
       Parts []Part `json:"parts"`
   }
   
   type Part struct {
       InlineData InlineData `json:"inlineData"`
   }
   
   type InlineData struct {
       MimeType string `json:"mimeType"`
       Data     string `json:"data"` // base64
   }
   ```

2. **Call Gemini API**
   ```go
   import "google.golang.org/api/generativelanguage/v1beta"
   
   service, _ := generativelanguage.NewService(ctx)
   call := service.Media.GenerateContent(model, request)
   response, err := call.Do()
   ```

3. **Process Response**
   - Extract audio from response parts
   - Handle streaming responses
   - Parse metadata

4. **Streaming Implementation**
   - Use Server-Sent Events (SSE) or WebSocket
   - Handle incremental audio chunks
   - Manage connection lifecycle

#### Resources
- Google Gemini API documentation
- Google Cloud Go client library
- Gemini Native Audio examples

---

### 4. Grok Voice Agent

**API Documentation:** xAI Grok API  
**Endpoint:** `https://api.x.ai/v1/voice/agent`  
**Protocol:** REST API with WebSocket streaming

#### Implementation Steps

1. **Prepare Request**
   ```go
   type GrokRequest struct {
       Audio    string `json:"audio"` // base64
       VoiceID  string `json:"voice_id"`
       Language string `json:"language"`
   }
   ```

2. **Call Grok API**
   ```go
   import "net/http"
   
   req, _ := http.NewRequest("POST", "https://api.x.ai/v1/voice/agent", body)
   req.Header.Set("Authorization", "Bearer "+apiKey)
   req.Header.Set("Content-Type", "application/json")
   
   resp, err := http.DefaultClient.Do(req)
   ```

3. **Process Response**
   - Extract audio from JSON response
   - Handle streaming if available
   - Parse metadata

4. **Streaming Implementation**
   - Establish WebSocket connection
   - Handle bidirectional audio streams
   - Manage connection lifecycle

#### Resources
- xAI API documentation
- Grok Voice Agent examples
- xAI SDK (if available)

---

## Testing Strategy

### Unit Tests

Each provider needs:

1. **Configuration Tests**
   ```go
   func Test{Provider}Config_Validate(t *testing.T) {
       tests := []struct {
           name    string
           config  *{Provider}Config
           wantErr bool
       }{
           {
               name: "valid config",
               config: validConfig(),
               wantErr: false,
           },
           {
               name: "missing API key",
               config: configWithoutAPIKey(),
               wantErr: true,
           },
       }
       // ...
   }
   ```

2. **Process Tests** (with mocked API)
   ```go
   func Test{Provider}Provider_Process(t *testing.T) {
       // Mock external API
       mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           // Return mock response
       }))
       defer mockServer.Close()
       
       // Test successful processing
       // Test error handling
       // Test retry logic
   }
   ```

3. **Streaming Tests**
   ```go
   func Test{Provider}StreamingSession(t *testing.T) {
       // Test connection establishment
       // Test audio sending
       // Test audio receiving
       // Test connection closure
       // Test error recovery
   }
   ```

### Integration Tests

1. **End-to-End Tests**
   - Real API calls (with test credentials)
   - Full conversation flow
   - Error scenarios

2. **Performance Tests**
   - Latency measurements
   - Throughput tests
   - Concurrent session tests

### Mock Implementation

Create mock providers for testing:

```go
type MockS2SProvider struct {
    // Mock implementation
}

func (m *MockS2SProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
    // Return mock output
}
```

---

## Integration Points

### 1. Voice Session Integration

S2S providers integrate with the voice session package:

```go
// pkg/voice/session/session.go
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithS2SProvider(s2sProvider),
    session.WithAgentInstance(agent, agentConfig),
)
```

### 2. Agent Integration

For external reasoning mode:

```go
// pkg/voice/session/internal/s2s_agent_integration.go
if opts.S2SProvider != nil && agentIntegration != nil {
    impl.s2sAgentIntegration = NewS2SAgentIntegration(
        opts.S2SProvider,
        agentIntegration,
        "external",
    )
}
```

### 3. Memory Integration

S2S providers should integrate with memory for conversation context:

```go
// Save conversation context
memory.SaveContext(ctx, map[string]any{
    "input":  input,
    "output": output,
})
```

### 4. Observability Integration

All operations should be instrumented:

```go
// Metrics
s2s.GetMetrics().RecordRequest(ctx, providerName, duration)

// Tracing
ctx, span := s2s.StartProcessSpan(ctx, provider, model, language)
defer span.End()

// Logging
s2s.LogInfo(ctx, "Processing audio", map[string]any{
    "provider": providerName,
    "size":     len(input.Data),
})
```

---

## Timeline

### Phase 1: Foundation (Week 1)

- [ ] Research each provider's API documentation
- [ ] Set up test accounts/credentials
- [ ] Create API client wrappers
- [ ] Implement basic request/response handling

### Phase 2: Core Implementation (Weeks 2-3)

**Week 2:**
- [ ] Implement Amazon Nova provider (non-streaming)
- [ ] Implement OpenAI Realtime provider (non-streaming)
- [ ] Add comprehensive tests

**Week 3:**
- [ ] Implement Gemini provider (non-streaming)
- [ ] Implement Grok provider (non-streaming)
- [ ] Add integration tests

### Phase 3: Streaming Support (Week 4)

- [ ] Implement Amazon Nova streaming
- [ ] Implement OpenAI Realtime streaming
- [ ] Implement Gemini streaming
- [ ] Implement Grok streaming
- [ ] Add streaming tests

### Phase 4: Polish (Week 5)

- [ ] Error handling improvements
- [ ] Performance optimization
- [ ] Documentation updates
- [ ] Example implementations
- [ ] Final testing

---

## Implementation Checklist

For each provider:

### Non-Streaming
- [ ] Implement `Process()` method
- [ ] Handle API authentication
- [ ] Implement request preparation
- [ ] Implement response parsing
- [ ] Add error handling
- [ ] Add retry logic
- [ ] Add OTEL instrumentation
- [ ] Add unit tests
- [ ] Add integration tests

### Streaming
- [ ] Implement `StartStreaming()` method
- [ ] Implement `StreamingSession` interface
- [ ] Handle connection establishment
- [ ] Implement `SendAudio()`
- [ ] Implement `ReceiveAudio()`
- [ ] Handle connection lifecycle
- [ ] Add error recovery
- [ ] Add streaming tests

### General
- [ ] Update provider documentation
- [ ] Add usage examples
- [ ] Update README
- [ ] Add to integration tests
- [ ] Performance benchmarking
- [ ] Security review

---

## Resources

### Documentation
- [S2S Package README](../../../pkg/voice/s2s/README.md)
- [Provider Implementation Guide](./implementing-providers.md)
- [Beluga AI Design Patterns](../package_design_patterns.md)

### API Documentation
- [AWS Bedrock Runtime API](https://docs.aws.amazon.com/bedrock/latest/userguide/API_Operations_Amazon_Bedrock_Runtime.html)
- [OpenAI Realtime API](https://platform.openai.com/docs/guides/realtime)
- [Google Gemini API](https://ai.google.dev/api)
- [xAI Grok API](https://docs.x.ai)

### Code References
- `pkg/voice/s2s/providers/amazon_nova/` - Structure reference
- `pkg/voice/s2s/iface/` - Interface definitions
- `pkg/voice/s2s/internal/` - Internal types and utilities

---

## Getting Help

- Review existing provider structures
- Check API documentation for each provider
- Review `docs/guides/implementing-providers.md`
- Ask questions in project discussions
