package interfaces

// Agent defines the interface for all agents in the system.
type Agent interface {
	// Initialize sets up the agent with necessary configurations.
	Initialize(config map[string]interface{}) error

	// Execute performs the main task of the agent.
	Execute() error

	// Shutdown gracefully stops the agent and cleans up resources.
	Shutdown() error
}