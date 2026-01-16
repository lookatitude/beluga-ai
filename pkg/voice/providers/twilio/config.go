package twilio

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// TwilioConfig extends the base Config with Twilio-specific fields.
type TwilioConfig struct {
	*iface.Config

	// Twilio credentials
	AccountSID  string `mapstructure:"account_sid" yaml:"account_sid" env:"TWILIO_ACCOUNT_SID" validate:"required"`
	AuthToken   string `mapstructure:"auth_token" yaml:"auth_token" env:"TWILIO_AUTH_TOKEN" validate:"required"`
	PhoneNumber string `mapstructure:"phone_number" yaml:"phone_number" env:"TWILIO_PHONE_NUMBER" validate:"required"`

	// API configuration
	APIVersion string `mapstructure:"api_version" yaml:"api_version" env:"TWILIO_API_VERSION" default:"2010-04-01"`
	BaseURL    string `mapstructure:"base_url" yaml:"base_url" env:"TWILIO_BASE_URL" default:"https://api.twilio.com"`

	// Webhook configuration
	WebhookURL        string `mapstructure:"webhook_url" yaml:"webhook_url" env:"TWILIO_WEBHOOK_URL"`
	StatusCallbackURL string `mapstructure:"status_callback_url" yaml:"status_callback_url" env:"TWILIO_STATUS_CALLBACK_URL"`

	// Multi-account support (FR-041)
	AccountName string `mapstructure:"account_name" yaml:"account_name" env:"TWILIO_ACCOUNT_NAME"` // Optional identifier for multiple accounts

	// Retry configuration
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" env:"TWILIO_MAX_RETRIES" default:"3"`

	// Connection pooling
	MaxIdleConns    int `mapstructure:"max_idle_conns" yaml:"max_idle_conns" env:"TWILIO_MAX_IDLE_CONNS" default:"100"`
	MaxConnsPerHost int `mapstructure:"max_conns_per_host" yaml:"max_conns_per_host" env:"TWILIO_MAX_CONNS_PER_HOST" default:"10"`

	// S2S (Speech-to-Speech) configuration
	S2SProvider string                 `mapstructure:"s2s_provider" yaml:"s2s_provider" env:"TWILIO_S2S_PROVIDER"`
	S2SConfig   map[string]any         `mapstructure:"s2s_config" yaml:"s2s_config" env:"TWILIO_S2S_CONFIG"`

	// VAD (Voice Activity Detection) configuration
	VADProvider string         `mapstructure:"vad_provider" yaml:"vad_provider" env:"TWILIO_VAD_PROVIDER"`
	VADConfig   map[string]any `mapstructure:"vad_config" yaml:"vad_config" env:"TWILIO_VAD_CONFIG"`

	// Turn Detection configuration
	TurnDetectorProvider string         `mapstructure:"turn_detector_provider" yaml:"turn_detector_provider" env:"TWILIO_TURN_DETECTOR_PROVIDER"`
	TurnDetectorConfig   map[string]any `mapstructure:"turn_detector_config" yaml:"turn_detector_config" env:"TWILIO_TURN_DETECTOR_CONFIG"`

	// Memory configuration (using map[string]any to match backend SessionConfig)
	MemoryConfig map[string]any `mapstructure:"memory_config" yaml:"memory_config" env:"TWILIO_MEMORY_CONFIG"`

	// Noise Cancellation configuration
	NoiseCancellationProvider string         `mapstructure:"noise_cancellation_provider" yaml:"noise_cancellation_provider" env:"TWILIO_NOISE_CANCELLATION_PROVIDER"`
	NoiseCancellationConfig   map[string]any `mapstructure:"noise_cancellation_config" yaml:"noise_cancellation_config" env:"TWILIO_NOISE_CANCELLATION_CONFIG"`
}

// NewTwilioConfig creates a new Twilio configuration from base config.
func NewTwilioConfig(baseConfig *iface.Config) *TwilioConfig {
	config := &TwilioConfig{
		Config:     baseConfig,
		APIVersion: "2010-04-01",
		BaseURL:    "https://api.twilio.com",
		MaxRetries: 3,
	}

	// Extract Twilio-specific config from ProviderConfig if present
	if baseConfig.ProviderConfig != nil {
		if accountSID, ok := baseConfig.ProviderConfig["account_sid"].(string); ok {
			config.AccountSID = accountSID
		}
		if authToken, ok := baseConfig.ProviderConfig["auth_token"].(string); ok {
			config.AuthToken = authToken
		}
		if phoneNumber, ok := baseConfig.ProviderConfig["phone_number"].(string); ok {
			config.PhoneNumber = phoneNumber
		}
		if webhookURL, ok := baseConfig.ProviderConfig["webhook_url"].(string); ok {
			config.WebhookURL = webhookURL
		}
		if statusCallbackURL, ok := baseConfig.ProviderConfig["status_callback_url"].(string); ok {
			config.StatusCallbackURL = statusCallbackURL
		}
		if accountName, ok := baseConfig.ProviderConfig["account_name"].(string); ok {
			config.AccountName = accountName
		}
		if apiVersion, ok := baseConfig.ProviderConfig["api_version"].(string); ok {
			config.APIVersion = apiVersion
		}
		if baseURL, ok := baseConfig.ProviderConfig["base_url"].(string); ok {
			config.BaseURL = baseURL
		}
		if maxRetries, ok := baseConfig.ProviderConfig["max_retries"].(int); ok {
			config.MaxRetries = maxRetries
		}
		if s2sProvider, ok := baseConfig.ProviderConfig["s2s_provider"].(string); ok {
			config.S2SProvider = s2sProvider
		}
		if s2sConfig, ok := baseConfig.ProviderConfig["s2s_config"].(map[string]any); ok {
			config.S2SConfig = s2sConfig
		}
		if vadProvider, ok := baseConfig.ProviderConfig["vad_provider"].(string); ok {
			config.VADProvider = vadProvider
		}
		if vadConfig, ok := baseConfig.ProviderConfig["vad_config"].(map[string]any); ok {
			config.VADConfig = vadConfig
		}
		if turnDetectorProvider, ok := baseConfig.ProviderConfig["turn_detector_provider"].(string); ok {
			config.TurnDetectorProvider = turnDetectorProvider
		}
		if turnDetectorConfig, ok := baseConfig.ProviderConfig["turn_detector_config"].(map[string]any); ok {
			config.TurnDetectorConfig = turnDetectorConfig
		}
		if memoryConfig, ok := baseConfig.ProviderConfig["memory_config"].(map[string]any); ok {
			config.MemoryConfig = memoryConfig
		}
		if noiseCancellationProvider, ok := baseConfig.ProviderConfig["noise_cancellation_provider"].(string); ok {
			config.NoiseCancellationProvider = noiseCancellationProvider
		}
		if noiseCancellationConfig, ok := baseConfig.ProviderConfig["noise_cancellation_config"].(map[string]any); ok {
			config.NoiseCancellationConfig = noiseCancellationConfig
		}
	}

	return config
}

// Validate validates the Twilio configuration.
func (c *TwilioConfig) Validate() error {
	if c.AccountSID == "" {
		return NewTwilioError("Validate", ErrCodeTwilioInvalidConfig, "account_sid is required")
	}
	if c.AuthToken == "" {
		return NewTwilioError("Validate", ErrCodeTwilioInvalidConfig, "auth_token is required")
	}
	if c.PhoneNumber == "" {
		return NewTwilioError("Validate", ErrCodeTwilioInvalidConfig, "phone_number is required")
	}
	return nil
}
