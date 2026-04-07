package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// GenerateDockerfile tests
// ---------------------------------------------------------------------------

func TestGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name       string
		cfg        DockerfileConfig
		wantErr    bool
		wantTokens []string
	}{
		{
			name: "minimal valid config",
			cfg: DockerfileConfig{
				AgentConfig: "config/agent.yaml",
				Port:        8080,
			},
			wantTokens: []string{
				"FROM golang:1.23 AS builder",
				"FROM gcr.io/distroless/static-debian12",
				"EXPOSE 8080",
				"COPY config/agent.yaml /config/",
				"ENTRYPOINT [\"/app/agent\"]",
				"CGO_ENABLED=0",
			},
		},
		{
			name: "custom base image and go version",
			cfg: DockerfileConfig{
				BaseImage:   "alpine:3.20",
				GoVersion:   "1.22",
				AgentConfig: "cfg/planner.yaml",
				Port:        9090,
			},
			wantTokens: []string{
				"FROM golang:1.22 AS builder",
				"FROM alpine:3.20",
				"EXPOSE 9090",
				"COPY cfg/planner.yaml /config/",
			},
		},
		{
			name:    "port zero is invalid",
			cfg:     DockerfileConfig{AgentConfig: "config/agent.yaml", Port: 0},
			wantErr: true,
		},
		{
			name:    "port too high is invalid",
			cfg:     DockerfileConfig{AgentConfig: "config/agent.yaml", Port: 70000},
			wantErr: true,
		},
		{
			name:    "empty AgentConfig is invalid",
			cfg:     DockerfileConfig{Port: 8080},
			wantErr: true,
		},
		{
			name:    "path traversal in AgentConfig rejected",
			cfg:     DockerfileConfig{AgentConfig: "../secrets/agent.yaml", Port: 8080},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateDockerfile(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil; output:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, token := range tt.wantTokens {
				if !strings.Contains(got, token) {
					t.Errorf("Dockerfile missing expected token %q\nFull output:\n%s", token, got)
				}
			}
			// Multi-stage: must contain "AS builder"
			if !strings.Contains(got, "AS builder") {
				t.Errorf("Dockerfile should be multi-stage (missing 'AS builder')")
			}
		})
	}
}

