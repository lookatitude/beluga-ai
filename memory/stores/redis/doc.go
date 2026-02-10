// Package redis provides a Redis-backed implementation of [memory.MessageStore].
// Messages are stored as JSON in a Redis sorted set, scored by a monotonically
// increasing sequence number to preserve insertion order. Search uses
// case-insensitive substring matching on text content parts.
//
// This implementation requires a Redis server (v5.0+) and uses
// github.com/redis/go-redis/v9 as the client library.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/memory/stores/redis"
//
//	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
//	store, err := redis.New(redis.Config{
//	    Client: client,
//	    Key:    "beluga:messages", // optional, this is the default
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = store.Append(ctx, msg)
//	results, err := store.Search(ctx, "query", 10)
//	all, err := store.All(ctx)
//
// The sorted set key defaults to "beluga:messages" and can be overridden
// via [Config].Key to support multi-tenant or multi-agent deployments.
package redis
