# External Dependencies Requiring Mocks

**Generated**: $(date -u +%Y-%m-%dT%H:%M:%SZ)
**Purpose**: Identify all external dependencies that require mock implementations for testing

## LLM Provider APIs

### pkg/llms
- **OpenAI API** (`pkg/llms/providers/openai`)
  - HTTP client calls
  - API authentication
  - Streaming responses
  - Rate limiting

- **Anthropic API** (`pkg/llms/providers/anthropic`)
  - HTTP client calls
  - API authentication
  - Streaming responses

- **AWS Bedrock** (`pkg/llms/providers/bedrock`)
  - AWS SDK calls
  - AWS authentication
  - Service endpoints

- **Ollama** (`pkg/llms/providers/ollama`)
  - HTTP client calls
  - Local API endpoints

### pkg/chatmodels
- **LLM Provider Dependencies** (via pkg/llms)
  - All LLM provider APIs listed above

## Embedding Provider APIs

### pkg/embeddings
- **OpenAI Embeddings API** (`pkg/embeddings/providers/openai`)
  - HTTP client calls
  - API authentication

- **Cohere Embeddings API** (`pkg/embeddings/providers/cohere`)
  - HTTP client calls
  - API authentication

- **Google Multimodal Embeddings** (`pkg/embeddings/providers/google_multimodal`)
  - Google Cloud API calls
  - Authentication

- **Ollama Embeddings** (`pkg/embeddings/providers/ollama`)
  - HTTP client calls
  - Local API endpoints

## Vector Store APIs

### pkg/vectorstores
- **PGVector** (`pkg/vectorstores/providers/pgvector`)
  - PostgreSQL database connections
  - SQL queries
  - Connection pooling

- **InMemory VectorStore** (`pkg/vectorstores/providers/inmemory`)
  - No external dependencies (in-memory only)

- **Other Vector Store Providers**
  - Database connections
  - API calls
  - Authentication

## Messaging Backends

### pkg/messaging
- **Message Queue Backends**
  - RabbitMQ connections
  - Redis pub/sub
  - Kafka connections
  - AWS SQS

## Voice Provider APIs

### pkg/voice
- **Twilio API** (`pkg/voice/providers/twilio`)
  - Twilio REST API calls
  - WebSocket connections
  - Authentication tokens

- **STT (Speech-to-Text) Providers**
  - External STT service APIs
  - Audio processing

- **TTS (Text-to-Speech) Providers**
  - External TTS service APIs
  - Audio generation

## File I/O Operations

### pkg/documentloaders
- **File System Operations**
  - File reading
  - Directory traversal
  - File metadata

- **Remote Document Sources**
  - HTTP/HTTPS requests
  - URL fetching

## Database Connections

### pkg/memory (Redis)
- **Redis Connections** (`pkg/memory/internal/redis`)
  - Redis client connections
  - Redis commands
  - Connection pooling

## Network Operations

### pkg/server
- **HTTP Server**
  - HTTP request handling
  - WebSocket connections
  - Server lifecycle

## Summary

### High Priority (Required for Unit Tests)
1. LLM Provider APIs (OpenAI, Anthropic, Bedrock, Ollama)
2. Embedding Provider APIs (OpenAI, Cohere, Google, Ollama)
3. Vector Store APIs (PGVector, other databases)
4. Voice Provider APIs (Twilio, STT, TTS)
5. Messaging Backends (RabbitMQ, Redis, Kafka, SQS)

### Medium Priority (Integration Tests)
1. Database connections (PostgreSQL, Redis)
2. File system operations
3. HTTP client operations

### Low Priority (May Use Real Connections in Integration Tests)
1. Network operations (can be tested with test servers)
2. File I/O (can use test fixtures)

## Mock Implementation Status

- ✅ **pkg/llms**: Has AdvancedMockLLM in test_utils.go
- ✅ **pkg/embeddings**: Has AdvancedMockEmbedder in test_utils.go
- ✅ **pkg/vectorstores**: Has AdvancedMockVectorStore in test_utils.go
- ✅ **pkg/chatmodels**: Has AdvancedMockChatModel in test_utils.go
- ✅ **pkg/memory**: Has AdvancedMockMemory in test_utils.go
- ✅ **pkg/messaging**: Has AdvancedMockMessagingBackend in test_utils.go
- ✅ **pkg/orchestration**: Has AdvancedMockOrchestrator in test_utils.go
- ✅ **pkg/prompts**: Has AdvancedMockPromptManager in test_utils.go
- ✅ **pkg/retrievers**: Has AdvancedMockRetriever in test_utils.go
- ✅ **pkg/server**: Has AdvancedMockServer in test_utils.go
- ✅ **pkg/multimodal**: Has AdvancedMockMultimodal in test_utils.go

**Action Items**:
1. Enhance existing mocks to support all error types
2. Create provider-specific mocks where needed
3. Document mock usage patterns
