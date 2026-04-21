package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/devloop"
	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/playground"
	"github.com/spf13/cobra"
)

// playgroundNew is indirected so tests can stub the dev-UI server
// without actually binding a listener.
var playgroundNew = playground.New

// newDevCmd returns the cobra subcommand for `beluga dev`. It runs the
// project under a filesystem watcher: edits trigger a debounced
// rebuild-and-restart. When --playground != "off" it also starts the
// loopback dev-UI on 127.0.0.1:<port>.
func newDevCmd() *cobra.Command {
	var (
		projectRoot    string
		playgroundFlag string
	)
	cmd := &cobra.Command{
		Use:   "dev [flags] [-- args...]",
		Short: "Watch the scaffolded project and restart on source changes",
		Long: "Rebuilds and restarts the scaffolded project binary whenever Go " +
			"sources change. Starts a loopback dev-UI at http://127.0.0.1:<port> " +
			"unless --playground=off. Use --playground=0 for an ephemeral port.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			childArgs := argsAfterDash(cmd, args)
			return runDev(cmd.Context(), projectRoot, playgroundFlag, childArgs, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringVar(&projectRoot, "project-root", ".",
		"scaffolded project root (directory with go.mod + .beluga/project.yaml)")
	cmd.Flags().StringVar(&playgroundFlag, "playground", "8089",
		`playground server port; "off" disables it, "0" picks an ephemeral port`)
	return cmd
}

func runDev(ctx context.Context, projectRoot, playgroundFlag string, childArgs []string, stdout, stderr io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project-root: %w", err)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	port, enabled, err := parsePlaygroundFlag(playgroundFlag)
	if err != nil {
		return err
	}

	cfg := devloop.Config{
		ProjectRoot: root,
		Watch:       true,
		Filter:      devloop.GoSourceFilter{},
		Stdout:      stdout,
		Stderr:      stderr,
		ChildArgs:   childArgs,
	}

	if enabled {
		srv, err := playgroundNew(playground.Config{Port: port})
		if err != nil {
			return fmt.Errorf("playground: %w", err)
		}
		if err := srv.Start(); err != nil {
			return fmt.Errorf("playground: %w", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Close(shutdownCtx)
		}()
		fmt.Fprintf(stdout, "beluga dev: playground at http://%s\n", srv.Addr())
		cfg.ExtraEnv = append(cfg.ExtraEnv,
			"BELUGA_PLAYGROUND_URL=http://"+srv.Addr(),
		)
		cfg.Stderr = newStderrTap(stderr, srv.StderrSink())
		cfg.OnRestart = func(seq int) {
			srv.StderrSink() <- playground.StderrLine{
				Bytes:      []byte(fmt.Sprintf("--- restart #%d ---\n", seq)),
				RestartSeq: seq,
			}
		}
	}

	return devloopRun(ctx, cfg)
}

// parsePlaygroundFlag maps the --playground string to (port, enabled, err).
// "off" → (_, false, nil); "0" → (0, true, nil) for ephemeral; integer
// → (n, true, nil); anything else → error.
func parsePlaygroundFlag(flag string) (int, bool, error) {
	if flag == "off" {
		return 0, false, nil
	}
	var port int
	if _, err := fmt.Sscanf(flag, "%d", &port); err != nil {
		return 0, false, fmt.Errorf("invalid --playground %q: want integer port or \"off\"", flag)
	}
	if port < 0 || port > 65535 {
		return 0, false, fmt.Errorf("invalid --playground %d: port must be 0-65535", port)
	}
	return port, true, nil
}

// stderrTap is an io.Writer that both forwards to the user-visible
// stderr and pushes each write as a StderrLine to the playground. It
// never blocks the child: sink writes use a non-blocking send so a
// wedged UI can never wedge the supervisor.
type stderrTap struct {
	out  io.Writer
	sink chan<- playground.StderrLine
	mu   sync.Mutex
}

func newStderrTap(out io.Writer, sink chan<- playground.StderrLine) *stderrTap {
	return &stderrTap{out: out, sink: sink}
}

func (t *stderrTap) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	n, err := t.out.Write(p)
	if err != nil {
		return n, err
	}
	// Copy because the caller may reuse p once Write returns.
	snap := make([]byte, len(p))
	copy(snap, p)
	select {
	case t.sink <- playground.StderrLine{Bytes: bytes.TrimRight(snap, "\x00")}:
	default:
	}
	return n, nil
}
