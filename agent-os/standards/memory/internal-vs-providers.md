# internal vs providers (Memory)

**internal/** holds Memory *implementations* (buffer, window, summary, vectorstore). **providers/** holds *swappable storage backends* (in-memory, composite, Redis, Postgres, etc.). Memory is used by default for short/medium term when no config; long-term requires config or is disabled.

- **internal/:** buffer, window, summary, vectorstore — the Memory logic and behavior. Not swappable by config; they are the built-in Memory types.
- **providers/:** Swappable backends used by Memory: BaseChatMessageHistory (RAM), CompositeChatMessageHistory, and persistence backends (e.g. Redis, Postgres). Config selects which provider to use for short/medium/long-term memory.
- **Defaults:** If no memory config is passed for an agent: use memory (RAM) by default for short- and medium-term. Long-term memory requires config; if absent, it is disabled.
- **New storage backend:** Put it in **providers/** (e.g. providers/redis, providers/postgres). It implements the storage/history contract that Memory implementations use.
