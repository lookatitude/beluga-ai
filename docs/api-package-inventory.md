# API Package Inventory

**Date**: 2025-01-27  
**Purpose**: Comprehensive inventory of all packages, sub-packages, and their documentation status

## Main Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| agents | `pkg/agents` | ✅ Documented | Main agents package |
| chatmodels | `pkg/chatmodels` | ✅ Documented | Chat model interfaces |
| config | `pkg/config` | ✅ Documented | Configuration management |
| core | `pkg/core` | ✅ Documented | Core utilities and error handling |
| documentloaders | `pkg/documentloaders` | ✅ Documented | Document loading from files and directories |
| embeddings | `pkg/embeddings` | ✅ Documented | Embedding interfaces |
| llms | `pkg/llms` | ✅ Documented | LLM interfaces |
| memory | `pkg/memory` | ✅ Documented | Memory management |
| monitoring | `pkg/monitoring` | ✅ Documented | Observability and metrics |
| orchestration | `pkg/orchestration` | ✅ Documented | Agent orchestration |
| prompts | `pkg/prompts` | ✅ Documented | Prompt management |
| retrievers | `pkg/retrievers` | ✅ Documented | Document retrieval |
| schema | `pkg/schema` | ✅ Documented | Data schemas |
| server | `pkg/server` | ✅ Documented | HTTP server |
| textsplitters | `pkg/textsplitters` | ✅ Documented | Text splitting for RAG pipelines |
| vectorstores | `pkg/vectorstores` | ✅ Documented | Vector storage |

## LLM Provider Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| anthropic | `pkg/llms/providers/anthropic` | ✅ Documented | Anthropic Claude provider |
| bedrock | `pkg/llms/providers/bedrock` | ✅ Documented | AWS Bedrock provider |
| cohere | `pkg/llms/providers/cohere` | ❌ Not Found | Listed in script but directory doesn't exist |
| ollama | `pkg/llms/providers/ollama` | ✅ Documented | Ollama provider |
| openai | `pkg/llms/providers/openai` | ✅ Documented | OpenAI provider |
| mock | `pkg/llms/providers/mock` | ❌ Not Documented | Mock provider exists but not in script |

## Voice Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| stt | `pkg/stt` | ✅ Documented | Speech-to-Text |
| tts | `pkg/tts` | ✅ Documented | Text-to-Speech |
| vad | `pkg/vad` | ✅ Documented | Voice Activity Detection |
| turndetection | `pkg/turndetection` | ✅ Documented | Turn Detection |
| transport | `pkg/audiotransport` | ✅ Documented | Transport layer |
| noise | `pkg/noisereduction` | ✅ Documented | Noise cancellation |
| session | `pkg/voicesession` | ✅ Documented | Session management |

## Tools Package

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| tools | `pkg/tools` | ✅ Documented | Agent tools interfaces |

## Embedding Provider Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| mock | `pkg/embeddings/providers/mock` | ❌ Not Documented | Mock embedding provider |
| ollama | `pkg/embeddings/providers/ollama` | ❌ Not Documented | Ollama embedding provider |
| openai | `pkg/embeddings/providers/openai` | ❌ Not Documented | OpenAI embedding provider |

## Vectorstore Provider Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| inmemory | `pkg/vectorstores/providers/inmemory` | ❌ Not Documented | In-memory vectorstore |
| pgvector | `pkg/vectorstores/providers/pgvector` | ❌ Not Documented | PostgreSQL pgvector provider |

## Config Provider Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| composite | `pkg/config/providers/composite` | ❌ Not Documented | Composite config provider |
| viper | `pkg/config/providers/viper` | ❌ Not Documented | Viper config provider |

## Chatmodel Provider Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| openai | `pkg/chatmodels/providers/openai` | ❌ Not Documented | OpenAI chatmodel provider |

## Agent Provider Packages

| Package | Path | Documentation Status | Notes |
|---------|------|----------------------|-------|
| react | `pkg/agents/providers/react` | ❌ Not Documented | ReAct agent provider |

## Internal/Interface Packages (Not Documented)

These packages are internal implementation details and should not be documented as public API:

- `pkg/*/iface` - Interface definitions (internal)
- `pkg/*/internal` - Internal implementation details
- `pkg/voice/benchmarks` - Benchmark code
- `pkg/embeddings/testdata` - Test data
- `pkg/embeddings/testutils` - Test utilities
- `pkg/core/model` - Internal models
- `pkg/core/utils` - Internal utilities

## Summary

### Documented Packages
- **Main packages**: 16/16 ✅ (added documentloaders, textsplitters)
- **LLM providers**: 4/5 ✅ (cohere missing, mock not in script)
- **Voice packages**: 7/7 ✅
- **Tools**: 1/1 ✅
- **Total documented**: 28 packages

### Missing from Documentation
- **Embedding providers**: 3 packages (mock, ollama, openai)
- **Vectorstore providers**: 2 packages (inmemory, pgvector)
- **Config providers**: 2 packages (composite, viper)
- **Chatmodel providers**: 1 package (openai)
- **Agent providers**: 1 package (react)
- **LLM providers**: 1 package (mock - exists but not in script)
- **Total missing**: 10 packages

### Script Issues
- `cohere` LLM provider listed in script but directory doesn't exist
- `mock` LLM provider exists but not listed in script
- Provider sub-packages (embeddings, vectorstores, config, chatmodels, agents) not included in script

## Recommendations

1. **Remove non-existent package**: Remove `cohere` from LLM_PROVIDERS array in `scripts/generate-docs.sh`
2. **Add missing LLM provider**: Add `mock` to LLM_PROVIDERS array
3. **Add provider sub-packages**: Consider adding provider packages for embeddings, vectorstores, config, chatmodels, and agents if they should be documented
4. **Documentation decision**: Determine if provider sub-packages should be public API documentation or remain internal
