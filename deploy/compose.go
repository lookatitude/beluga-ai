package deploy

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
)

// serviceNameRe matches safe Docker Compose service names.
var serviceNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// envKeyRe matches valid POSIX environment variable names.
var envKeyRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// AgentDeployment describes a single agent service within a Docker Compose file.
type AgentDeployment struct {
	// Name is the service name, e.g. "planner". Must not be empty.
	Name string

	// ConfigPath is the host path to the agent configuration file that will be
	// bind-mounted into the container at /config/. Must not be empty or contain
	// path traversal sequences.
	ConfigPath string

	// Port is the host port mapped to the container's port 8080.
	// Must be in the range [1, 65535].
	Port int

	// DependsOn lists service names this agent depends on. May be nil or empty.
	DependsOn []string

	// Environment is a set of environment variable key-value pairs injected
	// into the container. May be nil.
	Environment map[string]string
}

// ComposeConfig is the top-level configuration for Docker Compose generation.
type ComposeConfig struct {
	// Agents is the list of agent services to include. Must contain at least
	// one entry.
	Agents []AgentDeployment
}

// validateComposeConfig checks that cfg is well-formed and safe to render as YAML.
func validateComposeConfig(cfg ComposeConfig) error {
	if len(cfg.Agents) == 0 {
		return core.Errorf(core.ErrInvalidInput, "deploy: ComposeConfig must contain at least one agent")
	}
	names := make(map[string]struct{}, len(cfg.Agents))
	for i, a := range cfg.Agents {
		if a.Name == "" {
			return core.Errorf(core.ErrInvalidInput, "deploy: agent[%d].Name must not be empty", i)
		}
		// Validate Name against safe service name characters to prevent YAML injection.
		if !serviceNameRe.MatchString(a.Name) {
			return core.Errorf(core.ErrInvalidInput, "deploy: agent[%d].Name %q contains invalid characters; must match ^[a-zA-Z0-9_-]+$", i, a.Name)
		}
		if a.ConfigPath == "" {
			return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: ConfigPath must not be empty", a.Name)
		}
		// Reject newlines in ConfigPath to prevent YAML injection.
		if strings.ContainsAny(a.ConfigPath, "\n\r") {
			return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: ConfigPath must not contain newlines", a.Name)
		}
		// Use filepath.Clean-based path traversal check instead of naive ".." substring match.
		cleaned := filepath.Clean(a.ConfigPath)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: ConfigPath must not contain path traversal sequences or be absolute", a.Name)
		}
		if a.Port < 1 || a.Port > 65535 {
			return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: Port must be between 1 and 65535", a.Name)
		}
		// Validate environment variable keys and values.
		for k, v := range a.Environment {
			if !envKeyRe.MatchString(k) {
				return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: environment key %q is not a valid POSIX variable name", a.Name, k)
			}
			if strings.ContainsAny(v, "\n\r") {
				return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: environment value for key %q must not contain newlines", a.Name, k)
			}
		}
		// Validate DependsOn entries against safe service name characters.
		for _, dep := range a.DependsOn {
			if !serviceNameRe.MatchString(dep) {
				return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: DependsOn entry %q contains invalid characters", a.Name, dep)
			}
		}
		names[a.Name] = struct{}{}
	}
	// Verify that all DependsOn references resolve to declared services.
	for _, a := range cfg.Agents {
		for _, dep := range a.DependsOn {
			if _, ok := names[dep]; !ok {
				return core.Errorf(core.ErrInvalidInput, "deploy: agent %q: DependsOn %q is not a declared agent", a.Name, dep)
			}
		}
	}
	return nil
}

// GenerateCompose produces a Docker Compose YAML string from cfg.
//
// Each [AgentDeployment] becomes a service that binds its ConfigPath as a
// read-only volume, maps its Port to container port 8080, and declares
// depends_on relationships. Environment variables are written in key=value
// format.
//
// An error is returned if the config is invalid (empty agents list, missing
// names, invalid ports, or unresolvable depends_on references).
func GenerateCompose(cfg ComposeConfig) (string, error) {
	if err := validateComposeConfig(cfg); err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("version: \"3.9\"\n")
	sb.WriteString("services:\n")

	for _, a := range cfg.Agents {
		fmt.Fprintf(&sb, "  %s:\n", a.Name)
		sb.WriteString("    image: beluga-agent:latest\n")
		sb.WriteString("    restart: unless-stopped\n")

		// Volumes
		fmt.Fprintf(&sb, "    volumes:\n")
		fmt.Fprintf(&sb, "      - %s:/config:ro\n", a.ConfigPath)

		// Ports
		fmt.Fprintf(&sb, "    ports:\n")
		fmt.Fprintf(&sb, "      - \"%d:8080\"\n", a.Port)

		// Environment
		if len(a.Environment) > 0 {
			sb.WriteString("    environment:\n")
			// Sort keys for deterministic output.
			keys := make([]string, 0, len(a.Environment))
			for k := range a.Environment {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&sb, "      - %s=%s\n", k, a.Environment[k])
			}
		}

		// DependsOn
		if len(a.DependsOn) > 0 {
			sb.WriteString("    depends_on:\n")
			for _, dep := range a.DependsOn {
				fmt.Fprintf(&sb, "      - %s\n", dep)
			}
		}
	}

	return sb.String(), nil
}
