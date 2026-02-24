export interface Integration {
  name: string;
  category: string;
  priority: "Core" | "Extended" | "Community";
  description: string;
  docLink?: string;
}

export const categories = [
  "LLM",
  "Embeddings",
  "Vector Stores",
  "Voice STT",
  "Voice TTS",
  "Voice S2S",
  "Memory",
  "Document Loaders",
  "Guardrails",
  "Eval & Observability",
  "Workflows",
  "HTTP / API",
] as const;

export const integrations: Integration[] = [
  // LLM (20)
  { name: "OpenAI", category: "LLM", priority: "Core", description: "GPT-4o, GPT-4, GPT-3.5 with streaming and function calling" },
  { name: "Anthropic", category: "LLM", priority: "Core", description: "Claude 4, Claude 3.5 with prompt caching" },
  { name: "Google Gemini", category: "LLM", priority: "Core", description: "Gemini 2.0, 1.5 Pro/Flash with multimodal" },
  { name: "AWS Bedrock", category: "LLM", priority: "Core", description: "Multi-model access via AWS infrastructure" },
  { name: "Ollama", category: "LLM", priority: "Core", description: "Local model inference for development and edge" },
  { name: "Groq", category: "LLM", priority: "Core", description: "Ultra-fast inference with LPU hardware" },
  { name: "Mistral", category: "LLM", priority: "Extended", description: "Mistral Large, Medium, Small models" },
  { name: "DeepSeek", category: "LLM", priority: "Extended", description: "DeepSeek-V3 and reasoning models" },
  { name: "xAI Grok", category: "LLM", priority: "Extended", description: "Grok models with real-time knowledge" },
  { name: "Cohere", category: "LLM", priority: "Extended", description: "Command R+ with RAG-optimized models" },
  { name: "Together AI", category: "LLM", priority: "Extended", description: "Open-source model hosting and inference" },
  { name: "Fireworks AI", category: "LLM", priority: "Extended", description: "Fast inference for open models" },
  { name: "Azure OpenAI", category: "LLM", priority: "Extended", description: "OpenAI models via Azure with enterprise compliance" },
  { name: "Perplexity", category: "LLM", priority: "Extended", description: "Search-augmented language models" },
  { name: "SambaNova", category: "LLM", priority: "Extended", description: "Enterprise AI inference platform" },
  { name: "Cerebras", category: "LLM", priority: "Extended", description: "Wafer-scale inference engine" },
  { name: "OpenRouter", category: "LLM", priority: "Extended", description: "Multi-provider routing and fallback" },
  { name: "Hugging Face", category: "LLM", priority: "Extended", description: "Inference API for open models" },
  { name: "Vertex AI", category: "LLM", priority: "Extended", description: "Google Cloud AI platform" },
  { name: "AI21", category: "LLM", priority: "Community", description: "Jamba models for enterprise" },

  // Embeddings (8)
  { name: "OpenAI Embeddings", category: "Embeddings", priority: "Core", description: "text-embedding-3-small/large" },
  { name: "Google Embeddings", category: "Embeddings", priority: "Core", description: "Gecko and text-embedding models" },
  { name: "Ollama Embeddings", category: "Embeddings", priority: "Core", description: "Local embedding with any GGUF model" },
  { name: "Cohere Embed", category: "Embeddings", priority: "Extended", description: "Embed v3 with compression" },
  { name: "Voyage AI", category: "Embeddings", priority: "Extended", description: "Domain-specific embeddings" },
  { name: "Jina Embeddings", category: "Embeddings", priority: "Extended", description: "Multilingual embeddings" },
  { name: "Mistral Embed", category: "Embeddings", priority: "Extended", description: "Mistral embedding model" },
  { name: "Sentence Transformers", category: "Embeddings", priority: "Community", description: "HuggingFace sentence transformers" },

  // Vector Stores (12)
  { name: "pgvector", category: "Vector Stores", priority: "Core", description: "PostgreSQL vector extension" },
  { name: "Qdrant", category: "Vector Stores", priority: "Core", description: "High-performance vector database" },
  { name: "Pinecone", category: "Vector Stores", priority: "Core", description: "Managed vector database" },
  { name: "ChromaDB", category: "Vector Stores", priority: "Extended", description: "Open-source embedding database" },
  { name: "Weaviate", category: "Vector Stores", priority: "Extended", description: "Vector search with GraphQL" },
  { name: "Milvus", category: "Vector Stores", priority: "Extended", description: "Scalable vector database" },
  { name: "Turbopuffer", category: "Vector Stores", priority: "Extended", description: "Serverless vector database" },
  { name: "Redis Vector", category: "Vector Stores", priority: "Extended", description: "Redis with vector search" },
  { name: "Elasticsearch", category: "Vector Stores", priority: "Extended", description: "Vector search in Elasticsearch" },
  { name: "MongoDB Atlas", category: "Vector Stores", priority: "Extended", description: "MongoDB with vector search" },
  { name: "SQLite-vec", category: "Vector Stores", priority: "Community", description: "SQLite vector extension" },
  { name: "Vespa", category: "Vector Stores", priority: "Community", description: "Hybrid search engine" },

  // Voice STT (6)
  { name: "Deepgram", category: "Voice STT", priority: "Core", description: "Nova-3 real-time STT" },
  { name: "ElevenLabs Scribe", category: "Voice STT", priority: "Core", description: "High-accuracy transcription" },
  { name: "OpenAI Whisper", category: "Voice STT", priority: "Core", description: "Whisper and Transcribe API" },
  { name: "AssemblyAI", category: "Voice STT", priority: "Extended", description: "Slam-1 universal STT" },
  { name: "Groq STT", category: "Voice STT", priority: "Extended", description: "Fast Whisper inference" },
  { name: "Gladia", category: "Voice STT", priority: "Community", description: "Real-time transcription" },

  // Voice TTS (7)
  { name: "ElevenLabs TTS", category: "Voice TTS", priority: "Core", description: "High-quality voice synthesis" },
  { name: "Cartesia Sonic", category: "Voice TTS", priority: "Core", description: "Low-latency TTS" },
  { name: "PlayHT", category: "Voice TTS", priority: "Extended", description: "AI voice generation" },
  { name: "Groq TTS", category: "Voice TTS", priority: "Extended", description: "Fast text-to-speech" },
  { name: "Fish Audio", category: "Voice TTS", priority: "Extended", description: "Open-source TTS" },
  { name: "LMNT", category: "Voice TTS", priority: "Extended", description: "Ultra-fast voice synthesis" },
  { name: "Smallest.ai", category: "Voice TTS", priority: "Community", description: "Efficient TTS models" },

  // Voice S2S (3)
  { name: "OpenAI Realtime", category: "Voice S2S", priority: "Core", description: "Direct speech-to-speech" },
  { name: "Gemini Live", category: "Voice S2S", priority: "Core", description: "Google multimodal live" },
  { name: "Ultravox", category: "Voice S2S", priority: "Extended", description: "Open speech-language model" },

  // Memory (7)
  { name: "In-Memory", category: "Memory", priority: "Core", description: "Fast in-process memory store" },
  { name: "Redis Memory", category: "Memory", priority: "Core", description: "Distributed memory with persistence" },
  { name: "PostgreSQL Memory", category: "Memory", priority: "Core", description: "Relational memory store" },
  { name: "SQLite Memory", category: "Memory", priority: "Extended", description: "Embedded memory store" },
  { name: "Neo4j", category: "Memory", priority: "Extended", description: "Graph-based memory" },
  { name: "DragonflyDB", category: "Memory", priority: "Community", description: "Redis-compatible memory" },
  { name: "Memgraph", category: "Memory", priority: "Community", description: "Graph memory store" },

  // Document Loaders (8)
  { name: "Firecrawl", category: "Document Loaders", priority: "Core", description: "Web scraping and crawling" },
  { name: "Unstructured.io", category: "Document Loaders", priority: "Core", description: "Universal document parsing" },
  { name: "Docling", category: "Document Loaders", priority: "Extended", description: "Document understanding" },
  { name: "Confluence", category: "Document Loaders", priority: "Extended", description: "Atlassian wiki loader" },
  { name: "Notion Loader", category: "Document Loaders", priority: "Extended", description: "Notion workspace loader" },
  { name: "GitHub Loader", category: "Document Loaders", priority: "Extended", description: "Repository content loader" },
  { name: "Google Drive", category: "Document Loaders", priority: "Extended", description: "GDrive document loader" },
  { name: "S3/GCS", category: "Document Loaders", priority: "Extended", description: "Cloud storage loader" },

  // Guardrails (5)
  { name: "NeMo Guardrails", category: "Guardrails", priority: "Core", description: "NVIDIA safety rails" },
  { name: "Guardrails AI", category: "Guardrails", priority: "Extended", description: "Validation framework" },
  { name: "LLM Guard", category: "Guardrails", priority: "Extended", description: "Input/output scanning" },
  { name: "Lakera", category: "Guardrails", priority: "Extended", description: "Prompt injection detection" },
  { name: "Azure AI Safety", category: "Guardrails", priority: "Extended", description: "Content safety service" },

  // Eval & Observability (7)
  { name: "Langfuse", category: "Eval & Observability", priority: "Core", description: "LLM observability platform" },
  { name: "Arize Phoenix", category: "Eval & Observability", priority: "Core", description: "ML observability" },
  { name: "RAGAS", category: "Eval & Observability", priority: "Extended", description: "RAG evaluation framework" },
  { name: "LangSmith", category: "Eval & Observability", priority: "Extended", description: "LangChain observability" },
  { name: "Jaeger", category: "Eval & Observability", priority: "Extended", description: "Distributed tracing" },
  { name: "Grafana", category: "Eval & Observability", priority: "Extended", description: "Metrics visualization" },
  { name: "Datadog", category: "Eval & Observability", priority: "Extended", description: "Cloud monitoring" },

  // Workflows (4)
  { name: "Built-in Engine", category: "Workflows", priority: "Core", description: "Native durable execution" },
  { name: "Temporal", category: "Workflows", priority: "Extended", description: "Workflow orchestration" },
  { name: "NATS", category: "Workflows", priority: "Extended", description: "Message-based workflows" },
  { name: "Redis Streams", category: "Workflows", priority: "Community", description: "Stream-based workflows" },

  // HTTP / API (5)
  { name: "Gin", category: "HTTP / API", priority: "Core", description: "High-performance HTTP framework" },
  { name: "Fiber", category: "HTTP / API", priority: "Extended", description: "Express-inspired Go framework" },
  { name: "Echo", category: "HTTP / API", priority: "Extended", description: "Minimalist HTTP framework" },
  { name: "Chi", category: "HTTP / API", priority: "Extended", description: "Lightweight composable router" },
  { name: "Connect-Go", category: "HTTP / API", priority: "Extended", description: "gRPC-compatible HTTP API" },
];
