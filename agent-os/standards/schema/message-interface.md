# Message Interface

**Methods:** `GetType() MessageType`, `GetContent() string`, `ToolCalls() []ToolCall`, `AdditionalArgs() map[string]any`.

**Roles:** `GetType` and `GetContent` for routing and display; `ToolCalls` for tool/function flow; `AdditionalArgs` for provider- or format-specific data.

**Location:** `Message`, `MessageType`, `ToolCall`, and `FunctionCall` live in `iface` so both `schema` and other packages can depend on them.
