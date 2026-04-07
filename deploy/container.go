package deploy

import (
	"errors"
	"fmt"
	"strings"
)

// DockerfileConfig holds the parameters used to generate a multi-stage
// Dockerfile for a Beluga AI agent.
type DockerfileConfig struct {
	// BaseImage is the final runtime image, e.g. "gcr.io/distroless/static-debian12".
	// Defaults to "gcr.io/distroless/static-debian12" when empty.
	BaseImage string

	// GoVersion is the Go toolchain version used in the builder stage, e.g. "1.23".
	// Defaults to "1.23" when empty.
	GoVersion string

	// AgentConfig is the path to the agent configuration file that will be
	// copied into the image, e.g. "config/agent.yaml".
	AgentConfig string

	// Port is the port the agent listens on. Must be in the range [1, 65535].
	Port int
}

// defaults applies sensible zero-value defaults to cfg.
func (cfg *DockerfileConfig) defaults() {
	if cfg.BaseImage == "" {
		cfg.BaseImage = "gcr.io/distroless/static-debian12"
	}
	if cfg.GoVersion == "" {
		cfg.GoVersion = "1.23"
	}
}

// validate checks that cfg is well-formed.
func (cfg *DockerfileConfig) validate() error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.New("deploy: Port must be between 1 and 65535")
	}
	if cfg.AgentConfig == "" {
		return errors.New("deploy: AgentConfig must not be empty")
	}
	if strings.Contains(cfg.AgentConfig, "..") {
		return errors.New("deploy: AgentConfig must not contain path traversal sequences")
	}
	return nil
}

// GenerateDockerfile produces a multi-stage Dockerfile string from cfg.
//
// The builder stage uses the official golang image to compile the agent binary.
// The final stage uses cfg.BaseImage (defaulting to distroless) and exposes
// cfg.Port. cfg.AgentConfig is copied into the final image under /config/.
//
// An error is returned if Port is out of range or AgentConfig is empty.
func GenerateDockerfile(cfg DockerfileConfig) (string, error) {
	cfg.defaults()
	if err := cfg.validate(); err != nil {
		return "", err
	}

	var sb strings.Builder

	// Builder stage
	fmt.Fprintf(&sb, "# syntax=docker/dockerfile:1\n")
	fmt.Fprintf(&sb, "FROM golang:%s AS builder\n", cfg.GoVersion)
	sb.WriteString("WORKDIR /src\n")
	sb.WriteString("COPY go.mod go.sum ./\n")
	sb.WriteString("RUN go mod download\n")
	sb.WriteString("COPY . .\n")
	sb.WriteString("RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /agent ./cmd/agent\n")
	sb.WriteString("\n")

	// Final stage
	fmt.Fprintf(&sb, "FROM %s\n", cfg.BaseImage)
	sb.WriteString("WORKDIR /app\n")
	sb.WriteString("COPY --from=builder /agent /app/agent\n")
	fmt.Fprintf(&sb, "COPY %s /config/\n", cfg.AgentConfig)
	sb.WriteString("USER nonroot:nonroot\n")
	fmt.Fprintf(&sb, "EXPOSE %d\n", cfg.Port)
	sb.WriteString(`ENTRYPOINT ["/app/agent"]`)
	sb.WriteString("\n")

	return sb.String(), nil
}
