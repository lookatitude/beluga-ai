package contract

import (
	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// ContractProvider is an optional interface that agents may implement to
// expose their semantic contract. Use ContractOf to safely extract the
// contract from any agent.
type ContractProvider interface {
	// Contract returns the agent's semantic contract, or nil if none is set.
	Contract() *schema.Contract
}

// ContractOf extracts the contract from an agent. If the agent implements
// ContractProvider, its Contract() method is called. Otherwise, nil is returned
// (wildcard behavior).
func ContractOf(a agent.Agent) *schema.Contract {
	if cp, ok := a.(ContractProvider); ok {
		return cp.Contract()
	}
	return nil
}
