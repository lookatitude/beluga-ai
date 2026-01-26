# Configuration Propagation Standard

## Purpose

Define how configuration flows from parent packages to sub-packages in wrapper/aggregation packages. This ensures consistent configuration handling and validation across package hierarchies.

## Config Flow

**Direction:** Parent → Sub-package → Options → Validation

```
VoiceConfig (parent)
├── STT: stt.Config (embedded)
├── TTS: tts.Config (embedded)
├── VAD: vad.Config (embedded)
└── Root options (timeout, max_sessions)
```

## Parent Config Structure

Parent configs embed sub-package configs with standardized YAML/mapstructure tags:

```go
// pkg/voice/config.go
package voice

import (
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
    "github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

type Config struct {
    // Sub-package configs embedded with consistent tags
    STT       stt.Config       `yaml:"stt" mapstructure:"stt" json:"stt"`
    TTS       tts.Config       `yaml:"tts" mapstructure:"tts" json:"tts"`
    VAD       vad.Config       `yaml:"vad" mapstructure:"vad" json:"vad"`
    Transport transport.Config `yaml:"transport" mapstructure:"transport" json:"transport"`

    // Root-level shared options
    SessionTimeout  time.Duration `yaml:"session_timeout" mapstructure:"session_timeout" default:"5m" validate:"required"`
    MaxConcurrent   int           `yaml:"max_concurrent" mapstructure:"max_concurrent" default:"100" validate:"gte=1"`
    EnableTelemetry bool          `yaml:"enable_telemetry" mapstructure:"enable_telemetry" default:"true"`
}
```

## Sub-Package Config Structure

Each sub-package defines its own config:

```go
// pkg/voice/stt/config.go
package stt

type Config struct {
    Provider   string `yaml:"provider" mapstructure:"provider" validate:"required"`
    APIKey     string `yaml:"api_key" mapstructure:"api_key" validate:"required"`
    Model      string `yaml:"model" mapstructure:"model"`
    SampleRate int    `yaml:"sample_rate" mapstructure:"sample_rate" default:"16000"`
    Language   string `yaml:"language" mapstructure:"language" default:"en-US"`
    Encoding   string `yaml:"encoding" mapstructure:"encoding" default:"linear16"`

    // Provider-specific options
    DeepgramOptions *DeepgramOptions `yaml:"deepgram" mapstructure:"deepgram,omitempty"`
    OpenAIOptions   *OpenAIOptions   `yaml:"openai" mapstructure:"openai,omitempty"`
}
```

## YAML Configuration Example

```yaml
voice:
  # Root-level options
  session_timeout: 5m
  max_concurrent: 100
  enable_telemetry: true

  # STT sub-package config
  stt:
    provider: deepgram
    api_key: ${DEEPGRAM_API_KEY}
    model: nova-2
    sample_rate: 16000
    language: en-US
    deepgram:
      punctuate: true
      diarize: true

  # TTS sub-package config
  tts:
    provider: elevenlabs
    api_key: ${ELEVENLABS_API_KEY}
    voice_id: default
    elevenlabs:
      stability: 0.5
      similarity_boost: 0.75

  # VAD sub-package config
  vad:
    provider: silero
    threshold: 0.5
    min_silence_duration: 300ms

  # Transport sub-package config
  transport:
    provider: webrtc
    ice_servers:
      - urls: ["stun:stun.l.google.com:19302"]
```

## Config Loading

### Parent Loads and Propagates

```go
// pkg/voice/config.go
package voice

func LoadConfig(v *viper.Viper) (*Config, error) {
    cfg := &Config{}

    // Load entire voice section
    if err := v.UnmarshalKey("voice", cfg); err != nil {
        return nil, fmt.Errorf("unmarshal voice config: %w", err)
    }

    // Validate root and all embedded configs
    validate := validator.New()
    if err := validate.Struct(cfg); err != nil {
        return nil, fmt.Errorf("validate config: %w", err)
    }

    return cfg, nil
}

func DefaultConfig() *Config {
    return &Config{
        SessionTimeout:  5 * time.Minute,
        MaxConcurrent:   100,
        EnableTelemetry: true,
        STT: stt.Config{
            SampleRate: 16000,
            Language:   "en-US",
            Encoding:   "linear16",
        },
        TTS: tts.Config{
            SampleRate: 22050,
        },
        VAD: vad.Config{
            Threshold:          0.5,
            MinSilenceDuration: 300 * time.Millisecond,
        },
    }
}
```

### Sub-Package Receives Config

```go
// pkg/voice/voice.go
package voice

func NewVoiceAgent(cfg *Config) (*VoiceAgent, error) {
    // Validate root config
    if err := validate.Struct(cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    // Pass embedded configs to sub-packages
    sttProvider, err := stt.NewProvider(ctx, cfg.STT.Provider, &cfg.STT)
    if err != nil {
        return nil, fmt.Errorf("create stt provider: %w", err)
    }

    ttsProvider, err := tts.NewProvider(ctx, cfg.TTS.Provider, &cfg.TTS)
    if err != nil {
        return nil, fmt.Errorf("create tts provider: %w", err)
    }

    return &VoiceAgent{
        stt:    sttProvider,
        tts:    ttsProvider,
        config: cfg,
    }, nil
}
```

## Validation

### Hierarchical Validation

```go
// Validate at each level
func (c *Config) Validate() error {
    validate := validator.New()

    // Validate root config
    if err := validate.Struct(c); err != nil {
        return fmt.Errorf("root config: %w", err)
    }

    // Validate each sub-package config
    if err := c.STT.Validate(); err != nil {
        return fmt.Errorf("stt config: %w", err)
    }

    if err := c.TTS.Validate(); err != nil {
        return fmt.Errorf("tts config: %w", err)
    }

    return nil
}
```

### Sub-Package Config Validation

```go
// pkg/voice/stt/config.go
func (c *Config) Validate() error {
    validate := validator.New()

    if err := validate.Struct(c); err != nil {
        return err
    }

    // Provider-specific validation
    switch c.Provider {
    case "deepgram":
        if c.DeepgramOptions != nil {
            return c.DeepgramOptions.Validate()
        }
    case "openai":
        if c.OpenAIOptions != nil {
            return c.OpenAIOptions.Validate()
        }
    }

    return nil
}
```

## Standardized Config Keys

Use consistent naming across all packages:

| Key | Description | Type |
|-----|-------------|------|
| `provider` | Provider name | string |
| `api_key` | API authentication key | string |
| `timeout` | Operation timeout | duration |
| `max_retries` | Maximum retry attempts | int |
| `sample_rate` | Audio sample rate (voice) | int |
| `model` | Model identifier | string |

## Related Standards

- [wrapper-package-pattern.md](./wrapper-package-pattern.md) - Wrapper package patterns
- [config-duplication.md](./config-duplication.md) - Avoiding config duplication
- [backend/registry-shape.md](../backend/registry-shape.md) - Registry config handling
