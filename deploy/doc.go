// Package deploy provides utilities for generating deployment artifacts and
// health-check endpoints for Beluga AI agents.
//
// # Dockerfile Generation
//
// [GenerateDockerfile] produces a multi-stage Dockerfile that compiles a Go
// agent binary in a builder stage and copies it into a minimal runtime image:
//
//	cfg := deploy.DockerfileConfig{
//	    BaseImage:   "gcr.io/distroless/static-debian12",
//	    GoVersion:   "1.23",
//	    AgentConfig: "config/agent.yaml",
//	    Port:        8080,
//	}
//	dockerfile, err := deploy.GenerateDockerfile(cfg)
//
// # Docker Compose Generation
//
// [GenerateCompose] produces a Docker Compose YAML file that defines one
// service per [AgentDeployment], wiring environment variables, ports, and
// service dependencies:
//
//	cfg := deploy.ComposeConfig{
//	    Agents: []deploy.AgentDeployment{
//	        {Name: "planner", ConfigPath: "config/planner.yaml", Port: 8081},
//	        {Name: "executor", ConfigPath: "config/executor.yaml", Port: 8082, DependsOn: []string{"planner"}},
//	    },
//	}
//	compose, err := deploy.GenerateCompose(cfg)
//
// # Health Endpoints
//
// [NewHealthEndpoint] creates an HTTP handler that exposes /healthz (liveness)
// and /readyz (readiness) endpoints. Named checks are registered via
// [HealthEndpoint.AddCheck] and are executed by the readiness handler:
//
//	h := deploy.NewHealthEndpoint()
//	h.AddCheck("database", func(ctx context.Context) error {
//	    return db.PingContext(ctx)
//	})
//	mux.HandleFunc("/healthz", h.Healthz())
//	mux.HandleFunc("/readyz", h.Readyz())
//
// The liveness endpoint always returns 200 OK. The readiness endpoint returns
// 200 OK when all checks pass, and 503 Service Unavailable when any check fails.
package deploy
