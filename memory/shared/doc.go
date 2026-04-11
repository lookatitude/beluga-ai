// Package shared provides cross-agent shared memory with access control,
// provenance tracking, and conflict resolution.
//
// Shared memory enables multiple agents to read and write named fragments
// of data with fine-grained access control. Each fragment has a scope
// (Private, Team, or Global), a list of authorized readers and writers,
// and an immutable provenance chain that records who wrote what and when.
//
// # Fragments
//
// A [Fragment] is the unit of shared memory. It carries a key, content,
// scope, access lists, and versioning metadata. Fragments support three
// conflict resolution policies:
//
//   - [AppendOnly]: new content is appended to existing content.
//   - [LastWriteWins]: the latest write replaces the previous content.
//   - [RejectOnConflict]: writes to an existing fragment fail if the
//     caller's expected version does not match the current version.
//
// # Access Control
//
// Every fragment has Readers and Writers lists. An empty list means the
// fragment is unrestricted for that operation. When lists are non-empty,
// only agent IDs present in the list are permitted. Access is checked
// by [SharedMemory] before delegating to the underlying [SharedStore].
//
// # Provenance
//
// Each write produces a [Provenance] record containing a SHA-256 content
// hash, the author ID, a timestamp, and the hash of the previous version.
// Callers can verify the integrity of a fragment's lineage via
// [Provenance.Verify].
//
// # Usage
//
//	store := shared.NewInMemorySharedStore()
//	sm := shared.NewSharedMemory(store,
//	    shared.WithDefaultScope(shared.ScopeTeam),
//	    shared.WithConflictPolicy(shared.LastWriteWins),
//	    shared.WithProvenanceEnabled(true),
//	)
//
//	err := sm.Write(ctx, &shared.Fragment{
//	    Key:      "plan",
//	    Content:  "Step 1: gather data",
//	    AuthorID: "agent-1",
//	    Writers:  []string{"agent-1", "agent-2"},
//	    Readers:  []string{"agent-1", "agent-2", "agent-3"},
//	})
//
//	frag, err := sm.Read(ctx, "plan", "agent-2")
//
// # Hooks
//
// Lifecycle hooks are available via [WithHooks]. Multiple hooks compose
// with [ComposeHooks]. All hook fields are optional; nil means skip.
package shared
