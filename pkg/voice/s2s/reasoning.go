package s2s

// ReasoningMode represents the reasoning mode for S2S providers.
type ReasoningMode string

const (
	// ReasoningModeBuiltIn uses the provider's built-in reasoning capabilities.
	// The provider handles reasoning internally without external agent integration.
	ReasoningModeBuiltIn ReasoningMode = "built-in"

	// ReasoningModeExternal uses external Beluga AI agents for reasoning.
	// Audio is routed through external agents for custom reasoning logic.
	ReasoningModeExternal ReasoningMode = "external"
)

// IsValid checks if the reasoning mode is valid.
func (rm ReasoningMode) IsValid() bool {
	return rm == ReasoningModeBuiltIn || rm == ReasoningModeExternal
}

// String returns the string representation of the reasoning mode.
func (rm ReasoningMode) String() string {
	return string(rm)
}

// ToConfigString converts ReasoningMode to the config string format.
func (rm ReasoningMode) ToConfigString() string {
	return string(rm)
}

// ParseReasoningMode parses a string into a ReasoningMode.
func ParseReasoningMode(s string) ReasoningMode {
	mode := ReasoningMode(s)
	if mode.IsValid() {
		return mode
	}
	// Default to built-in if invalid
	return ReasoningModeBuiltIn
}
