// Package connect provides a Connect-Go based ServerAdapter for the Beluga AI
// server package. Connect-Go enables HTTP/1.1 and HTTP/2 communication that is
// compatible with gRPC, gRPC-Web, and Connect protocol clients.
//
// The adapter registers itself as "connect" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/connect"
//
//	adapter, err := server.New("connect", server.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
//
// The adapter uses the standard net/http server under the hood, supporting both
// HTTP/1.1 and HTTP/2. This makes it compatible with Connect, gRPC, and gRPC-Web
// clients without requiring separate servers.
//
// The underlying http.ServeMux is accessible via the Mux() method for advanced
// configuration such as registering Connect-Go service handlers directly.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using Connect-Go
//   - New — creates a new Connect-Go adapter with the given configuration
package connect
