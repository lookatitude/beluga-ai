// Package echo provides an Echo-based ServerAdapter for the Beluga AI server
// package. It wraps the github.com/labstack/echo/v4 HTTP framework, enabling
// Beluga agents to be served using Echo's routing and middleware ecosystem.
//
// The adapter registers itself as "echo" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/echo"
//
//	adapter, err := server.New("echo", server.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
//
// The underlying echo.Echo instance is accessible via the Echo() method for
// advanced configuration such as adding Echo-specific middleware or custom
// routes.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using Echo
//   - New — creates a new Echo adapter with the given configuration
package echo
