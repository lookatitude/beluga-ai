package httputil

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

// echoHandler responds 200 OK to every request.
var echoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// freeAddr returns an available local TCP address and immediately closes the
// listener so the address can be reused in tests.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freeAddr: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func TestServerLifecycle_ServeAndShutdownViaContext(t *testing.T) {
	addr := freeAddr(t)
	var lc ServerLifecycle

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- lc.Serve(ctx, addr, echoHandler, 0, 0, 0, "test")
	}()

	// Wait for the server to become reachable.
	if err := waitForServer(addr, 2*time.Second); err != nil {
		cancel()
		t.Fatalf("server did not start: %v", err)
	}

	// Verify the server is serving.
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		cancel()
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Cancel the context — Serve should return context.Canceled.
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("unexpected Serve error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after context cancellation")
	}
}

func TestServerLifecycle_Shutdown_NoServer(t *testing.T) {
	var lc ServerLifecycle
	if err := lc.Shutdown(context.Background(), "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerLifecycle_Shutdown_ExplicitCall(t *testing.T) {
	addr := freeAddr(t)
	var lc ServerLifecycle

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- lc.Serve(ctx, addr, echoHandler, 0, 0, 0, "test")
	}()

	if err := waitForServer(addr, 2*time.Second); err != nil {
		t.Fatalf("server did not start: %v", err)
	}

	if err := lc.Shutdown(context.Background(), "test"); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Serve should return nil (http.ErrServerClosed is swallowed).
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil from Serve after shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after explicit Shutdown")
	}
}

func TestServerLifecycle_ServeListenError(t *testing.T) {
	// Occupy a port so the second ListenAndServe fails immediately.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	defer l.Close()

	var lc ServerLifecycle
	err = lc.Serve(context.Background(), addr, echoHandler, 0, 0, 0, "test")
	if err == nil {
		t.Fatal("expected error when address is already in use")
	}
	if err == http.ErrServerClosed {
		t.Fatal("expected address-in-use error, not ErrServerClosed")
	}
}

func TestServerLifecycle_ErrPrefixInErrors(t *testing.T) {
	// Occupy a port to force a listen error and check the prefix is present.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	defer l.Close()

	const prefix = "server/myprefix"
	var lc ServerLifecycle
	err = lc.Serve(context.Background(), addr, echoHandler, 0, 0, 0, prefix)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); len(got) < len(prefix) || got[:len(prefix)] != prefix {
		t.Fatalf("expected error to start with %q, got %q", prefix, got)
	}
}

func TestServerLifecycle_TimeoutsForwarded(t *testing.T) {
	addr := freeAddr(t)
	var lc ServerLifecycle

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		lc.Serve(ctx, addr, echoHandler, //nolint:errcheck
			100*time.Millisecond, 200*time.Millisecond, 300*time.Millisecond, "test")
	}()

	if err := waitForServer(addr, 2*time.Second); err != nil {
		t.Fatalf("server did not start: %v", err)
	}

	// Access the internal srv to verify timeouts were forwarded.
	lc.mu.RLock()
	srv := lc.srv
	lc.mu.RUnlock()

	if srv == nil {
		t.Fatal("expected srv to be set")
	}
	if srv.ReadTimeout != 100*time.Millisecond {
		t.Fatalf("ReadTimeout: expected 100ms, got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 200*time.Millisecond {
		t.Fatalf("WriteTimeout: expected 200ms, got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 300*time.Millisecond {
		t.Fatalf("IdleTimeout: expected 300ms, got %v", srv.IdleTimeout)
	}
}

// waitForServer polls addr until a TCP connection succeeds or timeout expires.
func waitForServer(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
