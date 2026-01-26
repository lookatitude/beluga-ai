# Heavy Types and Direct Constructors

Summary, SummaryBuffer, VectorStore, and VectorStoreRetriever require LLM, Retriever, Embedder, or VectorStore. **Support injection** via Config or Option so Factory/Registry can create them when deps are provided. When deps are missing, the Registry/Factory creator **returns a clear error** that points to the direct constructor.

- **Injection:** Config or Option may carry LLM, Retriever, Embedder, VectorStore (e.g. Config.LLM, WithLLM, or type-specific config structs). When set, Factory/Registry uses them to create the heavy type. Prefer injection when the caller has these deps (e.g. from a container or builder).
- **Direct constructors:** NewConversationSummaryMemory(history, llm, memoryKey), NewVectorStoreMemory(retriever, memoryKey, returnDocs, k), NewVectorStoreRetrieverMemory(embedder, vectorStore, opts...), etc. Use when constructing manually or when injection is not available.
- **When deps are missing:** The Registry/Factory creator for that type returns an error with a clear message, e.g. "summary memory requires LLM: set Config.LLM or use NewConversationSummaryMemory directly". Do not create with nil deps.
- **Adding a new heavy type:** (1) Document and provide the direct constructor. (2) Add a Registry creator that: uses injected deps from Config/Options when present; otherwise returns a clear error pointing to the direct constructor.
