package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// AgentConfig represents the configuration for a single agent.
type AgentConfig struct {
	Type         string                 `json:"type" yaml:"type"`
	Name         string                 `json:"name" yaml:"name"`
	Role         string                 `json:"role" yaml:"role"`
	Settings     map[string]interface{} `json:"settings" yaml:"settings"`
	MaxRetries   int                    `json:"max_retries" yaml:"max_retries"`
	RetryDelay   int                    `json:"retry_delay" yaml:"retry_delay"`
	Dependencies []string               `json:"dependencies" yaml:"dependencies"`
	Description  string                 `json:"description" yaml:"description"`
}

// AgentConfigMap maps agent names to their configurations.
type AgentConfigMap map[string]*AgentConfig

// AgentModuleConfig represents the configuration for the entire agent module.
type AgentModuleConfig struct {
	Agents            []*AgentConfig      `json:"agents" yaml:"agents"`
	DefaultSettings   map[string]interface{} `json:"default_settings" yaml:"default_settings"`
	LoggingConfig     map[string]interface{} `json:"logging" yaml:"logging"`
	HealthCheckConfig map[string]interface{} `json:"health_check" yaml:"health_check"`
	WorkflowConfig    map[string]interface{} `json:"workflow" yaml:"workflow"`
}

// ConfigManager handles loading and accessing agent configurations.
type ConfigManager struct {
	config          *AgentModuleConfig
	agentConfigMap  AgentConfigMap
	envVarOverrides map[string]string
	configPath      string
}

// NewConfigManager creates a new configuration manager instance.
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config:          &AgentModuleConfig{},
		agentConfigMap:  make(AgentConfigMap),
		envVarOverrides: make(map[string]string),
	}
}

// LoadConfig loads agent configuration from a JSON or YAML file.
func (cm *ConfigManager) LoadConfig(filePath string) error {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file type and parse
	var config AgentModuleConfig
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Store the config
	cm.config = &config
	cm.configPath = filePath

	// Build the agent config map for easy lookups
	for _, agent := range config.Agents {
		cm.agentConfigMap[agent.Name] = agent
	}

	// Load environment variable overrides
	cm.loadEnvVarOverrides()

	return nil
}

// loadEnvVarOverrides loads configuration overrides from environment variables.
// The format is BELUGA_AGENT_[AGENT_NAME]_[SETTING]
func (cm *ConfigManager) loadEnvVarOverrides() {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if !strings.HasPrefix(key, "BELUGA_AGENT_") {
			continue
		}

		// Strip the prefix
		key = strings.TrimPrefix(key, "BELUGA_AGENT_")
		parts = strings.Split(key, "_")

		if len(parts) < 2 {
			continue
		}

		// Extract agent name and setting name
		agentName := parts[0]
		settingName := strings.Join(parts[1:], "_")

		cm.envVarOverrides[agentName+"_"+settingName] = value
	}
}

// GetAgentConfig retrieves the configuration for a specific agent.
func (cm *ConfigManager) GetAgentConfig(agentName string) (*AgentConfig, error) {
	config, exists := cm.agentConfigMap[agentName]
	if !exists {
		return nil, fmt.Errorf("configuration for agent %s not found", agentName)
	}

	// Create a copy to avoid modifying the original
	configCopy := *config

	// Apply environment variable overrides
	for key, value := range cm.envVarOverrides {
		parts := strings.SplitN(key, "_", 2)
		if len(parts) == 2 && parts[0] == agentName {
			settingName := parts[1]
			
			// Handle different settings
			switch settingName {
			case "MAX_RETRIES":
				if val, err := parseInt(value); err == nil {
					configCopy.MaxRetries = val
				}
			case "RETRY_DELAY":
				if val, err := parseInt(value); err == nil {
					configCopy.RetryDelay = val
				}
			case "TYPE":
				configCopy.Type = value
			case "ROLE":
				configCopy.Role = value
			default:
				// For other settings, add to the settings map
				if configCopy.Settings == nil {
					configCopy.Settings = make(map[string]interface{})
				}
				configCopy.Settings[strings.ToLower(settingName)] = value
			}
		}
	}

	return &configCopy, nil
}

// GetAllAgentConfigs returns a slice of all agent configurations.
func (cm *ConfigManager) GetAllAgentConfigs() []*AgentConfig {
	configs := make([]*AgentConfig, len(cm.config.Agents))
	for i, agent := range cm.config.Agents {
		// Apply any overrides
		config, _ := cm.GetAgentConfig(agent.Name)
		configs[i] = config
	}
	return configs
}

// GetDefaultSettings returns the default settings for all agents.
func (cm *ConfigManager) GetDefaultSettings() map[string]interface{} {
	// Return a copy to prevent modification of the original
	settings := make(map[string]interface{})
	for k, v := range cm.config.DefaultSettings {
		settings[k] = v
	}
	return settings
}

// GetLoggingConfig returns the logging configuration.
func (cm *ConfigManager) GetLoggingConfig() map[string]interface{} {
	// Return a copy to prevent modification of the original
	config := make(map[string]interface{})
	for k, v := range cm.config.LoggingConfig {
		config[k] = v
	}
	return config
}

// GetHealthCheckConfig returns the health check configuration.
func (cm *ConfigManager) GetHealthCheckConfig() map[string]interface{} {
	// Return a copy to prevent modification of the original
	config := make(map[string]interface{})
	for k, v := range cm.config.HealthCheckConfig {
		config[k] = v
	}
	return config
}

// SaveConfig saves the current configuration to a file.
func (cm *ConfigManager) SaveConfig(filePath string) error {
	if filePath == "" {
		filePath = cm.configPath
	}
	
	if filePath == "" {
		return fmt.Errorf("no file path provided and no previous config file loaded")
	}

	// Determine file format based on extension
	var data []byte
	var err error
	
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cm.config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Helper function to parse an integer from a string
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}