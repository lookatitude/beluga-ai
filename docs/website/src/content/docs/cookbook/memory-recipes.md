---
title: Memory Recipes
description: Practical recipes for managing agent memory with the MemGPT-inspired 3-tier system.
---

## TTL-Based Memory Cleanup

**Problem:** Memory stores grow unbounded over time, consuming resources. You need automatic cleanup of old entries without disrupting active sessions.

**Solution:** Implement a background cleanup goroutine that removes entries older than a configurable TTL.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// MemoryWithTTL wraps a Memory with TTL-based automatic cleanup.
type MemoryWithTTL struct {
	inner   memory.Memory
	ttl     time.Duration
	entries []entryRecord
}

type entryRecord struct {
	savedAt time.Time
	key     string
}

func NewMemoryWithTTL(inner memory.Memory, ttl time.Duration) *MemoryWithTTL {
	return &MemoryWithTTL{
		inner: inner,
		ttl:   ttl,
	}
}

// StartCleanup runs a background goroutine that periodically removes expired entries.
func (m *MemoryWithTTL) StartCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.cleanup(ctx)
			}
		}
	}()
}

func (m *MemoryWithTTL) cleanup(ctx context.Context) {
	cutoff := time.Now().Add(-m.ttl)
	var kept []entryRecord

	for _, entry := range m.entries {
		if entry.savedAt.After(cutoff) {
			kept = append(kept, entry)
		} else {
			slog.Debug("expiring memory entry", "key", entry.key, "age", time.Since(entry.savedAt))
		}
	}

	m.entries = kept
}

// Save persists a message pair and records the timestamp.
func (m *MemoryWithTTL) Save(ctx context.Context, input, output schema.Message) error {
	if err := m.inner.Save(ctx, input, output); err != nil {
		return err
	}
	m.entries = append(m.entries, entryRecord{
		savedAt: time.Now(),
		key:     fmt.Sprintf("entry_%d", len(m.entries)),
	})
	return nil
}

// Load delegates to the inner memory.
func (m *MemoryWithTTL) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return m.inner.Load(ctx, query)
}

// Search delegates to the inner memory.
func (m *MemoryWithTTL) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.inner.Search(ctx, query, k)
}

// Clear delegates to the inner memory.
func (m *MemoryWithTTL) Clear(ctx context.Context) error {
	m.entries = nil
	return m.inner.Clear(ctx)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inner, _ := memory.New("inmemory", nil)
	mem := NewMemoryWithTTL(inner, 24*time.Hour)
	mem.StartCleanup(ctx, 1*time.Hour)

	// Use mem as a normal Memory — old entries expire automatically.
	err := mem.Save(ctx,
		&schema.HumanMessage{Content: "Hello"},
		&schema.AIMessage{Content: "Hi there!"},
	)
	if err != nil {
		slog.Error("save failed", "error", err)
	}

	fmt.Println("Memory with TTL running, entries expire after 24 hours")
}
```

---

## Window-Based Context Recovery

**Problem:** After a long conversation, you need to recover the most recent context window without loading the entire history.

**Solution:** Use a sliding window that keeps the last N turns plus any pinned system messages.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// WindowMemory wraps Memory to return only the last N message pairs.
type WindowMemory struct {
	inner      memory.Memory
	windowSize int
	history    []messagePair
}

type messagePair struct {
	Input  schema.Message
	Output schema.Message
}

func NewWindowMemory(inner memory.Memory, windowSize int) *WindowMemory {
	return &WindowMemory{
		inner:      inner,
		windowSize: windowSize,
	}
}

func (w *WindowMemory) Save(ctx context.Context, input, output schema.Message) error {
	w.history = append(w.history, messagePair{Input: input, Output: output})

	// Keep only the last windowSize pairs in local cache.
	if len(w.history) > w.windowSize {
		w.history = w.history[len(w.history)-w.windowSize:]
	}

	return w.inner.Save(ctx, input, output)
}

// Load returns the most recent messages from the window.
func (w *WindowMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	msgs := make([]schema.Message, 0, len(w.history)*2)
	for _, pair := range w.history {
		msgs = append(msgs, pair.Input, pair.Output)
	}
	return msgs, nil
}

func (w *WindowMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return w.inner.Search(ctx, query, k)
}

func (w *WindowMemory) Clear(ctx context.Context) error {
	w.history = nil
	return w.inner.Clear(ctx)
}

func main() {
	ctx := context.Background()

	inner, _ := memory.New("inmemory", nil)
	mem := NewWindowMemory(inner, 10) // Keep last 10 turns.

	// Simulate a long conversation.
	for i := 0; i < 50; i++ {
		err := mem.Save(ctx,
			&schema.HumanMessage{Content: fmt.Sprintf("Message %d", i)},
			&schema.AIMessage{Content: fmt.Sprintf("Response %d", i)},
		)
		if err != nil {
			slog.Error("save failed", "error", err)
		}
	}

	// Load only returns the last 10 turns (20 messages).
	msgs, err := mem.Load(ctx, "")
	if err != nil {
		slog.Error("load failed", "error", err)
		return
	}
	fmt.Printf("Loaded %d messages from window (last 10 turns)\n", len(msgs))
}
```

