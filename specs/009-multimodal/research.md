# Research: Multimodal Models Support

**Feature**: Multimodal Models Support  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This document consolidates research findings for implementing multimodal model support in the Beluga AI Framework. It addresses technical decisions, provider capabilities, integration patterns, and best practices.

## Research Areas

### 1. Multimodal Provider Capabilities

**Decision**: Support both paid/closed-source and open-source multimodal providers with capability detection and graceful fallbacks.

**Rationale**: 
- Paid providers (OpenAI, Google, Anthropic, xAI) offer mature APIs with strong multimodal support
- Open-source providers (Qwen, Pixtral, Phi, DeepSeek, Gemma) provide cost-effective alternatives
- Capability detection enables routing content to appropriate providers
- Graceful fallbacks ensure text-only workflows continue when multimodal isn't available

**Alternatives Considered**:
- **Provider-specific packages**: Rejected - would fragment the codebase and make provider swapping difficult
- **Single provider only**: Rejected - limits user choice and framework value
- **No capability detection**: Rejected - would cause errors when providers don't support modalities

**Implementation Notes**:
- Each provider must implement capability detection (SupportsImage(), SupportsVideo(), SupportsAudio())
- Router component will check capabilities before routing content
- Fallback strategy: text-only processing if multimodal not supported

### 2. Content Block Format Standardization

**Decision**: Support multiple formats (base64, URLs, file paths) with normalization to provider-preferred formats.

**Rationale**:
- Different providers accept different formats (base64 for some, URLs for others)
- Users may provide content in various formats
- Normalization enables cross-provider compatibility
- Base64 is most universal but has size limitations

**Alternatives Considered**:
- **Single format only (base64)**: Rejected - URLs are more efficient for large files, file paths needed for local processing
- **No normalization**: Rejected - would require users to know provider-specific formats
- **Provider-specific formats only**: Rejected - would complicate user experience

**Implementation Notes**:
- ContentBlock interface with Format() method
- Normalizer component converts between formats
- Size limits: base64 for <10MB, URLs for larger files
- MIME type detection and validation

### 3. Integration with Existing Packages

**Decision**: Build on existing multimodal foundations (MultimodalEmbedder, ImageMessage, VideoMessage, VoiceDocument) and integrate through composition.

**Rationale**:
- Existing schema package already has multimodal message types
- Existing embeddings package has MultimodalEmbedder interface
- Composition avoids duplication and maintains consistency
- Integration points are well-defined

**Alternatives Considered**:
- **New schema types**: Rejected - would duplicate existing types and break compatibility
- **Replace existing interfaces**: Rejected - would break backward compatibility
- **Separate multimodal package with no integration**: Rejected - would fragment functionality

**Implementation Notes**:
- Use existing schema.ImageMessage, schema.VideoMessage, schema.VoiceDocument
- Extend MultimodalEmbedder interface if needed (currently sufficient)
- Compose with llms package for text processing
- Compose with voice package for audio processing
- Integrate with vectorstores for multimodal RAG

### 4. Provider Registry Pattern

**Decision**: Use global registry pattern matching existing packages (llms, embeddings, vectorstores).

**Rationale**:
- Consistency with framework patterns
- Enables easy provider swapping
- Auto-registration via init() functions
- Thread-safe with sync.Once

**Alternatives Considered**:
- **Factory pattern only**: Rejected - registry provides better discoverability
- **Manual registration**: Rejected - auto-registration is more user-friendly
- **Per-instance registries**: Rejected - global registry is simpler and matches framework

