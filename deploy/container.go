package deploy

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
)

// goVersionRe matches valid Go version strings such as "1.23" or "1.23.4".
var goVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// baseImageRe matches Docker image references composed of safe characters only.
// Newlines, semicolons, and shell meta-characters are excluded to prevent
// Dockerfile instruction injection.
var baseImageRe = regexp.MustCompile(`^[a-zA-Z0-9._:/@-]+$`)

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
		return core.Errorf(core.ErrInvalidInput, "deploy: Port must be between 1 and 65535")
	}
	// Validate GoVersion to prevent Dockerfile instruction injection.
	if !goVersionRe.MatchString(cfg.GoVersion) {
		return core.Errorf(core.ErrInvalidInput, "deploy: GoVersion must match ^[0-9]+\\.[0-9]+(\\.[0-9]+)?$")
	}
	// Validate BaseImage to prevent Dockerfile instruction injection.
	if !baseImageRe.MatchString(cfg.BaseImage) {
		return core.Errorf(core.ErrInvalidInput, "deploy: BaseImage contains invalid characters")
	}
	// Explicitly reject newlines in either image field.
	if strings.ContainsAny(cfg.GoVersion, "\n\r") || strings.ContainsAny(cfg.BaseImage, "\n\r") {
		return core.Errorf(core.ErrInvalidInput, "deploy: GoVersion and BaseImage must not contain newlines")
	}
	if cfg.AgentConfig == "" {
		return core.Errorf(core.ErrInvalidInput, "deploy: AgentConfig must not be empty")
	}
	if strings.ContainsAny(cfg.AgentConfig, "\n\r") {
		return core.Errorf(core.ErrInvalidInput, "deploy: AgentConfig must not contain newlines")
	}
	// Use filepath.Clean-based path traversal check instead of naive ".." substring match.
	cleaned := filepath.Clean(cfg.AgentConfig)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return core.Errorf(core.ErrInvalidInput, "deploy: AgentConfig must not contain path traversal sequences or be absolute")
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
