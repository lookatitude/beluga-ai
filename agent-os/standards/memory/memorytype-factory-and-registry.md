# MemoryType, Factory, and Registry

**MemoryType** constants identify built-in and registered types. **Factory** is the extension point for custom providers. **Registry** holds the global set of types and creates by name. **NewMemory(memoryType, Option...)** is the main convenience API.

- **MemoryType:** `buffer`, `buffer_window`, `summary`, `summary_buffer`, `vector_store`, `vector_store_retriever`. Use for NewMemory and for Config.Type. Add new constants when adding built-in types.
- **Factory:** Interface (e.g. CreateMemory(ctx, config)). Implementations can support custom or extra types. Use when extending the package with providers not (or not yet) in the global registry.
- **Registry:** RegisterMemoryType(name, creator), CreateMemory(ctx, typeName, config), ListAvailableMemoryTypes(), GetGlobalMemoryRegistry(). init() registers built-ins. Use when creating by string name, discovering types, or when plugging a new provider into the global registry.
- **NewMemory(memoryType, Option...):** Builds Config from Options, then uses Factory (or registry) to create. Stays the main convenience method for typical use.
- **Validation, Enabled, observability:** Done in the Factory (or in creators used by the registry). Registry's Create delegates to the registered creator; it does not duplicate validation or Enabled logic.
