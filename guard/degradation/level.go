package degradation

// AutonomyLevel represents the degree of freedom an agent has during
// execution. Levels are ordered from most permissive to most restrictive.
type AutonomyLevel int

const (
	// Full allows unrestricted access to all tools and capabilities.
	Full AutonomyLevel = iota

	// Restricted limits tool execution to an explicit allowlist.
	Restricted

	// ReadOnly prohibits all tool calls and write operations.
	ReadOnly

	// Sequestered fully isolates the agent; actions are logged but never
	// executed.
	Sequestered
)

// String returns a human-readable representation of the autonomy level.
func (l AutonomyLevel) String() string {
	switch l {
	case Full:
		return "full"
	case Restricted:
		return "restricted"
	case ReadOnly:
		return "read_only"
	case Sequestered:
		return "sequestered"
	default:
		return "unknown"
	}
}

// Capabilities describes what actions are permitted at a given autonomy level.
type Capabilities struct {
	// CanExecuteTools indicates whether tool execution is permitted.
	CanExecuteTools bool

	// ToolsAllowlisted indicates whether only allowlisted tools may run.
	// Only meaningful when CanExecuteTools is true.
	ToolsAllowlisted bool

	// CanWrite indicates whether write operations are permitted.
	CanWrite bool

	// CanRespond indicates whether the agent may produce responses.
	CanRespond bool
}

// LevelCapabilities returns the capabilities associated with a given
// autonomy level.
func LevelCapabilities(level AutonomyLevel) Capabilities {
	switch level {
	case Full:
		return Capabilities{
			CanExecuteTools:  true,
			ToolsAllowlisted: false,
			CanWrite:         true,
			CanRespond:       true,
		}
	case Restricted:
		return Capabilities{
			CanExecuteTools:  true,
			ToolsAllowlisted: true,
			CanWrite:         true,
			CanRespond:       true,
		}
	case ReadOnly:
		return Capabilities{
			CanExecuteTools:  false,
			ToolsAllowlisted: false,
			CanWrite:         false,
			CanRespond:       true,
		}
	default:
		// Sequestered and unknown levels are treated as fully isolated for
		// safety: no tools, no writes, no responses.
		return Capabilities{
			CanExecuteTools:  false,
			ToolsAllowlisted: false,
			CanWrite:         false,
			CanRespond:       false,
		}
	}
}
