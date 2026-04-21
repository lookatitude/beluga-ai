// Package providers is a side-effect-only package that triggers init()
// registration for the curated set of providers shipped with the beluga CLI.
//
// The providers listed here MUST be CGo-free. CGO_ENABLED=0 is set in the
// goreleaser build; any CGo dependency will silently break cross-compilation
// on the CI runner. Each addition to this list requires an explicit audit —
// check the provider's imports (and its transitive SDK imports) for `import
// "C"` before adding it here. See
// docs/consultations/2026-04-19-loo-142-architect-plan.md (risks for
// reviewer-security).
package providers

import (
	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/anthropic"          // anthropic LLM provider
	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/ollama"              // ollama LLM provider
	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/openai"              // openai LLM provider
	_ "github.com/lookatitude/beluga-ai/v2/memory/stores/inmemory"            // in-memory message store
	_ "github.com/lookatitude/beluga-ai/v2/rag/embedding/providers/ollama"    // ollama embedding provider
	_ "github.com/lookatitude/beluga-ai/v2/rag/embedding/providers/openai"    // openai embedding provider
	_ "github.com/lookatitude/beluga-ai/v2/rag/vectorstore/providers/inmemory" // in-memory vector store
)
