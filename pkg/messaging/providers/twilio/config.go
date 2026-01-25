package twilio

import (
	"errors"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
)

// TwilioConfig extends the base Config with Twilio-specific fields.
type TwilioConfig struct {
	*messaging.Config

	// Twilio credentials
	AccountSID string `mapstructure:"account_sid" yaml:"account_sid" env:"TWILIO_ACCOUNT_SID" validate:"required"`
	AuthToken  string `mapstructure:"auth_token" yaml:"auth_token" env:"TWILIO_AUTH_TOKEN" validate:"required"`

	// API configuration
	APIVersion string `mapstructure:"api_version" yaml:"api_version" env:"TWILIO_CONVERSATIONS_API_VERSION" default:"v1"`
	BaseURL    string `mapstructure:"base_url" yaml:"base_url" env:"TWILIO_BASE_URL" default:"https://conversations.twilio.com"`

	// Webhook configuration
	WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url" env:"TWILIO_WEBHOOK_URL"`

	// Multi-account support (FR-041)
	AccountName string `mapstructure:"account_name" yaml:"account_name" env:"TWILIO_ACCOUNT_NAME"` // Optional identifier for multiple accounts

	// Retry configuration
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" env:"TWILIO_MAX_RETRIES" default:"3"`
}

// NewTwilioConfig creates a new Twilio configuration from base config.
func NewTwilioConfig(baseConfig *messaging.Config) *TwilioConfig {
	config := &TwilioConfig{
		Config:     baseConfig,
		APIVersion: "v1",
		BaseURL:    "https://conversations.twilio.com",
		MaxRetries: 3,
	}

	// Extract Twilio-specific config from ProviderSpecific if present
	if baseConfig.ProviderSpecific != nil {
		if accountSID, ok := baseConfig.ProviderSpecific["account_sid"].(string); ok {
			config.AccountSID = accountSID
		}
		if authToken, ok := baseConfig.ProviderSpecific["auth_token"].(string); ok {
			config.AuthToken = authToken
		}
		if webhookURL, ok := baseConfig.ProviderSpecific["webhook_url"].(string); ok {
			config.WebhookURL = webhookURL
		}
		if accountName, ok := baseConfig.ProviderSpecific["account_name"].(string); ok {
			config.AccountName = accountName
		}
		if apiVersion, ok := baseConfig.ProviderSpecific["api_version"].(string); ok {
			config.APIVersion = apiVersion
		}
		if baseURL, ok := baseConfig.ProviderSpecific["base_url"].(string); ok {
			config.BaseURL = baseURL
		}
		if maxRetries, ok := baseConfig.ProviderSpecific["max_retries"].(int); ok {
			config.MaxRetries = maxRetries
		}
	}

	return config
}

// Validate validates the Twilio configuration.
func (c *TwilioConfig) Validate() error {
	if c.AccountSID == "" {
		return messaging.NewMessagingError("Validate", messaging.ErrCodeInvalidConfig, errors.New("account_sid is required"))
	}
	if c.AuthToken == "" {
		return messaging.NewMessagingError("Validate", messaging.ErrCodeInvalidConfig, errors.New("auth_token is required"))
	}
	return nil
}
