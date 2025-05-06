// Package config handles loading and accessing application configuration
// using Viper, supporting environment variables and config files.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the Beluga-ai framework.
// Tags are used by Viper to map config file keys and environment variables.
type Config struct {
	LLMs struct {
		OpenAI struct {
			APIKey  string `mapstructure:"api_key"`
			BaseURL string `mapstructure:"base_url"`
			Model   string `mapstructure:"model"`
		} `mapstructure:"openai"`
		Anthropic struct {
			APIKey  string `mapstructure:"api_key"`
			BaseURL string `mapstructure:"base_url"`
			Version string `mapstructure:"version"`
			Model   string `mapstructure:"model"`
		} `mapstructure:"anthropic"`
		Ollama struct {
			BaseURL string `mapstructure:"base_url"`
			Model   string `mapstructure:"model"`
		} `mapstructure:"ollama"`
		Bedrock struct {
			Region    string `mapstructure:"region"`
			AccessKey string `mapstructure:"access_key"` // Consider secure ways to handle credentials
			SecretKey string `mapstructure:"secret_key"` // Consider secure ways to handle credentials
			ModelID   string `mapstructure:"model_id"`   // Keep ModelID for specific Bedrock model selection
		} `mapstructure:"bedrock"`
		Gemini struct {
			APIKey string `mapstructure:"api_key"`
			Model  string `mapstructure:"model"`
		} `mapstructure:"gemini"`
		Cohere struct {
			APIKey string `mapstructure:"api_key"`
			Model  string `mapstructure:"model"`
		} `mapstructure:"cohere"`
	} `mapstructure:"llms"`

	// Add other configuration sections as needed (e.g., RAG, Agents, Orchestration)
	// Example:
	// RAG struct {
	// 	 PGVector struct {
	// 	 	 ConnectionString string `mapstructure:"connection_string"`
	// 	 } `mapstructure:"pgvector"`
	// } `mapstructure:"rag"`
}

var Cfg Config

// LoadConfig reads configuration from file and environment variables.
func LoadConfig(configPaths ...string) error {
	 v := viper.New()

	 // Set default values
	 v.SetDefault("llms.openai.model", "gpt-4o")
	 v.SetDefault("llms.anthropic.model", "claude-3-haiku-20240307")
	 v.SetDefault("llms.anthropic.version", "2023-06-01")
	 v.SetDefault("llms.ollama.base_url", "http://localhost:11434")
	 v.SetDefault("llms.ollama.model", "llama3")
	 v.SetDefault("llms.bedrock.region", "us-east-1")
	 v.SetDefault("llms.gemini.model", "gemini-1.5-flash-latest")
	 v.SetDefault("llms.cohere.model", "command-r") // Default Cohere model
	 // Add more defaults here

	 // Set config file paths
	 v.SetConfigName("config") // name of config file (without extension)
	 v.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	 // Add paths to search for the config file
	 v.AddConfigPath(".") // Current directory
	 v.AddConfigPath("/etc/beluga-ai/") // Path for system-wide config
	 v.AddConfigPath("$HOME/.beluga-ai") // Path for user-specific config
	 for _, path := range configPaths {
	 	 v.AddConfigPath(path)
	 }

	 // Read config file (optional)
	 if err := v.ReadInConfig(); err != nil {
	 	 if _, ok := err.(viper.ConfigFileNotFoundError); ok {
	 	 	 // Config file not found; ignore error if desired
	 	 	 fmt.Println("Config file not found, using defaults and environment variables.")
	 	 } else {
	 	 	 // Config file was found but another error was produced
	 	 	 return fmt.Errorf("error reading config file: %w", err)
	 	 }
	 }

	 // Enable environment variable overriding
	 v.SetEnvPrefix("BELUGA") // e.g., BELUGA_LLMS_OPENAI_APIKEY
	 v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	 v.AutomaticEnv()

	 // Unmarshal the config into the Cfg struct
	 if err := v.Unmarshal(&Cfg); err != nil {
	 	 return fmt.Errorf("unable to decode config into struct: %w", err)
	 }

	 // Optionally: Validate configuration values here
	 // if Cfg.LLMs.Cohere.APIKey == "" { ... }

	 return nil
}

