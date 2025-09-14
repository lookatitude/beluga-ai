package react

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/stretchr/testify/assert"
)

func TestReActAgentInterfaceCompliance(t *testing.T) {
	// This test verifies that ReActAgent implements the required interfaces
	// The assignment below will fail to compile if ReActAgent doesn't implement CompositeAgent
	var _ iface.CompositeAgent = (*ReActAgent)(nil)

	// Test passes if compilation succeeds
	assert.True(t, true, "ReActAgent implements CompositeAgent interface")
}

func TestReActAgentGetConfig(t *testing.T) {
	// This test verifies that GetConfig method is accessible
	var agent *ReActAgent = nil

	// This should compile without errors
	_ = func() {
		if agent != nil {
			_ = agent.GetConfig()
		}
	}
}