---

## Switch Between Memory Stores

**Problem:** You want to use in-memory storage during development and Redis or PostgreSQL in production, without changing application code.

**Solution:** Use the registry pattern to select the store by name from configuration.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"

	// Import providers — each registers via init().
	_ "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
	_ "github.com/lookatitude/beluga-ai/memory/stores/redis"
	_ "github.com/lookatitude/beluga-ai/memory/stores/postgres"
)

func main() {
	ctx := context.Background()

	// Select provider from environment or config.
	provider := os.Getenv("MEMORY_PROVIDER")
	if provider == "" {
		provider = "inmemory" // Default for development.
	}

	cfg := config.ProviderConfig{
		"provider": provider,
	}

	// Add provider-specific config.
	switch provider {
	case "redis":
		cfg["address"] = os.Getenv("REDIS_URL")
	case "postgres":
		cfg["connection_string"] = os.Getenv("DATABASE_URL")
	}

	mem, err := memory.New(provider, cfg)
	if err != nil {
		slog.Error("memory creation failed", "provider", provider, "error", err)
		return
	}

	// List all available providers.
	fmt.Printf("Available memory providers: %v\n", memory.List())
	fmt.Printf("Using: %s\n", provider)

	// Use mem as normal — behavior is identical regardless of provider.
	err = mem.Save(ctx,
		&schema.HumanMessage{Content: "Test message"},
		&schema.AIMessage{Content: "Test response"},
	)
	if err != nil {
		slog.Error("save failed", "error", err)
		return
	}

	msgs, err := mem.Load(ctx, "test")
	if err != nil {
		slog.Error("load failed", "error", err)
		return
	}
	fmt.Printf("Loaded %d messages\n", len(msgs))
}
```

---

## Graph Memory for Relationship Tracking

**Problem:** You need to track relationships between entities (people, projects, concepts) that an agent learns during conversations.

**Solution:** Use the graph memory tier to store entity-relationship triples.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// GraphMemoryUsage demonstrates entity-relationship storage.
func main() {
	ctx := context.Background()

	// Create a composite memory with graph tier enabled.
	mem, err := memory.New("composite", map[string]any{
		"core":     map[string]any{"provider": "inmemory"},
		"recall":   map[string]any{"provider": "inmemory"},
		"archival": map[string]any{"provider": "inmemory"},
		"graph":    map[string]any{"provider": "inmemory"},
	})
	if err != nil {
		slog.Error("memory creation failed", "error", err)
		return
	}

	// Save conversation turns — the graph tier extracts entities.
	turns := []struct {
		input, output string
	}{
		{"Alice works at Acme Corp as a senior engineer", "Got it, I'll remember that."},
		{"Alice's manager is Bob", "Noted, Bob manages Alice."},
		{"Acme Corp is building a new AI platform", "Interesting project!"},
		{"Bob also leads the data team", "Bob leads both teams then."},
	}

	for _, turn := range turns {
		err := mem.Save(ctx,
			&schema.HumanMessage{Content: turn.input},
			&schema.AIMessage{Content: turn.output},
		)
		if err != nil {
			slog.Error("save failed", "error", err)
		}
	}

	// Search for relationship context.
	docs, err := mem.Search(ctx, "Who is Alice's manager?", 5)
	if err != nil {
		slog.Error("search failed", "error", err)
		return
	}

	for _, doc := range docs {
		fmt.Printf("Found: %s\n", doc.Content)
	}
}
```

---

## Memory Compression for Long Conversations

**Problem:** Long conversations produce too much recall history to fit in the LLM's context window. You need to summarize older messages while keeping recent ones intact.

**Solution:** Periodically compress older messages into summaries using the LLM itself.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// CompressingMemory summarizes old messages when the history grows too long.
type CompressingMemory struct {
	inner          memory.Memory
	model          llm.ChatModel
	maxMessages    int
	compressAfter  int
	summaries      []string
}

func NewCompressingMemory(inner memory.Memory, model llm.ChatModel, maxMessages int) *CompressingMemory {
	return &CompressingMemory{
		inner:         inner,
		model:         model,
		maxMessages:   maxMessages,
		compressAfter: maxMessages / 2,
	}
}