// TestGenerateDockerfileInjectionPrevention validates that injection payloads in
// GoVersion, BaseImage, and AgentConfig are rejected.
func TestGenerateDockerfileInjectionPrevention(t *testing.T) {
	tests := []struct {
		name string
		cfg  DockerfileConfig
	}{
		{
			name: "newline in GoVersion rejected",
			cfg:  DockerfileConfig{GoVersion: "1.23\nRUN malicious", AgentConfig: "config/a.yaml", Port: 8080},
		},
		{
			name: "special chars in GoVersion rejected",
			cfg:  DockerfileConfig{GoVersion: "1.23;rm -rf /", AgentConfig: "config/a.yaml", Port: 8080},
		},
		{
			name: "newline in BaseImage rejected",
			cfg:  DockerfileConfig{BaseImage: "alpine\nRUN evil", AgentConfig: "config/a.yaml", Port: 8080},
		},
		{
			name: "shell metachar in BaseImage rejected",
			cfg:  DockerfileConfig{BaseImage: "alpine$(whoami)", AgentConfig: "config/a.yaml", Port: 8080},
		},
		{
			name: "newline in AgentConfig rejected",
			cfg:  DockerfileConfig{AgentConfig: "config/a.yaml\nRUN evil", Port: 8080},
		},
		{
			name: "absolute AgentConfig rejected",
			cfg:  DockerfileConfig{AgentConfig: "/etc/passwd", Port: 8080},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateDockerfile(tt.cfg)
			if err == nil {
				t.Errorf("expected error for injection payload, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GenerateCompose tests
// ---------------------------------------------------------------------------

func TestGenerateCompose(t *testing.T) {
	tests := []struct {
		name       string
		cfg        ComposeConfig
		wantErr    bool
		wantTokens []string
	}{
		{
			name: "single agent",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "planner", ConfigPath: "config/planner.yaml", Port: 8081},
				},
			},
			wantTokens: []string{
				"version: \"3.9\"",
				"services:",
				"  planner:",
				"config/planner.yaml:/config:ro",
				"\"8081:8080\"",
			},
		},
		{
			name: "multiple agents with depends_on",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "planner", ConfigPath: "config/planner.yaml", Port: 8081},
					{
						Name:        "executor",
						ConfigPath:  "config/executor.yaml",
						Port:        8082,
						DependsOn:   []string{"planner"},
						Environment: map[string]string{"LOG_LEVEL": "debug", "AGENT_MODE": "exec"},
					},
				},
			},
			wantTokens: []string{
				"  planner:",
				"  executor:",
				"depends_on:",
				"      - planner",
				"AGENT_MODE=exec",
				"LOG_LEVEL=debug",
			},
		},
		{
			name:    "empty agents list is invalid",
			cfg:     ComposeConfig{Agents: nil},
			wantErr: true,
		},
		{
			name: "agent with empty name is invalid",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "", ConfigPath: "config/agent.yaml", Port: 8080},
				},
			},
			wantErr: true,
		},
		{
			name: "agent with empty config path is invalid",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "a", ConfigPath: "", Port: 8080},
				},
			},
			wantErr: true,
		},
		{
			name: "agent with path traversal in config path rejected",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "a", ConfigPath: "../secrets", Port: 8080},
				},
			},
			wantErr: true,
		},
		{
			name: "agent with invalid port is invalid",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "a", ConfigPath: "config/a.yaml", Port: 0},
				},
			},
			wantErr: true,
		},
		{
			name: "unresolvable depends_on is invalid",
			cfg: ComposeConfig{
				Agents: []AgentDeployment{
					{Name: "executor", ConfigPath: "config/executor.yaml", Port: 8082, DependsOn: []string{"nonexistent"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateCompose(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil; output:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, token := range tt.wantTokens {
				if !strings.Contains(got, token) {
					t.Errorf("Compose YAML missing expected token %q\nFull output:\n%s", token, got)
				}
			}
		})
	}
}

// TestGenerateComposeInjectionPrevention validates that injection payloads in
// service names, config paths, environment keys/values, and depends_on entries
// are rejected before YAML is generated.
func TestGenerateComposeInjectionPrevention(t *testing.T) {
	tests := []struct {
		name string
		cfg  ComposeConfig
	}{
		{
			name: "newline in service Name rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc\nevil:", ConfigPath: "cfg/a.yaml", Port: 8080},
			}},
		},
		{
			name: "special chars in service Name rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc;evil", ConfigPath: "cfg/a.yaml", Port: 8080},
			}},
		},
		{
			name: "newline in ConfigPath rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc", ConfigPath: "cfg/a.yaml\nevil: true", Port: 8080},
			}},
		},
		{
			name: "absolute ConfigPath rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc", ConfigPath: "/etc/passwd", Port: 8080},
			}},
		},
		{
			name: "invalid env key rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc", ConfigPath: "cfg/a.yaml", Port: 8080,
					Environment: map[string]string{"BAD KEY!": "val"}},
			}},
		},
		{
			name: "newline in env value rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc", ConfigPath: "cfg/a.yaml", Port: 8080,
					Environment: map[string]string{"KEY": "val\nevil: injected"}},
			}},
		},
		{
			name: "special chars in DependsOn rejected",
			cfg: ComposeConfig{Agents: []AgentDeployment{
				{Name: "svc", ConfigPath: "cfg/a.yaml", Port: 8080,
					DependsOn: []string{"dep;evil"}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateCompose(tt.cfg)
			if err == nil {
				t.Errorf("expected error for injection payload, got nil")
			}
		})
	}
}

