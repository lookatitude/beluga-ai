package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
	"github.com/lookatitude/beluga-ai/tool"
)

type mockAgent struct {
	id     string
	result string
	err    error
	events []agent.Event
}

func (m *mockAgent) ID() string             { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }

func (m *mockAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return m.result, m.err
}

func (m *mockAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		for _, e := range m.events {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func TestRegistry(t *testing.T) {
	t.Run("grpc is registered", func(t *testing.T) {
		names := server.List()
		found := false
		for _, n := range names {
			if n == "grpc" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected 'grpc' in registry, got %v", names)
		}
	})

	t.Run("New returns grpc adapter", func(t *testing.T) {
		adapter, err := server.New("grpc", server.Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if adapter == nil {
			t.Fatal("expected non-nil adapter")
		}
	})
}

func TestAdapter_RegisterAgent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := New(server.Config{})
		ag := &mockAgent{id: "test", result: "hello"}
		if err := a.RegisterAgent("/chat", ag); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil agent returns error", func(t *testing.T) {
		a := New(server.Config{})
		if err := a.RegisterAgent("/chat", nil); err == nil {
			t.Fatal("expected error for nil agent")
		}
	})
}

func TestAdapter_RegisterHandler(t *testing.T) {
	a := New(server.Config{})
	err := a.RegisterHandler("/health", nil)
	if err == nil {
		t.Fatal("expected error for RegisterHandler on gRPC adapter")
	}
}

func TestAdapter_Shutdown_NoServer(t *testing.T) {
	a := New(server.Config{})
	if err := a.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdapter_Server(t *testing.T) {
	a := New(server.Config{})
	if a.Server() == nil {
		t.Fatal("expected non-nil grpc server")
	}
}

// startTestServer starts a gRPC server on a random port and returns the address and cleanup.
func startTestServer(t *testing.T, ag agent.Agent) (*Adapter, string, func()) {
	t.Helper()

	a := New(server.Config{})
	if err := a.RegisterAgent("/chat", ag); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Serve(ctx, addr)
	}()

	// Wait for server to start.
	time.Sleep(100 * time.Millisecond)

	return a, addr, func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil && err != context.Canceled {
				t.Errorf("Serve error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not stop")
		}
	}
}

func dialTest(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		ClientCodecOption(),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func TestAdapter_Invoke(t *testing.T) {
	ag := &mockAgent{id: "test", result: "hello world"}
	_, addr, cleanup := startTestServer(t, ag)
	defer cleanup()

	conn := dialTest(t, addr)
	defer conn.Close()

	reqData, _ := json.Marshal(InvokeRequest{Path: "/chat", Input: "hi"})
	req := rawBytes(reqData)
	var resp rawBytes
	err := conn.Invoke(context.Background(), "/beluga.AgentService/Invoke", &req, &resp)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	var invokeResp InvokeResponse
	if err := json.Unmarshal(resp, &invokeResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if invokeResp.Result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", invokeResp.Result)
	}
}

func TestAdapter_InvokeError(t *testing.T) {
	ag := &mockAgent{id: "test", err: fmt.Errorf("test error")}
	_, addr, cleanup := startTestServer(t, ag)
	defer cleanup()

	conn := dialTest(t, addr)
	defer conn.Close()

	reqData, _ := json.Marshal(InvokeRequest{Path: "/chat", Input: "hi"})
	req := rawBytes(reqData)
	var resp rawBytes
	err := conn.Invoke(context.Background(), "/beluga.AgentService/Invoke", &req, &resp)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	var invokeResp InvokeResponse
	if err := json.Unmarshal(resp, &invokeResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if invokeResp.Error != "test error" {
		t.Fatalf("expected 'test error', got %q", invokeResp.Error)
	}
}

func TestAdapter_Stream(t *testing.T) {
	ag := &mockAgent{
		id: "test",
		events: []agent.Event{
			{Type: agent.EventText, Text: "chunk1"},
			{Type: agent.EventText, Text: "chunk2"},
			{Type: agent.EventDone},
		},
	}
	_, addr, cleanup := startTestServer(t, ag)
	defer cleanup()

	conn := dialTest(t, addr)
	defer conn.Close()

	reqData, _ := json.Marshal(InvokeRequest{Path: "/chat", Input: "hi"})
	streamDesc := &grpc.StreamDesc{
		StreamName:   "Stream",
		ServerStreams: true,
	}
	stream, err := conn.NewStream(context.Background(), streamDesc, "/beluga.AgentService/Stream")
	if err != nil {
		t.Fatalf("NewStream: %v", err)
	}

	req := rawBytes(reqData)
	if err := stream.SendMsg(&req); err != nil {
		t.Fatalf("SendMsg: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}

	var events []StreamEvent
	for {
		var msg rawBytes
		if err := stream.RecvMsg(&msg); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("RecvMsg: %v", err)
		}
		var se StreamEvent
		if err := json.Unmarshal(msg, &se); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		events = append(events, se)
	}

	// Expect 3 events from agent + 1 done event from handler.
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}

	if events[0].Text != "chunk1" {
		t.Fatalf("expected 'chunk1', got %q", events[0].Text)
	}
	if events[1].Text != "chunk2" {
		t.Fatalf("expected 'chunk2', got %q", events[1].Text)
	}
	// Last event should be "done".
	last := events[len(events)-1]
	if last.Type != "done" {
		t.Fatalf("expected 'done' type, got %q", last.Type)
	}
}

func TestAdapter_InvokeUnknownAgent(t *testing.T) {
	ag := &mockAgent{id: "test", result: "hello"}
	_, addr, cleanup := startTestServer(t, ag)
	defer cleanup()

	conn := dialTest(t, addr)
	defer conn.Close()

	reqData, _ := json.Marshal(InvokeRequest{Path: "/unknown", Input: "hi"})
	req := rawBytes(reqData)
	var resp rawBytes
	err := conn.Invoke(context.Background(), "/beluga.AgentService/Invoke", &req, &resp)
	if err == nil {
		t.Fatal("expected error for unknown agent path")
	}
}