**Implementation Notes**:
- Global registry with GetRegistry() function
- Provider registration in providers/*/init.go files
- Registry interface: Register(name, factory), Create(ctx, name, config), ListProviders(), IsRegistered(name)
- Factory functions return MultimodalModel interface

### 5. Streaming Support

**Decision**: Support streaming for video/audio with chunk-based processing and incremental results.

**Rationale**:
- Real-time processing requires streaming
- Large files need chunking to avoid memory issues
- Incremental results improve user experience
- Provider APIs support streaming (OpenAI, Google, etc.)

**Alternatives Considered**:
- **No streaming**: Rejected - required for real-time use cases
- **Full buffering**: Rejected - causes memory issues and latency
- **Provider-specific streaming only**: Rejected - needs unified interface

**Implementation Notes**:
- Streaming interface: StreamProcess(ctx, input) <-chan Result
- Chunk size: 1MB for video, 64KB for audio
- Context cancellation support
- Error handling in streaming pipelines

### 6. Multimodal RAG Integration

**Decision**: Extend existing vectorstores with multimodal support through MultimodalEmbedder integration.

**Rationale**:
- Vectorstores already support embeddings
- MultimodalEmbedder interface exists
- Minimal changes to existing code
- Maintains backward compatibility

**Alternatives Considered**:
- **New multimodal vectorstore package**: Rejected - would duplicate functionality
- **Modify existing vectorstores directly**: Rejected - would break backward compatibility
- **Separate RAG package**: Rejected - RAG is a use case, not a separate package

**Implementation Notes**:
- Use MultimodalEmbedder for multimodal document embeddings
- Store multimodal vectors alongside text vectors
- Support multimodal queries (text+image search)
- Fuse multimodal and text results in retrieval

### 7. Agent Integration

**Decision**: Extend agents with multimodal capabilities through composition and interface extensions.

**Rationale**:
- Agents already process messages (schema.Message supports multimodal)
- Composition maintains separation of concerns
- Interface extensions enable opt-in multimodal support
- Maintains backward compatibility

**Alternatives Considered**:
- **New multimodal agent package**: Rejected - would fragment agent functionality
- **Modify existing agents directly**: Rejected - would break backward compatibility
- **Separate multimodal processing layer**: Rejected - adds unnecessary complexity

**Implementation Notes**:
- Agents check for multimodal content in messages
- Route multimodal content to appropriate providers
- Support multimodal reasoning in ReAct loops
- Preserve multimodal context in agent state

### 8. Error Handling

**Decision**: Use custom error types following framework patterns (MultimodalError with Op, Err, Code).

**Rationale**:
- Consistency with framework error handling
- Enables programmatic error handling
- Error codes for common failures
- Context cancellation support

**Alternatives Considered**:
- **Standard errors only**: Rejected - loses structured error information
- **Provider-specific errors**: Rejected - would fragment error handling
- **No error codes**: Rejected - error codes enable better error handling

**Implementation Notes**:
- MultimodalError struct with Op, Err, Code, Message fields
- Error codes: ErrCodeInvalidFormat, ErrCodeProviderError, ErrCodeUnsupportedModality, etc.
- Helper functions: NewMultimodalError, WrapError, IsMultimodalError, AsMultimodalError
- Context cancellation checks in all operations

### 9. Observability

**Decision**: Comprehensive OTEL integration (metrics, tracing, logging) following framework patterns.

**Rationale**:
- Framework requires OTEL observability
- Enables production monitoring and debugging
- Consistent with other packages
- Structured logging with trace IDs

**Alternatives Considered**:
- **No observability**: Rejected - required for production use
- **Logging only**: Rejected - metrics and tracing are essential
- **Provider-specific observability**: Rejected - needs unified observability

**Implementation Notes**:
- Metrics: latency histograms, throughput counters, error rates
- Tracing: spans for all public methods with attributes
- Logging: structured logging with OTEL context (trace IDs, span IDs)
- Metrics in metrics.go, tracing in methods, logging via logWithOTELContext

### 10. Performance Requirements

**Decision**: Target latencies: <500ms for voice (p95), <1s for video (p95), <200ms for images (p95).

**Rationale**:
- Based on provider API capabilities
- Interactive workflows require low latency
- Streaming helps achieve these targets
- Benchmarks will verify performance

**Alternatives Considered**:
- **No performance targets**: Rejected - performance is critical for user experience
- **Higher latencies**: Rejected - would degrade user experience
- **Provider-specific targets**: Rejected - unified targets enable consistent experience

**Implementation Notes**:
- Benchmark critical paths
- Optimize content normalization
- Use streaming for large files
- Cache provider capabilities
- Monitor performance in production

## Technical Decisions Summary

| Decision Area | Decision | Rationale |
|--------------|----------|-----------|
| Provider Support | Both paid and open-source with capability detection | Maximum flexibility and choice |
| Content Formats | Multiple formats with normalization | User convenience and cross-provider compatibility |
| Integration | Composition with existing packages | Maintains consistency and backward compatibility |
| Registry Pattern | Global registry matching framework | Consistency and ease of use |
| Streaming | Chunk-based streaming with incremental results | Real-time processing and memory efficiency |
| RAG Integration | Extend vectorstores via MultimodalEmbedder | Minimal changes, maximum compatibility |
| Agent Integration | Composition and interface extensions | Maintains separation of concerns |
| Error Handling | Custom error types with codes | Framework consistency and programmatic handling |
| Observability | Comprehensive OTEL integration | Production requirements |
| Performance | <500ms voice, <1s video, <200ms images (p95) | Interactive workflow requirements |

## Open Questions Resolved

1. **Q**: How to handle provider capability mismatches?  
   **A**: Capability detection with graceful fallbacks to text-only processing.

2. **Q**: What content formats to support?  
   **A**: Base64, URLs, file paths with normalization to provider-preferred formats.

3. **Q**: How to integrate with existing packages?  
   **A**: Composition using existing interfaces (MultimodalEmbedder, Message types).

4. **Q**: What registry pattern to use?  
   **A**: Global registry matching existing packages (llms, embeddings, vectorstores).

5. **Q**: How to support streaming?  
   **A**: Chunk-based streaming with incremental results and context cancellation.

6. **Q**: How to integrate multimodal RAG?  
   **A**: Extend vectorstores via MultimodalEmbedder interface.

7. **Q**: How to extend agents?  
   **A**: Composition and interface extensions maintaining backward compatibility.

8. **Q**: What error handling pattern?  
   **A**: Custom error types with Op, Err, Code following framework patterns.

9. **Q**: What observability requirements?  
   **A**: Comprehensive OTEL (metrics, tracing, logging) following framework patterns.

10. **Q**: What performance targets?  
    **A**: <500ms voice, <1s video, <200ms images (p95) for interactive workflows.

## Next Steps

1. Generate data-model.md with entity definitions
2. Generate contracts/ with API specifications
3. Generate quickstart.md with usage examples
4. Update agent context with multimodal patterns
