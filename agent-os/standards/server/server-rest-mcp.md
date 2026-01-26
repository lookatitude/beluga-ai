# Server, REST, MCP

**Server** is the base interface: `Start(ctx)`, `Stop(ctx)`, `IsHealthy(ctx)`. **RESTServer** and **MCPServer** extend it; each adds type-specific methods. Factories delegate to providers: `NewRESTServer(opts...)` and `NewMCPServer(opts...)` return `rest.NewServer(opts...)` and `mcp.NewServer(opts...)`.

**RESTServer:** `RegisterHandler(resource, StreamingHandler)`, `RegisterHTTPHandler(method, path, http.HandlerFunc)`, `RegisterMiddleware(Middleware)`, `GetMux() any`. REST lives in `providers/rest/`.

**MCPServer:** `RegisterTool(MCPTool)`, `RegisterResource(MCPResource)`, `ListTools`, `ListResources`, `CallTool(ctx, name, input)`. MCP lives in `providers/mcp/`.

**Interfaces** live in `iface/`; `server` re-exports `Server`, `RESTServer`, `MCPServer` and the factory funcs. Root `server` package does not implement—it forwards to providers.
