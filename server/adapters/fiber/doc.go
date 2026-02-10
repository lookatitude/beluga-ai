// Package fiber provides a Fiber v3-based ServerAdapter for the Beluga AI
// server package. It wraps the github.com/gofiber/fiber/v3 HTTP framework,
// enabling Beluga agents to be served using Fiber's high-performance routing.
//
// The adapter registers itself as "fiber" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/fiber"
//
//	adapter, err := server.New("fiber", server.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
//
// The underlying fiber.App is accessible via the App() method for advanced
// configuration such as adding Fiber-specific middleware or custom routes.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using Fiber v3
//   - New — creates a new Fiber adapter with the given configuration
package fiber
