package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// InvokeRequest is the gRPC invoke request.
type InvokeRequest struct {
	Path  string `json:"path"`
	Input string `json:"input"`
}

// InvokeResponse is the gRPC invoke response.
type InvokeResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// StreamEvent is an event emitted during gRPC streaming.
type StreamEvent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	AgentID  string         `json:"agent_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// jsonCodec is a gRPC codec that uses JSON encoding.
type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }

func (jsonCodec) Marshal(v any) ([]byte, error) {
	if m, ok := v.(*rawBytes); ok {
		return *m, nil
	}
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	if m, ok := v.(*rawBytes); ok {
		*m = append((*m)[:0], data...)
		return nil
	}
	return json.Unmarshal(data, v)
}

// rawBytes is a raw byte slice used for codec-level marshaling.
type rawBytes []byte

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

// Adapter implements server.ServerAdapter using gRPC.
type Adapter struct {
	grpcServer *grpc.Server
	agents     map[string]agent.Agent
	cfg        server.Config
	mu         sync.RWMutex
}

// Compile-time interface check.
var _ server.ServerAdapter = (*Adapter)(nil)

// New creates a new gRPC adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	return &Adapter{
		grpcServer: grpc.NewServer(grpc.ForceServerCodec(jsonCodec{})),
		agents:     make(map[string]agent.Agent),
		cfg:        cfg,
	}
}

// RegisterAgent registers an agent at the given path. The path is used as a
// routing key for the Invoke and Stream RPCs.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/grpc: agent must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.agents[path] = ag
	return nil
}

// RegisterHandler is not supported for gRPC. It returns an error indicating
// that raw HTTP handlers cannot be registered with a gRPC server.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	return fmt.Errorf("server/grpc: RegisterHandler not supported; use RegisterAgent")
}

// Serve starts the gRPC server on the given address. It blocks until the
// server is stopped or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("server/grpc: listen error: %w", err)
	}

	// Register the agent service.
	a.grpcServer.RegisterService(&agentServiceDesc, a)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		a.grpcServer.GracefulStop()
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully stops the gRPC server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		a.grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		a.grpcServer.Stop()
		return ctx.Err()
	}
}

// Server returns the underlying grpc.Server for advanced configuration.
func (a *Adapter) Server() *grpc.Server {
	return a.grpcServer
}

func (a *Adapter) getAgent(path string) (agent.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ag, ok := a.agents[path]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no agent registered at path %q", path)
	}
	return ag, nil
}

// agentServiceServer is the interface for the gRPC agent service.
type agentServiceServer interface{}

// agentServiceDesc is the gRPC service descriptor for agent services.
var agentServiceDesc = grpc.ServiceDesc{
	ServiceName: "beluga.AgentService",
	HandlerType: (*agentServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Invoke",
			Handler: func(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
				adapter := srv.(*Adapter)
				var req rawBytes
				if err := dec(&req); err != nil {
					return nil, err
				}
				return adapter.invoke(ctx, req)
			},
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:   "Stream",
			ServerStreams: true,
			ClientStreams: false,
			Handler: func(srv any, stream grpc.ServerStream) error {
				adapter := srv.(*Adapter)
				var req rawBytes
				if err := stream.RecvMsg(&req); err != nil {
					return err
				}
				return adapter.stream(req, stream)
			},
		},
	},
}

// invoke handles unary Invoke RPCs.
func (a *Adapter) invoke(ctx context.Context, reqBytes []byte) (*rawBytes, error) {
	var req InvokeRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	ag, err := a.getAgent(req.Path)
	if err != nil {
		return nil, err
	}

	result, err := ag.Invoke(ctx, req.Input)
	if err != nil {
		resp := InvokeResponse{Error: err.Error()}
		data, _ := json.Marshal(resp)
		rb := rawBytes(data)
		return &rb, nil
	}

	resp := InvokeResponse{Result: result}
	data, _ := json.Marshal(resp)
	rb := rawBytes(data)
	return &rb, nil
}

// stream handles server-streaming Stream RPCs.
func (a *Adapter) stream(reqBytes []byte, ss grpc.ServerStream) error {
	var req InvokeRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	ag, err := a.getAgent(req.Path)
	if err != nil {
		return err
	}

	for event, err := range ag.Stream(ss.Context(), req.Input) {
		if err != nil {
			errEvent := StreamEvent{Type: "error", Text: err.Error()}
			data, _ := json.Marshal(errEvent)
			rb := rawBytes(data)
			if sendErr := ss.SendMsg(&rb); sendErr != nil {
				return sendErr
			}
			return nil
		}

		se := StreamEvent{
			Type:     string(event.Type),
			Text:     event.Text,
			AgentID:  event.AgentID,
			Metadata: event.Metadata,
		}
		data, _ := json.Marshal(se)
		rb := rawBytes(data)
		if sendErr := ss.SendMsg(&rb); sendErr != nil {
			return sendErr
		}
	}

	doneEvent := StreamEvent{Type: "done"}
	data, _ := json.Marshal(doneEvent)
	rb := rawBytes(data)
	return ss.SendMsg(&rb)
}

// ClientCodecOption returns a gRPC dial option that configures the JSON codec
// for use with clients connecting to this adapter.
func ClientCodecOption() grpc.DialOption {
	return grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{}))
}

func init() {
	server.Register("grpc", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
