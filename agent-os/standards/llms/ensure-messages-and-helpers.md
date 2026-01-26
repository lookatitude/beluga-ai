# EnsureMessages and Helpers

**EnsureMessages:** `EnsureMessages(input any) ([]schema.Message, error)`. `string` → `[NewHumanMessage(s)]`; single `schema.Message` → `[msg]`; `[]schema.Message` → unchanged. Any other type → `ErrCodeInvalidRequest` with a clear message. For convenience and backwards compatibility.

**GetSystemAndHumanPromptsFromSchema:** Splits system and human messages; concatenates human messages (e.g. with newlines). Use for models that expect a single system + human string.

**Convenience (in `llms` package):** `GenerateText(ctx, model, prompt, opts)`, `StreamText(ctx, model, prompt, opts)`, `BatchGenerate(ctx, model, prompts, opts)`, `GenerateTextWithTools(ctx, model, prompt, tools, opts)`. Each uses `EnsureMessages` when needed and wires OTEL spans. Keep these in the `llms` package.
