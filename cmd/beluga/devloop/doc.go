// Package devloop owns the lifecycle of a scaffolded beluga project when
// invoked via `beluga run` or `beluga dev`. It is a Layer 7 subpackage of
// the beluga CLI — stdlib + fsnotify only, no imports from other beluga
// packages beyond schema-free cmd/beluga helpers.
//
// The public surface is a single function, [Run], which supervises a
// child process built from the caller's project tree. `beluga run` invokes
// [Run] with [Config.Watch]=false and executes the binary exactly once,
// forwarding stdio and the child's exit code. `beluga dev` invokes [Run]
// with [Config.Watch]=true, attaches an fsnotify [ChangeFilter]-gated
// watcher, and restarts the child on source changes.
//
// Process-group semantics matter for correctness: the child is started in
// its own process group (via platform-specific build-tagged exec files),
// so SIGTERM to the group cleanly terminates the child and any
// grandchildren it forked. Without this, a user who types Ctrl-C while
// the agent has spawned a tool subprocess would leak the tool.
//
// The debounce window (500ms) lives in the supervisor, not on individual
// filters; a batch of editor-save events on a multi-file refactor thus
// triggers a single rebuild.
package devloop
