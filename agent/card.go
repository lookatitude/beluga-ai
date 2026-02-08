package agent

// AgentCard describes an agent's capabilities for A2A (Agent-to-Agent)
// protocol discovery. Cards are used by external systems to understand
// what an agent can do and how to interact with it.
type AgentCard struct {
	// Name is the human-readable name of the agent.
	Name string `json:"name"`
	// Description explains what the agent does.
	Description string `json:"description"`
	// URL is the endpoint where the agent can be reached.
	URL string `json:"url,omitempty"`
	// Skills lists the agent's capabilities.
	Skills []string `json:"skills,omitempty"`
	// Protocols lists the supported protocols (e.g., "a2a", "mcp").
	Protocols []string `json:"protocols,omitempty"`
}

// BuildCard generates an AgentCard from an agent's metadata.
func BuildCard(a Agent) AgentCard {
	card := AgentCard{
		Name: a.ID(),
	}

	// Extract description from persona
	p := a.Persona()
	if p.Goal != "" {
		card.Description = p.Goal
	} else if p.Role != "" {
		card.Description = p.Role
	}

	// Extract skill names from tools
	for _, t := range a.Tools() {
		card.Skills = append(card.Skills, t.Name())
	}

	return card
}
