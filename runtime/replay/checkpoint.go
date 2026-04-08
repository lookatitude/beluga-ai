package replay

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// Checkpoint captures the complete state of an agent session at a specific
// point in time. It records the session ID, the conversation turns up to that
// point, arbitrary state, and all agent events that occurred.
type Checkpoint struct {
	// ID is the unique identifier for this checkpoint.
	ID string

	// SessionID identifies the session this checkpoint belongs to.
	SessionID string

	// TurnIndex is the zero-based index of the turn this checkpoint was
	// taken after. A value of -1 indicates a checkpoint before any turns.
	TurnIndex int

	// Turns contains the ordered conversation turns up to this checkpoint.
	Turns []schema.Turn

	// State holds arbitrary session-level state at the time of capture.
	State map[string]any

	// Events contains all agent events recorded up to this checkpoint.
	Events []schema.AgentEvent

	// Timestamp is when this checkpoint was created.
	Timestamp time.Time
}

// CheckpointStore is the interface for persisting and retrieving checkpoints.
// Implementations must be safe for concurrent use.
type CheckpointStore interface {
	// Save persists a checkpoint. If a checkpoint with the same ID already
	// exists, it is overwritten.
	Save(ctx context.Context, cp *Checkpoint) error

	// Get retrieves a checkpoint by its ID. Returns an error if not found.
	Get(ctx context.Context, id string) (*Checkpoint, error)

	// List returns all checkpoint IDs for a given session, ordered by
	// TurnIndex ascending.
	List(ctx context.Context, sessionID string) ([]string, error)

	// Delete removes a checkpoint by its ID. Returns an error if not found.
	Delete(ctx context.Context, id string) error
}
