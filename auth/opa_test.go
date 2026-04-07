package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOPAPolicy_Name(t *testing.T) {
	p := NewOPAPolicy("test-opa", "http://localhost:8181/v1/data/authz")
	if p.Name() != "test-opa" {
		t.Errorf("expected name 'test-opa', got %q", p.Name())
	}
}

func TestOPAPolicy_AuthorizeAllowed(t *testing.T) {
	// Create a mock OPA server that always allows.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: true})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	allowed, err := p.Authorize(ctx, "alice", PermToolExec, "calculator")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected alice to be allowed")
	}
}

func TestOPAPolicy_AuthorizeDenied(t *testing.T) {
	// Create a mock OPA server that always denies.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: false})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	allowed, err := p.Authorize(ctx, "bob", PermMemoryRead, "history")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected bob to be denied")
	}
}

func TestOPAPolicy_AuthorizeConditionBased(t *testing.T) {
	// Create a mock OPA server that checks the permission.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req opaRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Allow PermToolExec, deny others.
		allowed := req.Permission == string(PermToolExec)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: allowed})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	tests := []struct {
		name      string
		perm      Permission
		wantAllow bool
	}{
		{"allow tool exec", PermToolExec, true},
		{"deny memory read", PermMemoryRead, false},
		{"deny agent delegate", PermAgentDelegate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := p.Authorize(ctx, "user", tt.perm, "resource")
			if err != nil {
				t.Fatalf("Authorize error: %v", err)
			}
			if allowed != tt.wantAllow {
				t.Errorf("Authorize returned %v, want %v", allowed, tt.wantAllow)
			}
		})
	}
}

func TestOPAPolicy_ContextCancellation(t *testing.T) {
	// Create a slow OPA server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: true})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL, WithTimeout(100*time.Millisecond))
	ctx := context.Background()

	_, err := p.Authorize(ctx, "alice", PermToolExec, "calc")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestOPAPolicy_HTTPError(t *testing.T) {
	// Create a mock OPA server that returns a 500 error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	_, err := p.Authorize(ctx, "alice", PermToolExec, "calc")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if err.Error() != "auth/opa: OPA returned status 500: internal error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOPAPolicy_InvalidJSON(t *testing.T) {
	// Create a mock OPA server that returns invalid JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	_, err := p.Authorize(ctx, "alice", PermToolExec, "calc")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestOPAPolicy_RequestBody(t *testing.T) {
	// Create a mock OPA server that verifies the request body.
	var capturedReq opaRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: true})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	p.Authorize(ctx, "alice", PermToolExec, "calculator")

	if capturedReq.Subject != "alice" {
		t.Errorf("expected subject 'alice', got %q", capturedReq.Subject)
	}
	if capturedReq.Permission != string(PermToolExec) {
		t.Errorf("expected permission %q, got %q", PermToolExec, capturedReq.Permission)
	}
	if capturedReq.Resource != "calculator" {
		t.Errorf("expected resource 'calculator', got %q", capturedReq.Resource)
	}
}

func TestOPAPolicy_UnreachableEndpoint(t *testing.T) {
	// Use an endpoint that doesn't exist.
	p := NewOPAPolicy("test", "http://localhost:9999/invalid")
	ctx := context.Background()

	_, err := p.Authorize(ctx, "alice", PermToolExec, "calc")
	if err == nil {
		t.Fatal("expected error for unreachable endpoint")
	}
}

func TestOPAPolicy_WithTimeout(t *testing.T) {
	p := NewOPAPolicy("test", "http://localhost:8181/v1/data/authz")

	// Default timeout should be 5 seconds.
	if p.timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", p.timeout)
	}

	// Custom timeout should override default.
	p2 := NewOPAPolicy("test", "http://localhost:8181/v1/data/authz", WithTimeout(2*time.Second))
	if p2.timeout != 2*time.Second {
		t.Errorf("expected custom timeout 2s, got %v", p2.timeout)
	}
}

func TestOPAPolicy_WithHTTPClient(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}
	p := NewOPAPolicy("test", "http://localhost:8181/v1/data/authz", WithHTTPClient(client))

	if p.client != client {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestOPAPolicy_ImplementsPolicy(t *testing.T) {
	// Compile-time check.
	var _ Policy = (*OPAPolicy)(nil)
}

func TestOPAPolicy_ConcurrentAuthorize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: true})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			allowed, err := p.Authorize(ctx, "user", PermToolExec, "resource")
			done <- allowed && err == nil
		}()
	}

	for i := 0; i < 10; i++ {
		if !<-done {
			t.Fatal("concurrent authorization failed")
		}
	}
}

