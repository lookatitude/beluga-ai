package agents

import (
	"beluga/pkg/interfaces"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// AgentConfig represents the configuration structure for an agent.
type AgentConfig struct {
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	Role           string                 `json:"role"`
	Settings       map[string]interface{} `json:"settings"`
	MaxRetries     int                    `json:"max_retries"`
	RetryDelay     int                    `json:"retry_delay"`
	Dependencies   []string               `json:"dependencies"`
	Description    string                 `json:"description"`
}

// AgentRegistry maintains a registry of all created agents for reference and management.
type AgentRegistry struct {
	Agents map[string]interfaces.Agent
}

// NewAgentRegistry creates a new AgentRegistry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		Agents: make(map[string]interfaces.Agent),
	}
}

// RegisterAgent adds an agent to the registry.
func (r *AgentRegistry) RegisterAgent(name string, agent interfaces.Agent) {
	r.Agents[name] = agent
}

// GetAgent retrieves an agent from the registry by name.
func (r *AgentRegistry) GetAgent(name string) (interfaces.Agent, bool) {
	agent, exists := r.Agents[name]
	return agent, exists
}

// ListAgents returns a list of all registered agent names.
func (r *AgentRegistry) ListAgents() []string {
	agentNames := make([]string, 0, len(r.Agents))
	for name := range r.Agents {
		agentNames = append(agentNames, name)
	}
	return agentNames
}

// AgentFactory is responsible for creating agents dynamically.
type AgentFactory struct {
	Registry *AgentRegistry
}

// NewAgentFactory creates and returns a new instance of AgentFactory.
func NewAgentFactory() *AgentFactory {
	return &AgentFactory{
		Registry: NewAgentRegistry(),
	}
}

// CreateAgentFromConfig creates an agent based on the provided configuration.
func (f *AgentFactory) CreateAgentFromConfig(config *AgentConfig) (interfaces.Agent, error) {
	agent, err := f.CreateAgent(config.Type, config.Name, config.Settings)
	if err != nil {
		return nil, err
	}
	
	// Register the agent
	f.Registry.RegisterAgent(config.Name, agent)
	return agent, nil
}

// CreateAgent creates an agent based on the provided type and name.
func (f *AgentFactory) CreateAgent(agentType, name string, config map[string]interface{}) (interfaces.Agent, error) {
	var agent interfaces.Agent
	
	switch agentType {
	case "DataFetcherAgent":
		dataSource := getStringParam(config, "data_source", "default")
		dataFormat := getStringParam(config, "data_format", "json")
		
		dataFetcher := NewDataFetcherAgent(name, dataSource, dataFormat)
		if err := dataFetcher.Initialize(config); err != nil {
			return nil, fmt.Errorf("failed to initialize DataFetcherAgent: %w", err)
		}
		agent = dataFetcher
		
	case "AnalyzerAgent":
		analysisType := getStringParam(config, "analysis_type", "basic")
		
		analyzer := NewAnalyzerAgent(name, analysisType)
		if err := analyzer.Initialize(config); err != nil {
			return nil, fmt.Errorf("failed to initialize AnalyzerAgent: %w", err)
		}
		agent = analyzer
		
	case "DecisionMakerAgent":
		decisionMaker := NewDecisionMakerAgent(name)
		if err := decisionMaker.Initialize(config); err != nil {
			return nil, fmt.Errorf("failed to initialize DecisionMakerAgent: %w", err)
		}
		agent = decisionMaker
		
	case "ExecutorAgent":
		action := getStringParam(config, "action", "default_action")
		target := getStringParam(config, "target", "default_target")
		
		executor := NewExecutorAgent(name, action, target)
		if err := executor.Initialize(config); err != nil {
			return nil, fmt.Errorf("failed to initialize ExecutorAgent: %w", err)
		}
		agent = executor
		
	case "MonitorAgent":
		interval := time.Duration(getIntParam(config, "interval_seconds", 60)) * time.Second
		
		monitor := NewMonitorAgent(name, interval)
		if err := monitor.Initialize(config); err != nil {
			return nil, fmt.Errorf("failed to initialize MonitorAgent: %w", err)
		}
		
		// Add monitor targets if specified
		if targets, ok := config["monitor_targets"].([]interface{}); ok {
			for _, target := range targets {
				if targetStr, ok := target.(string); ok {
					monitor.AddMonitorTarget(targetStr)
				}
			}
		}
		
		agent = monitor
		
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
	
	// Register the created agent
	f.Registry.RegisterAgent(name, agent)
	
	return agent, nil
}

// LoadAgentsFromConfig loads and creates agents from a configuration file.
func (f *AgentFactory) LoadAgentsFromConfig(configPath string) ([]interfaces.Agent, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config file: %w", err)
	}
	
	var configs []AgentConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse agent config JSON: %w", err)
	}
	
	agents := make([]interfaces.Agent, 0, len(configs))
	for _, config := range configs {
		agent, err := f.CreateAgentFromConfig(&config)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent from config: %w", err)
		}
		agents = append(agents, agent)
	}
	
	return agents, nil
}

// LoadAgentsFromDirectory loads and creates agents from all config files in a directory.
func (f *AgentFactory) LoadAgentsFromDirectory(dirPath string) ([]interfaces.Agent, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	var agents []interfaces.Agent
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		ext := filepath.Ext(file.Name())
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}
		
		path := filepath.Join(dirPath, file.Name())
		agentsFromFile, err := f.LoadAgentsFromConfig(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load agents from %s: %w", path, err)
		}
		
		agents = append(agents, agentsFromFile...)
	}
	
	return agents, nil
}

// Helper functions for parameter extraction
func getStringParam(config map[string]interface{}, key, defaultValue string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return defaultValue
}

func getIntParam(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key].(float64); ok {
		return int(val)
	}
	if val, ok := config[key].(int); ok {
		return val
	}
	return defaultValue
}