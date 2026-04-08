package rl

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// TrajectoryCollector records policy decisions as episodes for offline
// training. It is safe for concurrent use.
type TrajectoryCollector struct {
	mu       sync.Mutex
	episodes []Episode
	current  *Episode
	hooks    Hooks
}

// NewTrajectoryCollector creates a TrajectoryCollector with optional hooks.
func NewTrajectoryCollector(hooks ...Hooks) *TrajectoryCollector {
	var h Hooks
	if len(hooks) > 0 {
		h = hooks[0]
	}
	return &TrajectoryCollector{hooks: h}
}

// RecordStep appends a step to the current episode. If no episode is active,
// a new one is started automatically.
func (c *TrajectoryCollector) RecordStep(features PolicyFeatures, action MemoryAction, confidence float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.current == nil {
		c.current = &Episode{
			ID:        generateEpisodeID(),
			StartTime: time.Now(),
		}
	}

	c.current.Steps = append(c.current.Steps, Step{
		Features:   features,
		Action:     action,
		Confidence: confidence,
		Timestamp:  time.Now(),
	})
}

// EndEpisode finalizes the current episode with the given outcome and
// adds it to the collected episodes. If hooks.OnEpisodeEnd is set, it is
// invoked with the completed episode.
func (c *TrajectoryCollector) EndEpisode(ctx context.Context, outcome any) error {
	c.mu.Lock()
	if c.current == nil {
		c.mu.Unlock()
		return core.NewError(
			"rl.collector",
			core.ErrInvalidInput,
			"no active episode to end",
			nil,
		)
	}

	c.current.Outcome = outcome
	c.current.EndTime = time.Now()
	episode := *c.current
	c.episodes = append(c.episodes, episode)
	c.current = nil
	c.mu.Unlock()

	// Invoke hook outside the lock.
	if c.hooks.OnEpisodeEnd != nil {
		c.hooks.OnEpisodeEnd(ctx, episode)
	}
	return nil
}

// Episodes returns a copy of all collected episodes.
func (c *TrajectoryCollector) Episodes() []Episode {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Episode, len(c.episodes))
	copy(out, c.episodes)
	return out
}

// Len returns the number of completed episodes.
func (c *TrajectoryCollector) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.episodes)
}

// Export serializes all collected episodes as JSON. This is intended for
// saving training data to disk or sending to a training service.
func (c *TrajectoryCollector) Export() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return json.Marshal(c.episodes)
}

// episodeCounter is a simple monotonic counter for generating episode IDs.
var (
	episodeCounterMu sync.Mutex
	episodeCounter   int64
)

// generateEpisodeID returns a unique episode ID based on timestamp and counter.
func generateEpisodeID() string {
	episodeCounterMu.Lock()
	episodeCounter++
	n := episodeCounter
	episodeCounterMu.Unlock()

	return time.Now().Format("20060102T150405") + "-" + formatInt64(n)
}

// formatInt64 converts an int64 to a string without importing strconv.
func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
