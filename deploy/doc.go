// Package deploy provides utilities for generating deployment artifacts and
// health-check endpoints for Beluga AI agents.
//
// # Dockerfile Generation
//
// [GenerateDockerfile] produces a multi-stage Dockerfile that compiles a Go
// agent binary in a builder stage and copies it into a minimal runtime image.
// All input fields are validated against safe character sets before rendering
// to prevent Dockerfile instruction injection.
//
//	cfg := deploy.DockerfileConfig{
//	    BaseImage:   "gcr.io/distroless/static-debian12",
//	    GoVersion:   "1.23",
//	    AgentConfig: "config/agent.yaml",
//	    Port:        8080,
//	}
//	dockerfile, err := deploy.GenerateDockerfile(cfg)
//	if err != nil {
//	    return err
//	}
//	// Write dockerfile to disk or pass to a Docker client.
//
// Defaults: BaseImage = "gcr.io/distroless/static-debian12", GoVersion = "1.23".
// Port must be in [1, 65535]. AgentConfig must not be empty, must not be an
// absolute path, and must not contain path traversal sequences.
//
// # Docker Compose Generation
//
// [GenerateCompose] produces a Docker Compose YAML file that defines one
// service per [AgentDeployment], wiring environment variables, ports, and
// service dependencies. Environment variable keys must be valid POSIX names.
// Service names must match ^[a-zA-Z0-9_-]+$.
//
//	cfg := deploy.ComposeConfig{
//	    Agents: []deploy.AgentDeployment{
//	        {
//	            Name:       "planner",
//	            ConfigPath: "config/planner.yaml",
//	            Port:       8081,
//	        },
//	        {
//	            Name:       "executor",
//	            ConfigPath: "config/executor.yaml",
//	            Port:       8082,
//	            DependsOn:  []string{"planner"},
//	            Environment: map[string]string{
//	                "LOG_LEVEL": "info",
//	            },
//	        },
//	    },
//	}
//	compose, err := deploy.GenerateCompose(cfg)
//	if err != nil {
//	    return err
//	}
//
// All DependsOn entries must reference declared agent names. Unresolvable
// references are rejected by [GenerateCompose].
//
// # Health Endpoints
//
// [NewHealthEndpoint] creates an HTTP handler that exposes /healthz (liveness)
// and /readyz (readiness) endpoints. Named checks are registered via
// [HealthEndpoint.AddCheck] and are executed by the readiness handler.
//
//	h := deploy.NewHealthEndpoint()
//	h.AddCheck("database", func(ctx context.Context) error {
//	    return db.PingContext(ctx)
//	})
//	h.AddCheck("llm-provider", func(ctx context.Context) error {
//	    return model.Health(ctx)
//	})
//	mux := http.NewServeMux()
//	mux.HandleFunc("/healthz", h.Healthz())
//	mux.HandleFunc("/readyz", h.Readyz())
//
// The liveness endpoint always returns 200 OK with {"status":"ok"}.
// The readiness endpoint returns 200 OK when all checks pass, and 503 Service
// Unavailable when any check fails. Each check runs with a 5-second deadline.
// Check error details are suppressed in the HTTP response to prevent
// information disclosure to unauthenticated callers — log errors internally.
//
// Health endpoints are intentionally unauthenticated to support orchestrator
// probes (Kubernetes liveness/readiness probes). Restrict access via
// infrastructure controls (NetworkPolicy, firewall rules) rather than
// application-layer authentication.
//
// # Related packages
//
//   - [github.com/lookatitude/beluga-ai/v2/k8s/operator] — Kubernetes CRD operator
//   - [github.com/lookatitude/beluga-ai/v2/runtime] — agent lifecycle management
package deploy
