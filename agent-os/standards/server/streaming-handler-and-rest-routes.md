# StreamingHandler and REST Routes

**StreamingHandler** (iface): `HandleStreaming(w, r)` and `HandleNonStreaming(w, r)`. One handler per resource; the REST implementation chooses based on path (e.g. `/{resource}/{id}/stream` → HandleStreaming, `/{resource}/{id}` and `/{resource}` → HandleNonStreaming).

**RegisterHandler(resource, handler):** binds a `StreamingHandler` to a resource name. Routes are derived from `{APIBasePath}/{resource}/{id}/stream`, `{APIBasePath}/{resource}/{id}`, `{APIBasePath}/{resource}`. **RegisterHTTPHandler(method, path, handler):** registers a raw `http.HandlerFunc` for method+path; use for one-off routes (e.g. `/custom`).

**/health:** always registered by the REST server; returns JSON with status, timestamp, uptime. No handler registration needed.

**Middleware order:** Apply user `WithMiddleware`/`RegisterMiddleware` first (outer), then built-in: CORS (if enabled), rate limit (if enabled), logging, metrics, tracing. Handler is the router. Order: user middlewares → CORS → rate limit → logging → metrics → tracing → router.
