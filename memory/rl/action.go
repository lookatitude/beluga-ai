package rl

import "fmt"

// MemoryAction represents a discrete action the RL policy can select when
// processing a new memory observation. The four actions correspond to the
// Memory-R1 action space.
type MemoryAction int

const (
	// ActionAdd instructs the memory to store the new content as a fresh entry.
	ActionAdd MemoryAction = iota

	// ActionUpdate instructs the memory to replace the most similar existing
	// entry with the new content.
	ActionUpdate

	// ActionDelete instructs the memory to remove the most similar existing
	// entry (the new content is not stored).
	ActionDelete

	// ActionNoop instructs the memory to take no action, discarding the new
	// content as redundant or low-value.
	ActionNoop
)

// String returns a human-readable name for the action.
func (a MemoryAction) String() string {
	switch a {
	case ActionAdd:
		return "add"
	case ActionUpdate:
		return "update"
	case ActionDelete:
		return "delete"
	case ActionNoop:
		return "noop"
	default:
		return fmt.Sprintf("unknown(%d)", int(a))
	}
}

// Valid reports whether the action is one of the four defined actions.
func (a MemoryAction) Valid() bool {
	return a >= ActionAdd && a <= ActionNoop
}

// NumActions is the total number of discrete actions in the action space.
const NumActions = 4
