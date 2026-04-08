// Package memory provides memory poisoning detection and cascading failure
// prevention for multi-agent systems. It detects anomalous content being
// written to shared memory stores, signs memory entries with HMAC-SHA256 to
// ensure integrity, and uses per-agent-pair circuit breakers to isolate
// compromised writers from shared memory.
//
// The package is designed to work with the memory.Middleware interface,
// allowing it to be composed with other middleware in the memory pipeline.
//
// Key components:
//
//   - AnomalyDetector: Interface for detecting suspicious content.
//     Built-in detectors include EntropyDetector, PatternDetector,
//     RateDetector, and SizeDetector.
//
//   - SignedMemoryMiddleware: HMAC-SHA256 signing and verification of
//     memory entries, ensuring tamper detection.
//
//   - InterAgentCircuitBreaker: Per-agent-pair circuit breakers that
//     trip when poisoning is detected, isolating the writer.
//
//   - MemoryGuard: Orchestrates detectors in a pipeline with
//     configurable thresholds and hooks.
package memory
