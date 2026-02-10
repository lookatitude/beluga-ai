// Package httpclient provides a shared HTTP client with retry, SSE streaming,
// WebSocket support, and typed JSON helpers used by providers without dedicated
// Go SDKs.
//
// This is an internal package and is not part of the public API. It is used by
// LLM provider packages (e.g., Groq, Together, Fireworks) that communicate over
// HTTP with OpenAI-compatible or custom REST APIs.
//
// # Client
//
// The [Client] type wraps net/http.Client with automatic retry on 429/503
// status codes and network errors, exponential backoff with jitter, and
// default headers (including bearer token authentication). Configuration
// uses the functional options pattern:
//
//	c := httpclient.New(
//	    httpclient.WithBaseURL("https://api.example.com/v1"),
//	    httpclient.WithBearerToken(apiKey),
//	    httpclient.WithRetries(3),
//	    httpclient.WithTimeout(30 * time.Second),
//	)
//
// # Typed JSON Requests
//
// The [DoJSON] generic function sends an HTTP request with a JSON body and
// decodes the JSON response into the specified type. It handles retries
// transparently:
//
//	type Response struct { Result string `json:"result"` }
//	resp, err := httpclient.DoJSON[Response](ctx, client, "POST", "/chat", reqBody)
//
// # Server-Sent Events
//
// The [StreamSSE] and [StreamSSEWithBody] functions open SSE connections and
// return iter.Seq2[SSEEvent, error] iterators that yield parsed SSE events.
// This is the primary mechanism for streaming LLM responses:
//
//	for event, err := range httpclient.StreamSSEWithBody(ctx, client, "POST", "/chat", body) {
//	    if err != nil { break }
//	    // handle event.Data
//	}
//
// # WebSocket
//
// The [WSConn] type wraps a WebSocket connection with typed JSON read/write
// helpers. It is used by voice transport and real-time providers:
//
//	ws, err := httpclient.DialWS(ctx, "wss://api.example.com/ws", nil)
//	if err != nil { return err }
//	defer ws.Close()
//	err = ws.WriteJSON(ctx, request)
//
// # Error Handling
//
// API errors are returned as [*APIError] with the HTTP status code and
// response body. The client automatically parses JSON error bodies to extract
// human-readable error messages.
package httpclient
