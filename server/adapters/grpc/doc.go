// Package grpc provides a gRPC-based ServerAdapter for the Beluga AI server
// package. It exposes agents via unary (Invoke) and server-streaming (Stream)
// RPCs using JSON encoding over gRPC. No .proto file is required — the service
// descriptor is defined programmatically.
//
// The adapter registers itself as "grpc" in the server registry via init().
// Import this package to make the adapter available:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/grpc"
//
//	adapter, err := server.New("grpc", server.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":50051")
//
// The gRPC service exposes the following methods under beluga.AgentService:
//
//   - Invoke (unary) — synchronous agent invocation
//   - Stream (server-streaming) — streaming agent events
//
// Requests and responses use JSON encoding. The ClientCodecOption function
// returns a gRPC dial option for connecting with the JSON codec.
//
// Note: RegisterHandler is not supported for gRPC adapters. Use RegisterAgent
// to expose agents.
//
// The underlying grpc.Server is accessible via the Server() method for
// advanced configuration such as adding interceptors.
//
// # Key Types
//
//   - Adapter — implements server.ServerAdapter using gRPC
//   - New — creates a new gRPC adapter with the given configuration
//   - InvokeRequest / InvokeResponse — unary RPC message types
//   - StreamEvent — streaming RPC event type
//   - ClientCodecOption — returns a dial option for JSON codec clients
package grpc
