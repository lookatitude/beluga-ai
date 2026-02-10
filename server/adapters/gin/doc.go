// Package gin provides a Gin-based ServerAdapter for the Beluga AI server
// package. It wraps the github.com/gin-gonic/gin HTTP framework, enabling
// Beluga agents to be served using Gin's routing and middleware ecosystem.
//
// The adapter registers itself as "gin" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/gin"
//
//	adapter, err := server.New("gin", server.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
//
// The underlying gin.Engine is accessible via the Engine() method for advanced
// configuration such as adding Gin-specific middleware or custom routes.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using Gin
//   - New — creates a new Gin adapter with the given configuration
package gin
