# Config, RESTConfig, MCPConfig, ServerOptions, Option

**Config** (in `iface/options.go`): base fields—Host, Port, ReadTimeout, WriteTimeout, IdleTimeout, MaxHeaderBytes, ShutdownTimeout, LogLevel, CORSOrigins, EnableCORS, EnableMetrics, EnableTracing. **RESTConfig** and **MCPConfig** embed `Config` and add type-specific fields.

**RESTConfig:** APIBasePath, MaxRequestSize, RateLimitRequests, EnableStreaming, EnableRateLimit. **MCPConfig:** ServerName (required), ServerVersion, ProtocolVersion, MaxConcurrentRequests, RequestTimeout.

**ServerOptions:** holds Logger, Tracer, Meter, RESTConfig, MCPConfig, Middlewares, Tools, Resources, Config. **Option** is `func(*ServerOptions)`. Use **WithConfig**, **WithRESTConfig**, **WithMCPConfig**, **WithLogger**, **WithTracer**, **WithMeter**, **WithMiddleware**, **WithMCPTool**, **WithMCPResource**.

**Re-exports:** `config.go` re-exports Config, RESTConfig, MCPConfig, Option, and all `With*` from iface so callers use `server.WithRESTConfig(...)` without importing iface.
