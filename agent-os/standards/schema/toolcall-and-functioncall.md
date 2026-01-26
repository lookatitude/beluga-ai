# ToolCall and FunctionCall

**ToolCall:** `ID`, `Type`, `Function` (FunctionCall). Top-level `Name` and `Arguments` are optional conveniences; `Function` alone is enough.

**FunctionCall:** `Name`, `Arguments`. Default: `Arguments` is a JSON string; callers parse as needed. Implementations may use another representation (e.g. `map[string]any`) when provided; if none is provided, use JSON.
