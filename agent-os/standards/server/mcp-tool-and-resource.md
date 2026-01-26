# MCPTool and MCPResource

**MCPTool** (iface): `Name()`, `Description()`, `InputSchema() map[string]any`, `Execute(ctx, input map[string]any) (any, error)`.

**MCPResource** (iface): `URI()`, `Name()`, `Description()`, `MimeType()`, `Read(ctx) ([]byte, error)`.

MCP is **JSON-RPC 2.0 over HTTP**. Single HTTP endpoint (e.g. `/mcp`); `Content-Type: application/json`. Methods: `initialize`, `tools/list`, `tools/call`, `resources/list`, `resources/read`. **ServerName** in MCPConfig is required (expose in initialize serverInfo). **ProtocolVersion** (e.g. `2024-11-05`) in MCPConfig; include in initialize response.

**Registration:** `RegisterTool`/`RegisterResource` at runtime; reject duplicate Name/URI. **WithMCPTool**/**WithMCPResource** at construction. Tools keyed by Name, resources by URI.
