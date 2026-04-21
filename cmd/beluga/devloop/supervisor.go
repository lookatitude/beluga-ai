package devloop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Run supervises a scaffolded project binary. With cfg.Watch=false it
// builds once, execs the binary, waits for exit, and returns the child
// error. With cfg.Watch=true it additionally watches the project tree
// (cfg.Filter-gated, cfg.Debounce-delayed) and rebuilds + restarts on
// accepted events. Run returns when the context is cancelled or when
// the child exits (non-watch mode only).
func Run(ctx context.Context, cfg Config) error {
	if err := validateConfig(&cfg); err != nil {
		return err
	}

	baseEnv, err := buildBaseEnv(&cfg)
	if err != nil {
		return err
	}

	sup := &supervisor{
		cfg:     cfg,
		baseEnv: baseEnv,
	}
	if cfg.Watch {
		return sup.runWithWatcher(ctx)
	}
	return sup.runOnce(ctx)
}

type supervisor struct {
	cfg     Config
	baseEnv []string
	seq     int
}

func validateConfig(cfg *Config) error {
	if cfg.ProjectRoot == "" {
		return errors.New("devloop: ProjectRoot is required")
	}
	if cfg.Stdout == nil || cfg.Stderr == nil {
		return errors.New("devloop: Stdout and Stderr are required")
	}
	if cfg.Watch && cfg.Filter == nil {
		return errors.New("devloop: Filter is required when Watch is true")
	}
	if cfg.Debounce == 0 {
		cfg.Debounce = defaultDebounce
	}
	if cfg.GraceTimeout == 0 {
		cfg.GraceTimeout = defaultGrace
	}
	return nil
}

func buildBaseEnv(cfg *Config) ([]string, error) {
	loaded, err := LoadProjectEnv(cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	return MergeEnv(os.Environ(), loaded, cfg.ExtraEnv), nil
}

func (s *supervisor) nextSeq() int {
	s.seq++
	return s.seq
}

// runOnce implements the `beluga run` path: one build, one exec, exit
// code forwarded. Returns a *exec.ExitError when the child exited
// non-zero, or a wrapped error when build/start fails.
func (s *supervisor) runOnce(ctx context.Context) error {
	bin, err := BuildBinary(ctx, s.cfg.ProjectRoot, s.nextSeq(), s.cfg.Stdout, s.cfg.Stderr)
	if err != nil {
		return err
	}
	defer removeIfExists(bin.OutputPath)

	child := newChildCmd(ctx, bin.OutputPath, s.baseEnv, os.Stdin, s.cfg.Stdout, s.cfg.Stderr)
	if err := child.Start(); err != nil {
		return fmt.Errorf("start child: %w", err)
	}
	if s.cfg.OnRestart != nil {
		s.cfg.OnRestart(s.seq)
	}

	waitCh := waitAsync(child)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-waitCh:
		return err
	case <-ctx.Done():
		return terminateGracefully(child, waitCh, s.cfg.GraceTimeout)
	case <-sigCh:
		return terminateGracefully(child, waitCh, s.cfg.GraceTimeout)
	}
}

// runWithWatcher implements the `beluga dev` path: initial build +
// child start, then watch-debounce-rebuild-restart until ctx is
// cancelled (or Ctrl-C). Child exits are not fatal — they are logged
// and the supervisor keeps watching, so an agent that crashes can be
// recovered by saving the file that fixes it.
func (s *supervisor) runWithWatcher(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()
	if err := addRecursive(watcher, s.cfg.ProjectRoot); err != nil {
		return fmt.Errorf("watch project: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	child, waitCh, binPath, err := s.startChild(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if child != nil {
			_ = terminateGracefully(child, waitCh, s.cfg.GraceTimeout)
		}
		removeIfExists(binPath)
	}()

	var debounce *time.Timer
	var debounceC <-chan time.Time
	arm := func() {
		if debounce != nil {
			debounce.Stop()
		}
		debounce = time.NewTimer(s.cfg.Debounce)
		debounceC = debounce.C
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sigCh:
			return nil
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !s.cfg.Filter.Accept(ev.Name, ev.Op) {
				continue
			}
			if ev.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(ev.Name); statErr == nil && info.IsDir() {
					_ = watcher.Add(ev.Name)
				}
			}
			arm()
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(s.cfg.Stderr, "devloop: watcher error: %v\n", err)
		case <-debounceC:
			debounceC = nil
			child, waitCh, binPath = s.restart(ctx, child, waitCh, binPath)
		case err := <-waitCh:
			if err != nil {
				fmt.Fprintf(s.cfg.Stderr, "devloop: child exited: %v\n", err)
			}
			waitCh = nil
			child = nil
		}
	}
}

func (s *supervisor) startChild(ctx context.Context) (*exec.Cmd, <-chan error, string, error) {
	bin, err := BuildBinary(ctx, s.cfg.ProjectRoot, s.nextSeq(), s.cfg.Stdout, s.cfg.Stderr)
	if err != nil {
		return nil, nil, "", err
	}
	child := newChildCmd(ctx, bin.OutputPath, s.baseEnv, os.Stdin, s.cfg.Stdout, s.cfg.Stderr)
	if err := child.Start(); err != nil {
		removeIfExists(bin.OutputPath)
		return nil, nil, "", fmt.Errorf("start child: %w", err)
	}
	if s.cfg.OnRestart != nil {
		s.cfg.OnRestart(s.seq)
	}
	return child, waitAsync(child), bin.OutputPath, nil
}

func (s *supervisor) restart(ctx context.Context, prev *exec.Cmd, prevWait <-chan error, prevBin string) (*exec.Cmd, <-chan error, string) {
	if prev != nil {
		_ = terminateGracefully(prev, prevWait, s.cfg.GraceTimeout)
	}
	removeIfExists(prevBin)

	child, waitCh, binPath, err := s.startChild(ctx)
	if err != nil {
		fmt.Fprintf(s.cfg.Stderr, "devloop: restart failed: %v\n", err)
		return nil, nil, ""
	}
	return child, waitCh, binPath
}

// waitAsync returns a channel that delivers the exec.Cmd.Wait error
// exactly once. Callers MUST consume the result (either directly or
// via terminateGracefully) to avoid leaking the goroutine.
func waitAsync(cmd *exec.Cmd) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- cmd.Wait() }()
	return ch
}

func removeIfExists(path string) {
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

// addRecursive walks root and registers every directory with the
// fsnotify watcher. Hidden directories (".git", ".idea", "node_modules",
// "vendor") are skipped entirely to avoid O(N) CPU on repositories with
// large caches.
func addRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if path != root && (strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules") {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

// ExitCode extracts the exit code from a [Run] error. An *exec.ExitError
// contributes its ExitCode(); any other error returns 1; nil returns 0.
// The cobra layer uses this to exit with the same code the scaffolded
// project binary did.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ee.ExitCode() >= 0 {
			return ee.ExitCode()
		}
	}
	return 1
}
