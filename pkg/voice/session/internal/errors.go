package internal

import "fmt"

// Helper function to create session errors (to avoid import cycle)
func newSessionError(op, code string, err error) error {
	// Return a simple error for now - in real implementation, use proper error type
	return fmt.Errorf("session %s: %s: %w", op, code, err)
}
