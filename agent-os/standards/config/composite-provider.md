# Composite Provider

**Shape:** `NewCompositeProvider(providers ...Provider)`. Supports multiple providers with a clear order: the first is the default, the rest are backups in addition order. Config must allow identifying the default and backups.

**Load:** Try each provider in order. First successful `Load` wins. No merging across providers.

**UnmarshalKey / Get\***: First provider where `IsSet(key)` returns true wins; use that provider's value.

**Errors:**

- Nil entry in the slice → immediate `ErrCodeInvalidParameters` (fail before trying others).
- All providers fail `Load` → `ErrCodeAllProvidersFailed`.
- Key not present in any provider → `ErrCodeKeyNotFound`.
- `UnmarshalKey` error from a provider → wrap with `ErrCodeParseFailed`.