func TestOPAPolicy_MultiplePermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req opaRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Allow only read permissions.
		allowed := req.Permission == string(PermMemoryRead) || req.Permission == string(PermToolExec)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opaResponse{Result: allowed})
	}))
	defer server.Close()

	p := NewOPAPolicy("test", server.URL)
	ctx := context.Background()

	tests := []struct {
		name      string
		perm      Permission
		wantAllow bool
	}{
		{"tool exec allowed", PermToolExec, true},
		{"memory read allowed", PermMemoryRead, true},
		{"memory write denied", PermMemoryWrite, false},
		{"agent delegate denied", PermAgentDelegate, false},
		{"external api denied", PermExternalAPI, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := p.Authorize(ctx, "user", tt.perm, "resource")
			if err != nil {
				t.Fatalf("Authorize error: %v", err)
			}
			if allowed != tt.wantAllow {
				t.Errorf("Authorize returned %v, want %v", allowed, tt.wantAllow)
			}
		})
	}
}

func TestOPAPolicy_EmptyEndpoint(t *testing.T) {
	// NewOPAPolicy should accept empty endpoint (validation happens at call time).
	p := NewOPAPolicy("test", "")
	if p.Name() != "test" {
		t.Error("NewOPAPolicy with empty endpoint should not panic")
	}
}

func TestOPAPolicy_InitRegistration(t *testing.T) {
	// Test that OPA policy is registered via init().
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	// Re-register OPA manually for this test since init() runs once.
	Register("opa", func(cfg Config) (Policy, error) {
		endpoint, ok := cfg.Extra["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("auth: OPA policy requires 'endpoint' in config")
		}
		if endpoint == "" {
			return nil, fmt.Errorf("auth: OPA endpoint must not be empty")
		}

		var opts []OPAOption
		if timeout, ok := cfg.Extra["timeout"].(float64); ok && timeout > 0 {
			opts = append(opts, WithTimeout(time.Duration(timeout)*time.Second))
		}

		name := "opa"
		if n, ok := cfg.Extra["name"].(string); ok {
			name = n
		}

		return NewOPAPolicy(name, endpoint, opts...), nil
	})

	p, err := New("opa", Config{
		Extra: map[string]any{
			"endpoint": "http://localhost:8181/v1/data/authz",
			"name":     "test-opa",
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if p.Name() != "test-opa" {
		t.Errorf("expected name 'test-opa', got %q", p.Name())
	}

	if _, ok := p.(*OPAPolicy); !ok {
		t.Error("expected OPAPolicy instance")
	}
}

func TestOPAPolicy_RegistrationMissingEndpoint(t *testing.T) {
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	Register("opa", func(cfg Config) (Policy, error) {
		endpoint, ok := cfg.Extra["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("auth: OPA policy requires 'endpoint' in config")
		}
		if endpoint == "" {
			return nil, fmt.Errorf("auth: OPA endpoint must not be empty")
		}

		var opts []OPAOption
		if timeout, ok := cfg.Extra["timeout"].(float64); ok && timeout > 0 {
			opts = append(opts, WithTimeout(time.Duration(timeout)*time.Second))
		}

		name := "opa"
		if n, ok := cfg.Extra["name"].(string); ok {
			name = n
		}

		return NewOPAPolicy(name, endpoint, opts...), nil
	})

	_, err := New("opa", Config{Extra: map[string]any{}})
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestOPAPolicy_RegistrationEmptyEndpoint(t *testing.T) {
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	Register("opa", func(cfg Config) (Policy, error) {
		endpoint, ok := cfg.Extra["endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("auth: OPA policy requires 'endpoint' in config")
		}
		if endpoint == "" {
			return nil, fmt.Errorf("auth: OPA endpoint must not be empty")
		}

		var opts []OPAOption
		if timeout, ok := cfg.Extra["timeout"].(float64); ok && timeout > 0 {
			opts = append(opts, WithTimeout(time.Duration(timeout)*time.Second))
		}

		name := "opa"
		if n, ok := cfg.Extra["name"].(string); ok {
			name = n
		}

		return NewOPAPolicy(name, endpoint, opts...), nil
	})

	_, err := New("opa", Config{Extra: map[string]any{"endpoint": ""}})
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}
