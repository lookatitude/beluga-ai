package computeruse

import (
	"context"
)

// ActionType identifies the kind of computer action.
type ActionType string

const (
	// ActionScreenshot captures the current screen state.
	ActionScreenshot ActionType = "screenshot"

	// ActionClick performs a mouse click at coordinates.
	ActionClick ActionType = "click"

	// ActionType_ performs keyboard text input.
	ActionType_ ActionType = "type"

	// ActionScroll performs a scroll action.
	ActionScroll ActionType = "scroll"

	// ActionKeyPress sends a key press event.
	ActionKeyPress ActionType = "key_press"
)

// ActionRequest describes a computer action to perform.
type ActionRequest struct {
	// Type is the kind of action to perform.
	Type ActionType

	// X is the horizontal coordinate for click/scroll actions.
	X int

	// Y is the vertical coordinate for click/scroll actions.
	Y int

	// Text is the text to type for type actions, or the key name for key_press.
	Text string

	// ScrollDelta is the scroll amount (positive = down, negative = up).
	ScrollDelta int
}

// ActionResult holds the outcome of a computer action.
type ActionResult struct {
	// Screenshot is the screen capture after the action, as PNG bytes.
	// May be nil for non-screenshot actions if the backend does not
	// automatically capture.
	Screenshot []byte

	// Description is a human-readable description of the action result.
	Description string

	// Success indicates whether the action completed without error.
	Success bool
}

// ComputerAction is the interface for performing screen interactions.
// Implementations drive actual computer or browser automation.
type ComputerAction interface {
	// Execute performs the requested action and returns the result.
	Execute(ctx context.Context, req ActionRequest) (*ActionResult, error)

	// Screenshot captures the current screen state as PNG bytes.
	Screenshot(ctx context.Context) ([]byte, error)
}
