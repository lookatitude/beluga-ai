package dragonfly

import (
	goredis "github.com/redis/go-redis/v9"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/memory"
	redisstore "github.com/lookatitude/beluga-ai/v2/memory/stores/redis"
)

// defaultKey is the default sorted-set key, distinct from the Redis store's
// default so both can share one server.
const defaultKey = "beluga:dragonfly:messages"

// Config holds configuration for the DragonflyDB MessageStore.
type Config struct {
	// Client is the Redis-compatible client to use. Required.
	Client *goredis.Client
	// Key is the key for the sorted set. Defaults to "beluga:dragonfly:messages".
	Key string
}

// MessageStore is a DragonflyDB-backed implementation of memory.MessageStore.
// DragonflyDB speaks the Redis protocol, so it is a thin wrapper over the
// Redis store ([redisstore.MessageStore]); all methods are promoted from it.
type MessageStore struct {
	*redisstore.MessageStore
}

// New creates a new DragonflyDB MessageStore with the given config.
func New(cfg Config) (*MessageStore, error) {
	if cfg.Client == nil {
		return nil, core.Errorf(core.ErrInvalidInput, "dragonfly: client is required")
	}
	key := cfg.Key
	if key == "" {
		key = defaultKey
	}
	inner, err := redisstore.New(redisstore.Config{Client: cfg.Client, Key: key})
	if err != nil {
		return nil, err
	}
	return &MessageStore{MessageStore: inner}, nil
}

// Verify interface compliance.
var _ memory.MessageStore = (*MessageStore)(nil)