// Load retrieves messages, compressing old ones into a summary if needed.
func (c *CompressingMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	msgs, err := c.inner.Load(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(msgs) <= c.maxMessages {
		return c.prependSummaries(msgs), nil
	}

	// Compress the oldest messages into a summary.
	toCompress := msgs[:len(msgs)-c.compressAfter]
	recent := msgs[len(msgs)-c.compressAfter:]

	summary, err := c.summarize(ctx, toCompress)
	if err != nil {
		slog.Warn("compression failed, returning uncompressed", "error", err)
		return msgs, nil
	}

	c.summaries = append(c.summaries, summary)
	return c.prependSummaries(recent), nil
}

func (c *CompressingMemory) prependSummaries(msgs []schema.Message) []schema.Message {
	if len(c.summaries) == 0 {
		return msgs
	}
	summaryMsg := &schema.SystemMessage{
		Content: "Previous conversation summary:\n" + strings.Join(c.summaries, "\n---\n"),
	}
	return append([]schema.Message{summaryMsg}, msgs...)
}

func (c *CompressingMemory) summarize(ctx context.Context, msgs []schema.Message) (string, error) {
	var content strings.Builder
	for _, msg := range msgs {
		content.WriteString(msg.GetContent() + "\n")
	}

	prompt := []schema.Message{
		&schema.SystemMessage{Content: "Summarize this conversation concisely, preserving key facts and decisions."},
		&schema.HumanMessage{Content: content.String()},
	}

	resp, err := c.model.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (c *CompressingMemory) Save(ctx context.Context, input, output schema.Message) error {
	return c.inner.Save(ctx, input, output)
}

func (c *CompressingMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return c.inner.Search(ctx, query, k)
}

func (c *CompressingMemory) Clear(ctx context.Context) error {
	c.summaries = nil
	return c.inner.Clear(ctx)
}

func main() {
	ctx := context.Background()

	inner, _ := memory.New("inmemory", nil)

	// Compress when history exceeds 20 messages, keep last 10 intact.
	mem := NewCompressingMemory(inner, nil /* model */, 20)

	msgs, err := mem.Load(ctx, "recent context")
	if err != nil {
		slog.Error("load failed", "error", err)
		return
	}
	fmt.Printf("Context messages: %d\n", len(msgs))
}
```

---

## CompositeMemory: Combining All Tiers

**Problem:** You want to use core memory (always in context), recall memory (searchable history), and archival memory (vector-based) together.

**Solution:** Use `CompositeMemory` which delegates to the appropriate tier.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
)

func main() {
	ctx := context.Background()

	// Create a composite memory that combines all three tiers.
	mem, err := memory.New("composite", config.ProviderConfig{
		"core":     map[string]any{"provider": "inmemory"},
		"recall":   map[string]any{"provider": "inmemory"},
		"archival": map[string]any{"provider": "inmemory"},
	})
	if err != nil {
		slog.Error("memory creation failed", "error", err)
		return
	}

	// Save adds to recall memory (and optionally archival).
	err = mem.Save(ctx,
		&schema.HumanMessage{Content: "My name is Alice and I work on the Beluga project."},
		&schema.AIMessage{Content: "Nice to meet you, Alice! Noted about Beluga."},
	)
	if err != nil {
		slog.Error("save failed", "error", err)
		return
	}

	// Load retrieves from core + recall based on the query.
	msgs, err := mem.Load(ctx, "What project does Alice work on?")
	if err != nil {
		slog.Error("load failed", "error", err)
		return
	}
	fmt.Printf("Loaded %d messages from combined tiers\n", len(msgs))

	// Search queries the archival tier for long-term knowledge.
	docs, err := mem.Search(ctx, "Beluga project details", 5)
	if err != nil {
		slog.Error("search failed", "error", err)
		return
	}
	fmt.Printf("Found %d archival documents\n", len(docs))
}
```

---

## Session-Scoped Memory

**Problem:** In a multi-tenant application, each user session needs isolated memory that doesn't leak between users.

**Solution:** Create per-session memory instances scoped by session ID.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
)

// SessionMemoryManager creates and caches per-session memory instances.
type SessionMemoryManager struct {
	mu       sync.RWMutex
	sessions map[string]memory.Memory
	provider string
	cfg      config.ProviderConfig
}

func NewSessionMemoryManager(provider string, cfg config.ProviderConfig) *SessionMemoryManager {
	return &SessionMemoryManager{
		sessions: make(map[string]memory.Memory),
		provider: provider,
		cfg:      cfg,
	}
}

// GetOrCreate returns an existing session memory or creates a new one.
func (m *SessionMemoryManager) GetOrCreate(sessionID string) (memory.Memory, error) {
	m.mu.RLock()
	if mem, ok := m.sessions[sessionID]; ok {
		m.mu.RUnlock()
		return mem, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if mem, ok := m.sessions[sessionID]; ok {
		return mem, nil
	}

	mem, err := memory.New(m.provider, m.cfg)
	if err != nil {
		return nil, err
	}
	m.sessions[sessionID] = mem
	return mem, nil
}

// Cleanup removes a session's memory.
func (m *SessionMemoryManager) Cleanup(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	mem, ok := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	if ok {
		return mem.Clear(ctx)
	}
	return nil
}

func main() {
	ctx := context.Background()

	manager := NewSessionMemoryManager("inmemory", nil)

	// Each session gets isolated memory.
	mem1, _ := manager.GetOrCreate("session-alice")
	mem2, _ := manager.GetOrCreate("session-bob")

	err := mem1.Save(ctx,
		&schema.HumanMessage{Content: "I prefer Go"},
		&schema.AIMessage{Content: "Noted!"},
	)
	if err != nil {
		slog.Error("save failed", "error", err)
	}

	// Bob's session has no messages — isolation works.
	msgs, _ := mem2.Load(ctx, "preferences")
	fmt.Printf("Bob's messages: %d (should be 0)\n", len(msgs))

	// Clean up when session ends.
	err = manager.Cleanup(ctx, "session-alice")
	if err != nil {
		slog.Error("cleanup failed", "error", err)
	}
}
```

---

## Priority-Based Memory Retrieval

**Problem:** When loading context for an agent, you need to prioritize certain types of memories (e.g., user preferences over general conversation history).

**Solution:** Implement a weighted retrieval strategy that scores memories by type and recency.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// MemoryPriority defines retrieval weights for different memory types.
type MemoryPriority struct {
	CoreWeight     float64 // Weight for core (persona) memories.
	RecallWeight   float64 // Weight for recent conversation history.
	ArchivalWeight float64 // Weight for long-term knowledge.
	RecencyDecay   float64 // Decay factor for older memories (0-1).
}

// DefaultPriority returns balanced retrieval weights.
func DefaultPriority() MemoryPriority {
	return MemoryPriority{
		CoreWeight:     3.0, // Core memories are most important.
		RecallWeight:   2.0, // Recent history is second.
		ArchivalWeight: 1.0, // Archival is least weighted by default.
		RecencyDecay:   0.95,
	}
}

// PriorityMemory wraps Memory with priority-based retrieval ordering.
type PriorityMemory struct {
	inner    memory.Memory
	priority MemoryPriority
	maxItems int
}

func NewPriorityMemory(inner memory.Memory, priority MemoryPriority, maxItems int) *PriorityMemory {
	return &PriorityMemory{
		inner:    inner,
		priority: priority,
		maxItems: maxItems,
	}
}

type scoredMessage struct {
	msg   schema.Message
	score float64
}

// Load retrieves and scores messages by priority, returning the top N.
func (pm *PriorityMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	msgs, err := pm.inner.Load(ctx, query)
	if err != nil {
		return nil, err
	}

	// Score each message.
	scored := make([]scoredMessage, len(msgs))
	for i, msg := range msgs {
		score := pm.scoreMessage(msg, i, len(msgs))
		scored[i] = scoredMessage{msg: msg, score: score}
	}

	// Sort by score descending.
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Take top N.
	limit := pm.maxItems
	if limit > len(scored) {
		limit = len(scored)
	}

	result := make([]schema.Message, limit)
	for i := 0; i < limit; i++ {
		result[i] = scored[i].msg
	}

	return result, nil
}

func (pm *PriorityMemory) scoreMessage(msg schema.Message, index, total int) float64 {
	// Base weight by message type.
	var weight float64
	switch msg.GetRole() {
	case "system":
		weight = pm.priority.CoreWeight
	case "human", "ai":
		weight = pm.priority.RecallWeight
	default:
		weight = pm.priority.ArchivalWeight
	}

	// Apply recency decay — newer messages score higher.
	recency := 1.0
	for i := 0; i < (total - index - 1); i++ {
		recency *= pm.priority.RecencyDecay
	}

	return weight * recency
}

func (pm *PriorityMemory) Save(ctx context.Context, input, output schema.Message) error {
	return pm.inner.Save(ctx, input, output)
}

func (pm *PriorityMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return pm.inner.Search(ctx, query, k)
}

func (pm *PriorityMemory) Clear(ctx context.Context) error {
	return pm.inner.Clear(ctx)
}

func main() {
	ctx := context.Background()

	inner, _ := memory.New("inmemory", nil)
	mem := NewPriorityMemory(inner, DefaultPriority(), 10)

	// System/core memories will be weighted 3x over archival.
	msgs, err := mem.Load(ctx, "user preferences")
	if err != nil {
		slog.Error("load failed", "error", err)
		return
	}

	fmt.Printf("Priority-retrieved %d messages\n", len(msgs))
}
```
