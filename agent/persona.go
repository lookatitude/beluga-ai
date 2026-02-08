package agent

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/schema"
)

// Persona defines the identity and behavior of an agent using the RGB
// (Role, Goal, Backstory) framework. Traits provide additional personality
// characteristics.
type Persona struct {
	// Role describes what the agent is (e.g., "senior software engineer").
	Role string
	// Goal describes the agent's objective (e.g., "help users write clean code").
	Goal string
	// Backstory provides additional context about the agent's background.
	Backstory string
	// Traits are personality characteristics (e.g., "concise", "friendly").
	Traits []string
}

// ToSystemMessage converts the persona into a system message suitable for
// inclusion in the LLM conversation. Empty fields are omitted.
func (p Persona) ToSystemMessage() *schema.SystemMessage {
	var parts []string

	if p.Role != "" {
		parts = append(parts, fmt.Sprintf("You are a %s.", p.Role))
	}
	if p.Goal != "" {
		parts = append(parts, fmt.Sprintf("Your goal is to %s.", p.Goal))
	}
	if p.Backstory != "" {
		parts = append(parts, p.Backstory)
	}
	if len(p.Traits) > 0 {
		parts = append(parts, fmt.Sprintf("Your traits: %s.", strings.Join(p.Traits, ", ")))
	}

	if len(parts) == 0 {
		return nil
	}

	return schema.NewSystemMessage(strings.Join(parts, "\n"))
}

// IsEmpty reports whether the persona has no content.
func (p Persona) IsEmpty() bool {
	return p.Role == "" && p.Goal == "" && p.Backstory == "" && len(p.Traits) == 0
}
