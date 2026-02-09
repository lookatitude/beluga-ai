---
title: "Embedding Providers"
description: "Embedding provider implementations: OpenAI, Cohere, Google, Jina, Mistral, Ollama, Voyage, and more"
---

## cohere

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
```

Package cohere provides a Cohere embeddings provider for the Beluga AI framework.
It implements the [embedding.Embedder] interface using the Cohere Embed API
via the internal httpclient.

## Registration

The provider registers as "cohere" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"

emb, err := embedding.New("cohere", config.ProviderConfig{
    APIKey: "...",
})
```

## Models

Supported models and their default dimensions:
- embed-english-v3.0 (1024, default)
- embed-multilingual-v3.0 (1024)
- embed-english-light-v3.0 (384)
- embed-multilingual-light-v3.0 (384)
- embed-english-v2.0 (4096)

## Configuration

ProviderConfig fields:
- APIKey — Cohere API key (required)
- Model — model name (default: "embed-english-v3.0")
- BaseURL — API base URL (default: "https://api.cohere.com/v2")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality
- Options["input_type"] — input type hint (default: "search_document")

---

## google

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"
```

Package google provides a Google AI embeddings provider for the Beluga AI framework.
It implements the [embedding.Embedder] interface using the internal httpclient
to call the Google AI Gemini embedding API.

## Registration

The provider registers as "google" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"

emb, err := embedding.New("google", config.ProviderConfig{
    APIKey: "...",
})
```

## Models

Supported models and their default dimensions:
- text-embedding-004 (768, default)
- embedding-001 (768)
- text-multilingual-embedding-002 (768)

## Configuration

ProviderConfig fields:
- APIKey — Google AI API key (required)
- Model — model name (default: "text-embedding-004")
- BaseURL — API base URL (default: "https://generativelanguage.googleapis.com/v1beta")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality

---

## inmemory

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"
```

Package inmemory provides a deterministic hash-based Embedder for testing.
It generates reproducible embeddings by hashing the input text with FNV-1a,
making it suitable for unit tests and local development without external API
calls. The resulting vectors are normalized to unit length.

## Registration

The provider registers as "inmemory" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"

emb, err := embedding.New("inmemory", config.ProviderConfig{})
```

## Configuration

ProviderConfig fields:
- Options["dimensions"] — vector size (default: 128)

---

## jina

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"
```

Package jina provides a Jina AI embeddings provider for the Beluga AI framework.
It implements the [embedding.Embedder] interface using the Jina Embeddings API
via the internal httpclient.

## Registration

The provider registers as "jina" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"

emb, err := embedding.New("jina", config.ProviderConfig{
    APIKey: "...",
})
```

## Models

Supported models and their default dimensions:
- jina-embeddings-v2-base-en (768, default)
- jina-embeddings-v2-small-en (512)
- jina-embeddings-v2-base-de (768)
- jina-embeddings-v2-base-zh (768)
- jina-embeddings-v3 (1024)

## Configuration

ProviderConfig fields:
- APIKey — Jina AI API key (required)
- Model — model name (default: "jina-embeddings-v2-base-en")
- BaseURL — API base URL (default: "https://api.jina.ai/v1")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality

---

## mistral

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"
```

Package mistral provides an Embedder backed by the Mistral AI embeddings API.
It implements the [embedding.Embedder] interface using Mistral's embed endpoint
via the internal httpclient.

## Registration

The provider registers as "mistral" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"

emb, err := embedding.New("mistral", config.ProviderConfig{
    APIKey: "...",
    Model:  "mistral-embed",
})
```

## Models

Supported models and their default dimensions:
- mistral-embed (1024, default)

## Configuration

ProviderConfig fields:
- APIKey — Mistral AI API key (required)
- Model — model name (default: "mistral-embed")
- BaseURL — API base URL (default: "https://api.mistral.ai/v1")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality

---

## ollama

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
```

Package ollama provides an Ollama embeddings provider for the Beluga AI framework.
It implements the [embedding.Embedder] interface using the Ollama REST API
via the internal httpclient. Ollama enables running embedding models locally
without external API calls.

## Registration

The provider registers as "ollama" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"

emb, err := embedding.New("ollama", config.ProviderConfig{
    BaseURL: "http://localhost:11434",
})
```

## Models

Supported models and their default dimensions:
- nomic-embed-text (768, default)
- mxbai-embed-large (1024)
- all-minilm (384)
- snowflake-arctic-embed (1024)

## Configuration

ProviderConfig fields:
- Model — model name (default: "nomic-embed-text")
- BaseURL — Ollama server URL (default: "http://localhost:11434")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality

---

## openai

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
```

Package openai provides an OpenAI embeddings provider for the Beluga AI framework.
It implements the [embedding.Embedder] interface using the openai-go SDK.

## Registration

The provider registers as "openai" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"

emb, err := embedding.New("openai", config.ProviderConfig{
    APIKey: "sk-...",
})
```

## Models

Supported models and their default dimensions:
- text-embedding-3-small (1536, default)
- text-embedding-3-large (3072)
- text-embedding-ada-002 (1536)

## Configuration

ProviderConfig fields:
- APIKey — OpenAI API key (required)
- Model — model name (default: "text-embedding-3-small")
- BaseURL — API base URL (default: "https://api.openai.com/v1")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality

---

## sentence_transformers

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers"
```

Package sentencetransformers provides an Embedder backed by the HuggingFace
Inference API for Sentence Transformers models. It implements the
[embedding.Embedder] interface using the feature-extraction pipeline endpoint.

## Registration

The provider registers as "sentence_transformers" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers"

emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
    APIKey: "hf_...",
    Model:  "sentence-transformers/all-MiniLM-L6-v2",
})
```

## Models

Supported models and their default dimensions:
- sentence-transformers/all-MiniLM-L6-v2 (384, default)
- sentence-transformers/all-MiniLM-L12-v2 (384)
- sentence-transformers/all-mpnet-base-v2 (768)
- sentence-transformers/paraphrase-MiniLM-L6-v2 (384)
- BAAI/bge-small-en-v1.5 (384)
- BAAI/bge-base-en-v1.5 (768)
- BAAI/bge-large-en-v1.5 (1024)

## Configuration

ProviderConfig fields:
- APIKey — HuggingFace API token (required)
- Model — model name (default: "sentence-transformers/all-MiniLM-L6-v2")
- BaseURL — API base URL (default: "https://api-inference.huggingface.co")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality

---

## voyage

```go
import "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"
```

Package voyage provides a Voyage AI embeddings provider for the Beluga AI framework.
It implements the [embedding.Embedder] interface using the Voyage Embed API
via the internal httpclient.

## Registration

The provider registers as "voyage" in the embedding registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"

emb, err := embedding.New("voyage", config.ProviderConfig{
    APIKey: "...",
})
```

## Models

Supported models and their default dimensions:
- voyage-2 (1024, default)
- voyage-large-2 (1536)
- voyage-code-2 (1536)
- voyage-lite-02-instruct (1024)
- voyage-3 (1024)
- voyage-3-lite (512)

## Configuration

ProviderConfig fields:
- APIKey — Voyage AI API key (required)
- Model — model name (default: "voyage-2")
- BaseURL — API base URL (default: "https://api.voyageai.com/v1")
- Timeout — request timeout
- Options["dimensions"] — override output dimensionality
- Options["input_type"] — input type hint (default: "document")