// TestGenerateComposeEnvironmentOrder verifies that environment variables are
// written in deterministic (sorted) order regardless of map iteration order.
func TestGenerateComposeEnvironmentOrder(t *testing.T) {
	cfg := ComposeConfig{
		Agents: []AgentDeployment{
			{
				Name:       "svc",
				ConfigPath: "cfg/svc.yaml",
				Port:       8080,
				Environment: map[string]string{
					"ZZZ": "last",
					"AAA": "first",
					"MMM": "middle",
				},
			},
		},
	}

	out, err := GenerateCompose(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	posAAA := strings.Index(out, "AAA=first")
	posMMM := strings.Index(out, "MMM=middle")
	posZZZ := strings.Index(out, "ZZZ=last")

	if posAAA < 0 || posMMM < 0 || posZZZ < 0 {
		t.Fatalf("missing environment variables in output:\n%s", out)
	}
	if !(posAAA < posMMM && posMMM < posZZZ) {
		t.Errorf("environment variables not in sorted order in output:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// HealthEndpoint tests
// ---------------------------------------------------------------------------

func TestHealthEndpointHealthz(t *testing.T) {
	h := NewHealthEndpoint()
	handler := h.Healthz()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestHealthEndpointReadyz(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(h *HealthEndpoint)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "no checks registered always healthy",
			setupFn:    func(_ *HealthEndpoint) {},
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name: "all checks pass",
			setupFn: func(h *HealthEndpoint) {
				h.AddCheck("db", func(_ context.Context) error { return nil })
				h.AddCheck("cache", func(_ context.Context) error { return nil })
			},
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name: "one check fails returns 503",
			setupFn: func(h *HealthEndpoint) {
				h.AddCheck("db", func(_ context.Context) error { return nil })
				h.AddCheck("cache", func(_ context.Context) error { return errors.New("connection refused") })
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "fail",
		},
		{
			name: "all checks fail returns 503",
			setupFn: func(h *HealthEndpoint) {
				h.AddCheck("svc", func(_ context.Context) error { return errors.New("down") })
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthEndpoint()
			tt.setupFn(h)

			handler := h.Readyz()
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected HTTP %d, got %d; body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			var resp healthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse body: %v", err)
			}
			if resp.Status != tt.wantBody {
				t.Errorf("expected status %q, got %q", tt.wantBody, resp.Status)
			}
		})
	}
}

// TestHealthEndpointReadyzErrorSuppressed verifies that raw error messages are
// NOT present in the response body (information disclosure prevention) but the
// check name is still reported so operators can identify which check failed.
func TestHealthEndpointReadyzErrorSuppressed(t *testing.T) {
	h := NewHealthEndpoint()
	h.AddCheck("redis", func(_ context.Context) error {
		return errors.New("dial tcp: connection refused")
	})

	rec := httptest.NewRecorder()
	h.Readyz()(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	body := rec.Body.String()
	// Error message must NOT appear to prevent information disclosure.
	if strings.Contains(body, "connection refused") {
		t.Errorf("error message must not be disclosed in response body, got: %s", body)
	}
	// Check name must still appear so operators know which check failed.
	if !strings.Contains(body, "redis") {
		t.Errorf("expected check name 'redis' in body, got: %s", body)
	}
	// Status for failing check must be "unhealthy".
	if !strings.Contains(body, "unhealthy") {
		t.Errorf("expected 'unhealthy' status in body, got: %s", body)
	}
}

// TestHealthEndpointMethodRestriction verifies that non-GET/HEAD methods are
// rejected with 405 on both Healthz and Readyz.
func TestHealthEndpointMethodRestriction(t *testing.T) {
	h := NewHealthEndpoint()
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run("healthz/"+method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.Healthz()(rec, httptest.NewRequest(method, "/healthz", nil))
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", rec.Code)
			}
		})
		t.Run("readyz/"+method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.Readyz()(rec, httptest.NewRequest(method, "/readyz", nil))
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", rec.Code)
			}
		})
	}

	// HEAD should be accepted.
	t.Run("healthz/HEAD", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.Healthz()(rec, httptest.NewRequest(http.MethodHead, "/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for HEAD, got %d", rec.Code)
		}
	})
	t.Run("readyz/HEAD", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.Readyz()(rec, httptest.NewRequest(http.MethodHead, "/readyz", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for HEAD, got %d", rec.Code)
		}
	})
}

// TestHealthEndpointAddCheckConcurrent verifies that AddCheck is safe to call
// concurrently.
func TestHealthEndpointAddCheckConcurrent(t *testing.T) {
	h := NewHealthEndpoint()
	done := make(chan struct{})

	const n = 50
	for i := 0; i < n; i++ {
		go func() {
			h.AddCheck("c", func(_ context.Context) error { return nil })
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}

	rec := httptest.NewRecorder()
	h.Readyz()(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after concurrent AddCheck, got %d", rec.Code)
	}
}

// TestHealthEndpointReadyzContextCancellation verifies that a cancelled
// request context propagates into check functions.
func TestHealthEndpointReadyzContextCancellation(t *testing.T) {
	h := NewHealthEndpoint()

	checkCtxDone := make(chan struct{}, 1)
	h.AddCheck("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			checkCtxDone <- struct{}{}
			return ctx.Err()
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	// Cancel immediately so the check's context (derived from request context)
	// gets the deadline from WithTimeout, but the parent is already cancelled.
	cancel()

	rec := httptest.NewRecorder()
	h.Readyz()(rec, req)

	// Should return 503 because the check returned an error.
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
