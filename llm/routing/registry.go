package routing

import (
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// RouterFactory constructs a CostRouter from a RouterConfig.
type RouterFactory func(cfg RouterConfig) (CostRouter, error)

// ClassifierFactory constructs a ComplexityClassifier from a ClassifierConfig.
type ClassifierFactory func(cfg ClassifierConfig) (ComplexityClassifier, error)

// EnforcerFactory constructs a BudgetEnforcer from an EnforcerConfig.
type EnforcerFactory func(cfg EnforcerConfig) (BudgetEnforcer, error)

// RouterConfig is the named-provider configuration for routers.
type RouterConfig struct {
	Models       []ModelConfig
	Classifier   ComplexityClassifier
	Enforcer     BudgetEnforcer
	FallbackTier ModelTier
}

// ClassifierConfig is the named-provider configuration for classifiers.
type ClassifierConfig struct {
	// Params holds classifier-specific parameters.
	Params map[string]any
}

// EnforcerConfig is the named-provider configuration for enforcers.
type EnforcerConfig struct {
	DailyLimit float64
	// Params holds enforcer-specific parameters.
	Params map[string]any
}

var (
	routerMu         sync.RWMutex
	routerRegistry   = make(map[string]RouterFactory)
	classifierMu     sync.RWMutex
	classifierReg    = make(map[string]ClassifierFactory)
	enforcerMu       sync.RWMutex
	enforcerRegistry = make(map[string]EnforcerFactory)
)

// RegisterRouter registers a router factory under name.
func RegisterRouter(name string, f RouterFactory) {
	routerMu.Lock()
	defer routerMu.Unlock()
	routerRegistry[name] = f
}

// NewRouter constructs a CostRouter by name.
func NewRouter(name string, cfg RouterConfig) (CostRouter, error) {
	routerMu.RLock()
	f, ok := routerRegistry[name]
	routerMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "routing: unknown router %q (registered: %v)", name, ListRouters())
	}
	return f(cfg)
}

// ListRouters returns the registered router names, sorted.
func ListRouters() []string {
	routerMu.RLock()
	defer routerMu.RUnlock()
	names := make([]string, 0, len(routerRegistry))
	for n := range routerRegistry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RegisterClassifier registers a classifier factory under name.
func RegisterClassifier(name string, f ClassifierFactory) {
	classifierMu.Lock()
	defer classifierMu.Unlock()
	classifierReg[name] = f
}

// NewClassifier constructs a ComplexityClassifier by name.
func NewClassifier(name string, cfg ClassifierConfig) (ComplexityClassifier, error) {
	classifierMu.RLock()
	f, ok := classifierReg[name]
	classifierMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "routing: unknown classifier %q (registered: %v)", name, ListClassifiers())
	}
	return f(cfg)
}

// ListClassifiers returns the registered classifier names, sorted.
func ListClassifiers() []string {
	classifierMu.RLock()
	defer classifierMu.RUnlock()
	names := make([]string, 0, len(classifierReg))
	for n := range classifierReg {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RegisterEnforcer registers a budget enforcer factory under name.
func RegisterEnforcer(name string, f EnforcerFactory) {
	enforcerMu.Lock()
	defer enforcerMu.Unlock()
	enforcerRegistry[name] = f
}

// NewEnforcer constructs a BudgetEnforcer by name.
func NewEnforcer(name string, cfg EnforcerConfig) (BudgetEnforcer, error) {
	enforcerMu.RLock()
	f, ok := enforcerRegistry[name]
	enforcerMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "routing: unknown enforcer %q (registered: %v)", name, ListEnforcers())
	}
	return f(cfg)
}

// ListEnforcers returns the registered enforcer names, sorted.
func ListEnforcers() []string {
	enforcerMu.RLock()
	defer enforcerMu.RUnlock()
	names := make([]string, 0, len(enforcerRegistry))
	for n := range enforcerRegistry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func init() {
	RegisterRouter("default", func(cfg RouterConfig) (CostRouter, error) {
		opts := []Option{WithModels(cfg.Models...)}
		if cfg.Classifier != nil {
			opts = append(opts, WithClassifier(cfg.Classifier))
		}
		if cfg.Enforcer != nil {
			opts = append(opts, WithBudgetEnforcer(cfg.Enforcer))
		}
		if cfg.FallbackTier != "" {
			opts = append(opts, WithFallbackTier(cfg.FallbackTier))
		}
		return NewCostRouter(opts...), nil
	})
	RegisterClassifier("heuristic", func(_ ClassifierConfig) (ComplexityClassifier, error) {
		return &HeuristicClassifier{}, nil
	})
	RegisterEnforcer("inmemory", func(cfg EnforcerConfig) (BudgetEnforcer, error) {
		return NewBudgetEnforcer(cfg.DailyLimit), nil
	})
}
