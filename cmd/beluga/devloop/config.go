package devloop

import (
	"io"
	"time"
)

// Config drives a single call to [Run]. The zero value is invalid;
// populate at minimum ProjectRoot and Stdout/Stderr.
type Config struct {
	// ProjectRoot is the absolute path to the scaffolded project root.
	// `go build` is invoked here, .env is read from here, and the
	// watcher (when enabled) is rooted here.
	ProjectRoot string

	// Watch toggles rebuild-on-change behaviour. When false (the
	// `beluga run` path), Run builds once, execs the binary, waits for
	// exit, and returns the child's exit code.
	Watch bool

	// Filter decides whether a filesystem event should trigger a
	// rebuild. Required when Watch is true; ignored otherwise.
	Filter ChangeFilter

	// Debounce is the quiet period after the last accepted filesystem
	// event before rebuild is triggered. Zero defaults to 500ms.
	Debounce time.Duration

	// GraceTimeout is how long the supervisor waits after SIGTERM
	// before escalating to SIGKILL. Zero defaults to 3s.
	GraceTimeout time.Duration

	// ExtraEnv overlays environment variables on top of OS env and
	// parsed .env entries. Values follow "KEY=value" format; the last
	// entry for a given key wins.
	ExtraEnv []string

	// Stdout and Stderr are the writers the child inherits. They must
	// not be nil — the cobra layer passes os.Stdout / os.Stderr.
	Stdout io.Writer
	Stderr io.Writer

	// OnRestart, when non-nil, is invoked after a successful rebuild
	// has produced a new running child. It receives the sequence
	// number (starting at 1 for the first child after Run begins). The
	// playground server uses this to correlate trace exports with
	// restart boundaries.
	OnRestart func(seq int)
}

// defaultDebounce is the quiet period applied when Config.Debounce is zero.
const defaultDebounce = 500 * time.Millisecond

// defaultGrace is the SIGTERM→SIGKILL escalation delay applied when
// Config.GraceTimeout is zero.
const defaultGrace = 3 * time.Second
