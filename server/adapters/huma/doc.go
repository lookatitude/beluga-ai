// Package huma provides a Huma-based ServerAdapter for the Beluga AI server
// package. Huma is an OpenAPI-first framework that wraps standard net/http with
// automatic API documentation generation.
//
// The adapter registers itself as "huma" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/huma"
//
//	adapter, err := server.New("huma", server.Config{
//	    Extra: map[string]any{
//	        "title":   "My API",
//	        "version": "2.0.0",
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
//
// The Config.Extra map supports the following keys:
//
//   - "title" (string) — API title for OpenAPI docs (default: "Beluga AI")
//   - "version" (string) — API version for OpenAPI docs (default: "1.0.0")
//
// The underlying huma.API and http.ServeMux are accessible via the API() and
// Mux() methods for advanced configuration such as registering Huma operations
// or custom routes.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using Huma
//   - New — creates a new Huma adapter with the given configuration
package huma
