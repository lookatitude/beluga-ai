// Package mock provides a fixture-driven llm.ChatModel implementation for the
// Beluga AI framework. It is registered under the provider name "mock" and is
// the canonical backend for the beluga test dev-loop command and for any local
// development that needs deterministic, network-free LLM behaviour.
//
// Fixtures are loaded (in priority order) from:
//
//  1. The Fixtures field of this package's functional options via the
//     programmatic New constructor.
//  2. cfg.Options["fixtures_file"] — path to a JSON file.
//  3. cfg.Options["fixtures"] — an already-parsed []Fixture or
//     []map[string]any (for JSON unmarshalled config).
//  4. The BELUGA_MOCK_FIXTURES environment variable — path to a JSON file.
//
// When the fixture queue is exhausted the mock returns a final-answer
// AIMessage (text content, no tool calls). Agent planners treat this as a
// finish signal, guaranteeing that a missing fixture cannot dead-lock a
// ReAct loop. The fallback message is configurable via WithFallback.
//
// The registered provider name is "mock". Combined with
// BELUGA_LLM_PROVIDER=mock, BELUGA_DETERMINISTIC=1, BELUGA_SEED=42,
// OTEL_SDK_DISABLED=true, and BELUGA_TEST=1 this provider enables the
// "beluga test" canonical environment described in the DX-1 S3 design
// brief.
package mock
