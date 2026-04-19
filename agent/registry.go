package agent

import (
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
)

// PlannerConfig holds configuration for creating a planner via the registry.
type PlannerConfig struct {
	// LLM is the language model the planner will use for reasoning.
	LLM llm.ChatModel
	// Extra holds planner-specific configuration.
	Extra map[string]any
}

// PlannerFactory is a constructor function for creating a Planner from config.
type PlannerFactory func(cfg PlannerConfig) (Planner, error)

var (
	plannerMu       sync.RWMutex
	plannerRegistry = make(map[string]PlannerFactory)
)

// RegisterPlanner registers a planner factory under the given name.
// This is typically called from init() in planner implementation files.
func RegisterPlanner(name string, factory PlannerFactory) {
	plannerMu.Lock()
	defer plannerMu.Unlock()
	plannerRegistry[name] = factory
}

// NewPlanner creates a new planner by looking up the registered factory.
func NewPlanner(name string, cfg PlannerConfig) (Planner, error) {
	plannerMu.RLock()
	factory, ok := plannerRegistry[name]
	plannerMu.RUnlock()

	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "planner %q not registered", name)
	}
	return factory(cfg)
}

// ListPlanners returns the sorted names of all registered planners.
func ListPlanners() []string {
	plannerMu.RLock()
	defer plannerMu.RUnlock()

	names := make([]string, 0, len(plannerRegistry))
	for name := range plannerRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
