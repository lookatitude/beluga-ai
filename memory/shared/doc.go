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
// conflict resolution policies, fixed at creation:
//
//   - [AppendOnly]: new content is appended to existing content.
//   - [LastWriteWins]: the latest write replaces the previous content.
//   - [RejectOnConflict]: writes to an existing fragment fail if the
//     caller's expected version does not match the current version.
//
// # Access Control
//
// Every fragment has Readers and Writers lists plus UnrestrictedRead and
// UnrestrictedWrite flags. Only agent IDs present in a list (or every agent,
// when the matching flag is set) are permitted; an empty list without the
// flag denies everyone, so revoking the last reader or writer locks the
// fragment rather than opening it. A fragment created with an empty list and
// no flag defaults to its author only.
//
// Access-control fields and the conflict policy are fixed when a fragment is
// created. Later writes keep them (a content-only update never widens
// access); change access with [SharedMemory.Grant] and [SharedMemory.Revoke].
// A store read failure other than not-found fails the write closed.
// [SharedMemory.WatchAs] authorizes every delivered change against the
// fragment's Readers. Access is checked by [SharedMemory] before delegating
// to the underlying [SharedStore].
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
