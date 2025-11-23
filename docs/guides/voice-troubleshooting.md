# Voice Troubleshooting Guide

This guide helps diagnose and fix common issues with Voice Agents.

## Common Issues

### Session Won't Start

**Symptoms:**
- `Start()` returns error
- Session remains in "initial" state

**Solutions:**
1. Check provider configuration:
   ```go
   if sttProvider == nil || ttsProvider == nil {
       return errors.New("providers not configured")
   }
   ```

2. Verify API keys:
   ```bash
   echo $DEEPGRAM_API_KEY
   echo $OPENAI_API_KEY
   ```

3. Check network connectivity:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   ```

### Audio Not Processing

**Symptoms:**
- `ProcessAudio()` returns error
- No transcripts generated

**Solutions:**
1. Verify audio format:
   ```go
   // Audio should be PCM, 16kHz, 16-bit, mono
   ```

2. Check STT provider:
   ```go
   transcript, err := sttProvider.Transcribe(ctx, audio)
   if err != nil {
       log.Printf("STT error: %v", err)
   }
   ```

3. Verify session state:
   ```go
   if voiceSession.GetState() != "listening" {
       return errors.New("session not listening")
   }
   ```

### High Latency

**Symptoms:**
- Slow response times
- Audio delays

**Solutions:**
1. Use streaming:
   ```go
   sttProvider.StartStreaming(ctx)
   ```

2. Reduce audio chunk size:
   ```go
   chunkSize := 1600 // 20ms at 16kHz
   ```

3. Check network latency:
   ```bash
   ping api.deepgram.com
   ```

### Provider Errors

**Symptoms:**
- Provider returns errors
- Fallback not working

**Solutions:**
1. Check error codes:
   ```go
   if err != nil {
       var providerErr *ProviderError
       if errors.As(err, &providerErr) {
           switch providerErr.Code {
           case ErrCodeRateLimit:
               // Handle rate limit
           case ErrCodeTimeout:
               // Handle timeout
           }
       }
   }
   ```

2. Implement retry logic:
   ```go
   recovery := internal.NewErrorRecovery(3, 1*time.Second)
   err := recovery.RetryWithBackoff(ctx, "operation", fn)
   ```

3. Use circuit breaker:
   ```go
   breaker := internal.NewCircuitBreaker(5, 2, 30*time.Second)
   err := breaker.Call(fn)
   ```

### Memory Leaks

**Symptoms:**
- Memory usage grows over time
- Sessions not cleaned up

**Solutions:**
1. Always stop sessions:
   ```go
   defer voiceSession.Stop(ctx)
   ```

2. Clear buffers:
   ```go
   session.ClearBuffers()
   ```

3. Monitor session count:
   ```go
   activeSessions := metrics.GetActiveSessions()
   ```

### State Machine Issues

**Symptoms:**
- Invalid state transitions
- Session stuck in state

**Solutions:**
1. Check state transitions:
   ```go
   // Valid: initial -> listening -> ended
   // Invalid: ended -> listening
   ```

2. Verify state machine:
   ```go
   state := voiceSession.GetState()
   if !isValidTransition(state, newState) {
       return errors.New("invalid transition")
   }
   ```

## Debugging

### Enable Debug Logging

```go
import "log/slog"

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

### Trace Operations

```go
ctx, span := tracer.Start(ctx, "voice.session.operation")
defer span.End()
```

### Monitor Metrics

```go
// Check metrics
latency := metrics.GetLatency()
throughput := metrics.GetThroughput()
errorRate := metrics.GetErrorRate()
```

## Provider-Specific Issues

### Deepgram

- **Rate Limits**: Implement exponential backoff
- **Streaming**: Check connection stability
- **API Keys**: Verify key permissions

### Google Cloud

- **Credentials**: Check JSON file path
- **Quotas**: Monitor API quotas
- **Regions**: Use appropriate region

### Azure

- **Subscription**: Verify subscription key
- **Regions**: Match region configuration
- **Endpoints**: Check endpoint URLs

### OpenAI

- **API Keys**: Verify key validity
- **Rate Limits**: Implement rate limiting
- **Models**: Check model availability

## Performance Issues

### High CPU Usage

- Reduce concurrent sessions
- Optimize audio processing
- Use efficient providers

### High Memory Usage

- Reduce buffer sizes
- Clear old data
- Use streaming

### Network Issues

- Check connectivity
- Monitor latency
- Use CDN if available

## Getting Help

1. Check logs for error messages
2. Review metrics for patterns
3. Test with minimal configuration
4. Check provider status pages
5. Review documentation

## Example: Debug Session

```go
// Enable debug logging
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Create session with logging
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithLogger(logger),
)

// Monitor state changes
voiceSession.OnStateChanged(func(state SessionState) {
    logger.Debug("State changed", "state", state)
})

// Check metrics
metrics := session.GetMetrics()
logger.Debug("Metrics", "active_sessions", metrics.GetActiveSessions())
```

