// Package sandbox provides sandboxed code execution for the Beluga AI framework.
//
// It defines the [Sandbox] interface for executing code in isolated environments,
// a registry for sandbox providers ([RegisterSandbox], [NewSandbox], [ListSandboxes]),
// a process-based implementation for development ([ProcessSandbox]),
// a pool for reusing sandbox instances ([SandboxPool]),
// a [SandboxTool] adapter that wraps a Sandbox as an agent-callable [tool.Tool],
// and composable lifecycle [Hooks].
//
// # Sandbox Interface
//
// The Sandbox interface is the core abstraction:
//
//	type Sandbox interface {
//	    Execute(ctx context.Context, code string, cfg SandboxConfig) (ExecutionResult, error)
//	    Close(ctx context.Context) error
//	}
//
// Implementations may range from local process execution (for development) to
// remote container-based sandboxes (for production).
//
// # ProcessSandbox (Development Only)
//
// [ProcessSandbox] executes code via os/exec in a temporary directory. It enforces
// timeouts via context but does NOT provide kernel-level isolation (no seccomp,
// no namespaces, no cgroups). It is suitable for development and testing only.
//
//	sb := sandbox.NewProcessSandbox(
//	    sandbox.WithWorkDir("/tmp/sandbox"),
//	    sandbox.WithEnv([]string{"PATH=/usr/bin"}),
//	)
//	result, err := sb.Execute(ctx, "print('hello')", sandbox.SandboxConfig{
//	    Language: "python",
//	    Timeout:  5 * time.Second,
//	})
//
// # SandboxPool
//
// [SandboxPool] pre-creates N sandbox instances for fast reuse via checkout/checkin:
//
//	pool := sandbox.NewSandboxPool("process",
//	    sandbox.WithPoolSize(4),
//	    sandbox.WithWarmup(true),
//	)
//	sb, err := pool.Checkout(ctx)
//	defer pool.Checkin(sb)
//
// # SandboxTool
//
// [SandboxTool] wraps a Sandbox as a [tool.Tool] for use by agents:
//
//	t := sandbox.NewSandboxTool(sb)
//	result, err := t.Execute(ctx, map[string]any{
//	    "code":     "print('hello')",
//	    "language": "python",
//	})
//
// # Hooks
//
// [Hooks] provide lifecycle callbacks: BeforeExecute, AfterExecute, OnTimeout, OnError.
// Compose multiple hooks with [ComposeHooks].
//
// # Registry
//
// Sandbox providers register via [RegisterSandbox] (typically in init()) and are
// instantiated via [NewSandbox]. [ListSandboxes] returns all registered provider names.
//
//	func init() {
//	    sandbox.RegisterSandbox("process", func() (sandbox.Sandbox, error) {
//	        return sandbox.NewProcessSandbox(), nil
//	    })
//	}
package sandbox
