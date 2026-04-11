package shared

import "time"

// Scope defines the visibility level for a shared memory fragment.
type Scope string

const (
	// ScopePrivate limits the fragment to the owning agent.
	ScopePrivate Scope = "private"

	// ScopeTeam makes the fragment visible to agents in the same team.
	ScopeTeam Scope = "team"

	// ScopeGlobal makes the fragment visible to all agents.
	ScopeGlobal Scope = "global"
)

// ConflictPolicy determines how concurrent writes to the same key are resolved.
type ConflictPolicy string

const (
	// AppendOnly concatenates new content to existing content.
	AppendOnly ConflictPolicy = "append_only"

	// LastWriteWins replaces the existing content with the new content.
	LastWriteWins ConflictPolicy = "last_write_wins"

	// RejectOnConflict rejects writes when the fragment's version does not
	// match the caller's expected version.
	RejectOnConflict ConflictPolicy = "reject_on_conflict"
)

// Permission represents an access type for shared memory operations.
type Permission string

const (
	// PermRead allows reading a fragment.
	PermRead Permission = "read"

	// PermWrite allows writing a fragment.
	PermWrite Permission = "write"
)

// Fragment is the unit of shared memory. It carries content, access
// control lists, versioning metadata, and provenance information.
type Fragment struct {
	// ID is a unique identifier for the fragment, typically assigned by the store.
	ID string

	// Key is the human-readable name used to look up the fragment.
	Key string

	// Content is the fragment's payload.
	Content string

	// Scope controls the visibility level of the fragment.
	Scope Scope

	// AuthorID identifies the agent that last wrote this fragment.
	AuthorID string

	// Version is incremented on each successful write.
	Version int64

	// Provenance records the cryptographic lineage of writes.
	Provenance *Provenance

	// ConflictPolicy determines how concurrent writes are resolved.
	ConflictPolicy ConflictPolicy

	// Readers lists agent IDs allowed to read this fragment.
	// An empty list means unrestricted read access.
	Readers []string

	// Writers lists agent IDs allowed to write this fragment.
	// An empty list means unrestricted write access.
	Writers []string

	// CreatedAt is the time the fragment was first created.
	CreatedAt time.Time

	// UpdatedAt is the time of the most recent write.
	UpdatedAt time.Time

	// Metadata holds arbitrary key-value data associated with the fragment.
	Metadata map[string]string
}

// FragmentChange describes a mutation to a fragment, delivered via Watch.
type FragmentChange struct {
	// Key is the affected fragment key.
	Key string

	// Fragment is the new state after the change (nil for deletes).
	Fragment *Fragment

	// Op indicates the type of change.
	Op ChangeOp
}

// ChangeOp describes the type of fragment mutation.
type ChangeOp string

const (
	// OpWrite indicates a fragment was created or updated.
	OpWrite ChangeOp = "write"

	// OpDelete indicates a fragment was removed.
	OpDelete ChangeOp = "delete"

	// OpGrant indicates permissions were granted on a fragment.
	OpGrant ChangeOp = "grant"

	// OpRevoke indicates permissions were revoked on a fragment.
	OpRevoke ChangeOp = "revoke"
)
