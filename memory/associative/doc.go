// Package associative implements a Zettelkasten-style associative memory system
// for Beluga AI agents. It models knowledge as interconnected notes that are
// automatically enriched with keywords, tags, and descriptions via an LLM,
// embedded for semantic search, and linked to related notes based on cosine
// similarity.
//
// The system follows the A-MEM (Agentic Memory) architecture with four stages:
//
//  1. Enrichment: An LLM extracts keywords, tags, and a description from raw content.
//  2. Embedding: The enriched note is embedded into a vector space.
//  3. Linking: Top-k most similar existing notes are identified and bidirectional
//     links are created.
//  4. Refinement (optional): Neighbor notes have their keywords, tags, and
//     descriptions retroactively updated by the LLM to reflect the new context
//     introduced by the incoming note.
//
// # Usage
//
// Create an AssociativeMemory with an embedder and optional LLM for enrichment:
//
//	mem := associative.NewAssociativeMemory(embedder,
//	    associative.WithLLM(chatModel),
//	    associative.WithLinkCandidates(10),
//	    associative.WithRetroactiveRefinement(true),
//	    associative.WithMaxTags(8),
//	)
//
//	// Add a note (enrichment + embedding + linking happens automatically)
//	note, err := mem.AddNote(ctx, "The Go programming language uses goroutines for concurrency")
//
//	// Search for related notes
//	notes, err := mem.SearchNotes(ctx, "concurrent programming", 5)
//
// # Memory Interface
//
// AssociativeMemory implements the [memory.Memory] interface, mapping
// Save/Load/Search/Clear to note operations. This allows it to be used
// as a drop-in tier in the composite memory system.
//
// # Hooks
//
// Lifecycle hooks are available for observing note creation, linking, and
// refinement events. Compose multiple hooks with [ComposeHooks].
package associative
