# Voice Performance Tuning Guide

This guide provides strategies for optimizing voice agent performance.

## Performance Targets

- **Latency**: <200ms for end-to-end processing
- **Throughput**: 1000+ audio chunks/second
- **Concurrency**: 100+ concurrent sessions
- **Memory**: <100MB per session

## Latency Optimization

### 1. Use Streaming

Streaming reduces latency by processing audio incrementally:

```go
// Use streaming STT
sttProvider.StartStreaming(ctx)

// Use streaming TTS
ttsProvider.StreamGenerate(ctx, text)
```

### 2. Optimize Audio Chunk Size

Smaller chunks = lower latency, but more overhead:

```go
// Optimal chunk size: 20-40ms
chunkSize := 3200 // bytes for 16kHz, 16-bit, mono
```

### 3. Parallel Processing

Process STT and TTS in parallel when possible:

```go
// Process audio while generating response
go func() {
    transcript, _ := sttProvider.Transcribe(ctx, audio)
    response, _ := agentCallback(ctx, transcript)
    ttsProvider.GenerateSpeech(ctx, response)
}()
```

## Throughput Optimization

### 1. Connection Pooling

Reuse provider connections:

```go
// Create provider pool
sttPool := NewSTTProviderPool(size: 10)
```

### 2. Batch Processing

Process multiple audio chunks together:

```go
// Batch audio chunks
chunks := [][]byte{chunk1, chunk2, chunk3}
transcripts := sttProvider.TranscribeBatch(ctx, chunks)
```

### 3. Async Processing

Use goroutines for non-blocking operations:

```go
go func() {
    voiceSession.ProcessAudio(ctx, audio)
}()
```

## Memory Optimization

### 1. Limit Buffer Sizes

```go
config := session.DefaultConfig()
config.MaxBufferSize = 1024 * 1024 // 1MB
```

### 2. Clear Old Data

Regularly clear processed audio:

```go
// Clear processed audio buffers
session.ClearBuffers()
```

### 3. Use Streaming

Streaming reduces memory usage:

```go
// Stream instead of buffering
reader := ttsProvider.StreamGenerate(ctx, text)
```

## Concurrency Optimization

### 1. Limit Concurrent Sessions

```go
// Use semaphore to limit concurrency
sem := make(chan struct{}, 100)
```

### 2. Connection Limits

Configure provider connection limits:

```go
config := provider.Config{
    MaxConnections: 100,
}
```

### 3. Resource Pooling

Pool expensive resources:

```go
// Pool STT providers
sttPool := NewProviderPool(10)
```

## Monitoring

### Metrics to Track

- **Latency**: End-to-end processing time
- **Throughput**: Audio chunks/second
- **Error Rate**: Failed operations
- **Memory Usage**: Per session
- **CPU Usage**: Per operation

### Observability

All operations emit OTEL metrics:

```go
// Metrics are automatically emitted
// voice.session.latency
// voice.session.throughput
// voice.session.errors
```

## Benchmarking

Run benchmarks to measure performance:

```bash
go test -bench=. ./pkg/voice/...
```

## Best Practices

1. **Use streaming** for real-time interactions
2. **Monitor metrics** continuously
3. **Set appropriate timeouts**
4. **Implement circuit breakers**
5. **Use connection pooling**
6. **Optimize audio chunk sizes**
7. **Cache frequently used data**

## Troubleshooting Performance Issues

### High Latency

- Check network latency
- Use streaming providers
- Reduce audio chunk size
- Optimize agent processing

### Low Throughput

- Increase connection pool size
- Use batch processing
- Optimize provider configuration
- Check system resources

### High Memory Usage

- Reduce buffer sizes
- Use streaming
- Clear old data regularly
- Monitor session count

## Example: Optimized Session

```go
// Optimized configuration
config := &session.Config{
    Timeout:           30 * time.Minute,
    MaxRetries:        3,
    EnableKeepAlive:   true,
    KeepAliveInterval: 30 * time.Second,
}

// Use streaming providers
sttProvider := deepgram.NewDeepgramSTT(ctx, deepgram.Config{
    Streaming: true,
})

ttsProvider := openai.NewOpenAITTS(ctx, openai.Config{
    Streaming: true,
})

voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithConfig(config),
)
```

