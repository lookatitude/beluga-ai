// Package iface defines option and configuration interfaces for the core package.
// T004: Move Option interface to iface/option.go while preserving existing imports
package iface

// Option represents a functional option for configuring operations.
// This interface is used throughout the framework for flexible configuration.
// Note: Using pointer to map to maintain compatibility with existing code.
type Option interface {
	// Apply applies the option to the given configuration map.
	Apply(config *map[string]any)
}

// OptionFunc is a function type that implements the Option interface.
// This allows ordinary functions to be used as options.
type OptionFunc func(*map[string]any)

// Apply implements the Option interface for OptionFunc.
func (f OptionFunc) Apply(config *map[string]any) {
	f(config)
}

// ConfigValidator defines the interface for validating configuration maps.
type ConfigValidator interface {
	// Validate checks if the configuration is valid and returns an error if not.
	Validate(config map[string]any) error
}

// OptionApplier defines the interface for types that can have options applied to them.
type OptionApplier interface {
	// ApplyOptions applies a list of options to the component's configuration.
	ApplyOptions(options ...Option) error
}

// ConfigurableComponent combines OptionApplier with health checking.
type ConfigurableComponent interface {
	OptionApplier
	AdvancedHealthChecker
}
