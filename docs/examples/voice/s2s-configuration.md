# S2S Configuration Guide

This guide provides comprehensive examples for configuring Speech-to-Speech (S2S) providers in the Beluga AI Framework.

## Overview

S2S providers enable end-to-end speech conversations without explicit intermediate text steps. This guide covers:

- Basic provider configuration
- Multi-provider setup with fallback
- External reasoning mode configuration
- Performance optimization
- Production-ready configurations

## Basic Configuration

### Amazon Nova 2 Sonic

```yaml
s2s:
  provider: "amazon_nova"
  api_key: "${AWS_ACCESS_KEY_ID}"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "low"
  reasoning_mode: "built-in"
  timeout: "30s"
  max_retries: 3
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 100
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
  provider_specific:
    region: "us-east-1"
    model: "nova-2-sonic"
    voice_id: "Ruth"
    language_code: "en-US"
```

### Grok Voice Agent

```yaml
s2s:
  provider: "grok"
  api_key: "${XAI_API_KEY}"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "medium"
  reasoning_mode: "built-in"
  timeout: "30s"
  max_retries: 3
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 50
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
  provider_specific:
    model: "grok-voice-agent"
    voice_id: "alloy"
    language_code: "en-US"
    temperature: 0.8
```

### Gemini 2.5 Flash Native Audio

```yaml
s2s:
  provider: "gemini"
  api_key: "${GOOGLE_API_KEY}"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "low"
  reasoning_mode: "built-in"
  timeout: "30s"
  max_retries: 3
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 100
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
  provider_specific:
    model: "gemini-2.5-flash-native-audio"
    voice_id: "default"
    language_code: "en-US"
```

### OpenAI Realtime

```yaml
s2s:
  provider: "openai_realtime"
  api_key: "${OPENAI_API_KEY}"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "low"
  reasoning_mode: "built-in"
  timeout: "30s"
  max_retries: 3
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 100
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
  provider_specific:
    model: "gpt-4o-realtime-preview"
    voice_id: "alloy"
    language_code: "en-US"
```

## Multi-Provider Configuration

Configure multiple providers with automatic fallback:

```yaml
s2s:
  provider: "amazon_nova"
  api_key: "${AWS_ACCESS_KEY_ID}"
  fallback_providers:
    - "grok"
    - "gemini"
    - "openai_realtime"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "low"
  reasoning_mode: "built-in"
  timeout: "30s"
  max_retries: 3
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 100
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
```

**How it works:**
- Primary provider (`amazon_nova`) is used first
- If primary fails, system automatically tries fallback providers in order
- Circuit breaker prevents rapid switching between providers
- Metrics track fallback events

## External Reasoning Mode

Configure S2S to route audio through Beluga AI agents:

```yaml
s2s:
  provider: "amazon_nova"
  api_key: "${AWS_ACCESS_KEY_ID}"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "medium"
  reasoning_mode: "external"  # Routes through Beluga AI agents
  timeout: "30s"
  max_retries: 3
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 50
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
```

**When to use:**
- Custom reasoning logic required
- Integration with Beluga AI agents
- Memory and orchestration integration needed
- Custom workflow triggers

## Performance Optimization

### Low Latency Configuration

```yaml
s2s:
  provider: "amazon_nova"
  api_key: "${AWS_ACCESS_KEY_ID}"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "low"  # Optimize for low latency
  reasoning_mode: "built-in"
  timeout: "10s"  # Shorter timeout
  max_retries: 2  # Fewer retries for speed
  retry_delay: "500ms"  # Faster retry
  retry_backoff: 1.5
  max_concurrent_sessions: 200
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
```

### High Quality Configuration

```yaml
s2s:
  provider: "amazon_nova"
  api_key: "${AWS_ACCESS_KEY_ID}"
  sample_rate: 48000  # Higher sample rate
  channels: 2  # Stereo
  language: "en-US"
  latency_target: "high"  # Allow more time for quality
  reasoning_mode: "built-in"
  timeout: "60s"  # Longer timeout
  max_retries: 5  # More retries for reliability
  retry_delay: "2s"
  retry_backoff: 2.0
  max_concurrent_sessions: 50
  enable_tracing: true
  enable_metrics: true
  enable_structured_logging: true
```

## Production Configuration

Recommended configuration for production environments:

```yaml
s2s:
  provider: "amazon_nova"
  api_key: "${AWS_ACCESS_KEY_ID}"
  fallback_providers:
    - "grok"
    - "gemini"
  sample_rate: 24000
  channels: 1
  language: "en-US"
  latency_target: "medium"
  reasoning_mode: "built-in"
  timeout: "30s"
  max_retries: 5  # More retries for reliability
  retry_delay: "1s"
  retry_backoff: 2.0
  max_concurrent_sessions: 100
  enable_tracing: true  # Full observability
  enable_metrics: true
  enable_structured_logging: true
```

## Configuration Options

### Audio Settings

- **sample_rate**: Audio sample rate in Hz (8000, 16000, 24000, 48000)
- **channels**: Number of audio channels (1 = mono, 2 = stereo)
- **language**: Language code (e.g., "en-US", "es-ES")

### Performance Settings

- **latency_target**: Target latency ("low", "medium", "high")
- **timeout**: Request timeout (e.g., "30s", "1m")
- **max_retries**: Maximum retry attempts (0-10)
- **retry_delay**: Delay between retries (e.g., "1s", "500ms")
- **retry_backoff**: Exponential backoff multiplier (1.0-5.0)
- **max_concurrent_sessions**: Maximum concurrent sessions per provider (1-1000)

### Observability Settings

- **enable_tracing**: Enable OpenTelemetry distributed tracing
- **enable_metrics**: Enable OTEL metrics collection
- **enable_structured_logging**: Enable structured logging with context

### Reasoning Mode

- **reasoning_mode**: "built-in" (provider handles reasoning) or "external" (routes through Beluga AI agents)

## Environment Variables

Use environment variables for sensitive configuration:

```bash
export AWS_ACCESS_KEY_ID=your-aws-key
export XAI_API_KEY=your-xai-key
export GOOGLE_API_KEY=your-google-key
export OPENAI_API_KEY=your-openai-key
```

Then reference in YAML:

```yaml
api_key: "${AWS_ACCESS_KEY_ID}"
```

## Best Practices

1. **Always configure fallback providers** for production reliability
2. **Use environment variables** for API keys
3. **Enable observability** (tracing, metrics, logging) in production
4. **Set appropriate timeouts** based on your latency requirements
5. **Monitor metrics** to track provider performance and fallback events
6. **Use external reasoning mode** when custom logic is needed
7. **Configure retry logic** appropriately for your use case

## Troubleshooting

### Provider Not Found

```yaml
# Ensure provider name matches registered providers:
# - amazon_nova
# - grok
# - gemini
# - openai_realtime
provider: "amazon_nova"  # Must match exactly
```

### API Key Issues

```yaml
# Use environment variables
api_key: "${AWS_ACCESS_KEY_ID}"

# Or set directly (not recommended for production)
api_key: "your-api-key-here"
```

### Fallback Not Working

```yaml
# Ensure fallback providers are configured
fallback_providers:
  - "grok"
  - "gemini"
  # Providers must be registered and have valid API keys
```

## Related Documentation

- [S2S Package README](https://github.com/lookatitude/beluga-ai/tree/main/pkg/voice/s2s/README.md)
- [Voice Providers Guide](../../guides/voice-providers.md)
- [Voice Session Package](https://github.com/lookatitude/beluga-ai/tree/main/pkg/voice/session/README.md)
