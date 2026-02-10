// Package chi provides a Chi-based ServerAdapter for the Beluga AI server
// package. It wraps the github.com/go-chi/chi/v5 router, enabling Beluga
// agents to be served using Chi's lightweight, composable routing.
//
// The adapter registers itself as "chi" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/chi"
//
//	adapter, err := server.New("chi", server.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
//
// The underlying chi.Router is accessible via the Router() method for advanced
// configuration such as adding Chi middleware or custom routes.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using Chi
//   - New — creates a new Chi adapter with the given configuration
package chi
