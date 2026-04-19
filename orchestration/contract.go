package orchestration

import (
	"strings"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/agent/contract"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// ValidatePipelineContracts checks that the contracts of agents in a pipeline
// are compatible in sequence. Returns nil if all contracts are compatible or if
// agents do not have contracts. Returns an error listing all incompatibilities.
func ValidatePipelineContracts(agents ...agent.Agent) error {
	errs := contract.CheckPipelineCompatibility(agents)
	if len(errs) == 0 {
		return nil
	}

	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return core.Errorf(core.ErrInvalidInput, "orchestration/contract: pipeline incompatibilities: %s", strings.Join(msgs, "; "))
}
