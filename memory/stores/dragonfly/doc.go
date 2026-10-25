// Package dragonfly provides a DragonflyDB-backed implementation of [memory.MessageStore].
// DragonflyDB is fully Redis-compatible, so this package is a thin wrapper
// over memory/stores/redis that only changes the default key.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/v2/memory/stores/dragonfly"
//
//	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
//	store, err := dragonfly.New(dragonfly.Config{
//	    Client: client,
//	    Key:    "beluga:dragonfly:messages", // optional, this is the default
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = store.Append(ctx, msg)
//	results, err := store.Search(ctx, "query", 10)
//	all, err := store.All(ctx)
//
// # DragonflyDB vs Redis
//
// DragonflyDB is a modern in-memory data store that provides a Redis-compatible
// API with higher throughput and lower memory usage. This store is functionally
// identical to the Redis store but uses a distinct default key prefix
// ("beluga:dragonfly:messages") to avoid collisions in mixed deployments.
//
// Messages are stored as JSON in a sorted set (ZSET). Scores come from an
// INCR counter at "<key>:seq", so order survives restarts and replicas.
//
// This implementation requires github.com/redis/go-redis/v9.
package dragonfly
