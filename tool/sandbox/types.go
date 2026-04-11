package sandbox

import "time"

// NetworkPolicy controls the network access level for a sandbox.
type NetworkPolicy string

const (
	// NetworkIsolated denies all network access from the sandbox.
	NetworkIsolated NetworkPolicy = "isolated"

	// NetworkAllowListed permits network access only to explicitly allowed hosts.
	NetworkAllowListed NetworkPolicy = "allow_listed"

	// NetworkUnrestricted permits all outbound network access.
	NetworkUnrestricted NetworkPolicy = "unrestricted"
)

// ResourceLimits defines the compute resources available to a sandbox.
type ResourceLimits struct {
	// MemoryMB is the maximum memory in megabytes. Zero means no limit.
	MemoryMB int

	// CPUs is the number of CPU cores available. Zero means no limit.
	CPUs float64

	// MaxOutputBytes is the maximum combined size of stdout and stderr
	// captured from the sandbox. Zero means use the default (1 MB).
	MaxOutputBytes int
}

// SandboxConfig configures a single code execution request.
type SandboxConfig struct {
	// Language is the programming language of the code to execute
	// (e.g., "python", "javascript", "go", "bash").
	Language string

	// Timeout is the maximum duration for the execution. Zero means
	// use the sandbox's default timeout.
	Timeout time.Duration

	// NetworkPolicy controls network access for this execution.
	// Zero value defaults to NetworkIsolated.
	//
	// IMPORTANT: Enforcement of NetworkPolicy depends on the sandbox
	// provider. Container-based providers (Docker, gVisor, Firecracker,
	// E2B) enforce this via network namespaces or egress policies.
	// ProcessSandbox CANNOT enforce network policies and always runs
	// with the parent process's network access — it rejects any
	// non-unrestricted policy at Execute() time.
	NetworkPolicy NetworkPolicy

	// Resources sets compute resource limits for this execution.
	Resources ResourceLimits

	// AllowedHosts is the list of permitted hosts when NetworkPolicy
	// is NetworkAllowListed. Ignored for other policies.
	AllowedHosts []string
}

// ExecutionResult holds the output of a sandboxed code execution.
type ExecutionResult struct {
	// Output is the captured stdout from the execution.
	Output string

	// Error is the captured stderr from the execution.
	Error string

	// ExitCode is the process exit code. Zero indicates success.
	ExitCode int

	// Duration is the wall-clock time the execution took.
	Duration time.Duration
}

// defaultTimeout is the fallback execution timeout when none is specified.
const defaultTimeout = 30 * time.Second

// defaultMaxOutputBytes is the fallback maximum output size (1 MB).
const defaultMaxOutputBytes = 1024 * 1024
