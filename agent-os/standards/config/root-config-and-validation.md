# Root Config and Validation

**Root config struct:** Use `mapstructure` and `yaml` tags. `Config.String()` must redact (e.g. return `"<redacted configuration - sensitive fields not displayed>"`). Unredacted output is allowed only in explicit debugging paths, not in normal or production use.

**ValidationError:** `Field`, `Message`. **ValidationErrors:** slice of `ValidationError`; `Error()` joins all messages.

**ValidateConfig:** Always aggregate: one `ValidationError` per field or section that fails. Return a single `ValidationErrors` when any check fails. Do not return on first error — collect all, then return.

**Delegation:** Delegate to nested types' `Validate()` (e.g. `LLMProviderConfig.Validate`, `ToolConfig` rules like name and provider required). **SetDefaults:** Apply to root and nested schema types (e.g. `DefaultCallOptions`, temperature, max_tokens).
