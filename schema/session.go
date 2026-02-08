package schema

import "time"

// Session represents a conversation session containing a sequence of turns.
// Sessions track the full conversation history and maintain arbitrary state
// across turns.
type Session struct {
	// ID is the unique identifier for this session.
	ID string
	// Turns contains the ordered sequence of conversation turns.
	Turns []Turn
	// State holds arbitrary session-level state accessible across turns.
	State map[string]any
	// CreatedAt is the timestamp when the session was created.
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the session was last modified.
	UpdatedAt time.Time
}

// Turn represents a single input-output exchange within a session.
type Turn struct {
	// Input is the user's message that initiated this turn.
	Input Message
	// Output is the agent's response to the input.
	Output Message
	// Timestamp is when this turn occurred.
	Timestamp time.Time
	// Metadata holds arbitrary key-value pairs associated with this turn.
	Metadata map[string]any
}
