// Package tasks provides built-in sleep-time compute tasks for the Beluga AI
// framework.
//
// It contains concrete implementations of the sleeptime.Task interface that
// run during agent idle periods. Current tasks include contradiction resolution
// (detecting and resolving conflicting facts across turns) and memory
// reorganization (consolidating old conversation turns by topic proximity).
//
// This package lives in Layer 6 (Agent runtime) and depends on the
// runtime/sleeptime package.
package tasks
