// Package consolidation implements memory consolidation with intentional
// forgetting for the Beluga AI memory subsystem.
//
// It provides a background worker that periodically evaluates stored memory
// records using configurable policies (threshold-based, frequency-based, or
// composite) and either prunes low-utility records or compresses them via an
// LLM-backed summarisation step. Utility scoring combines recency (exponential
// half-life decay), importance, relevance, and emotional salience into a
// single composite score.
//
// The consolidation loop follows a ticker-plus-jitter pattern for distributed
// friendliness and implements the core.Lifecycle interface (Start/Stop/Health)
// for integration with the application lifecycle manager.
package consolidation
