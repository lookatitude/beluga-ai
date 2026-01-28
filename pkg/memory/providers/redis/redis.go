// Package redis provides Redis-backed memory implementations.
// This file re-exports types from internal/redis for the provider pattern.
package redis

import (
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/redis"
)

// Type aliases for backward compatibility and provider pattern.
type (
	// RedisMemory is a memory implementation that stores messages in Redis.
	RedisMemory = redis.RedisMemory

	// RedisClient defines the interface for Redis operations.
	RedisClient = redis.RedisClient

	// RedisConfig holds configuration for RedisMemory.
	RedisConfig = redis.RedisConfig

	// RedisChatMessageHistory implements ChatMessageHistory using Redis.
	RedisChatMessageHistory = redis.RedisChatMessageHistory
)

// NewRedisMemory creates a new Redis-backed memory instance.
var NewRedisMemory = redis.NewRedisMemory

// NewRedisChatMessageHistory creates a new Redis-backed chat message history.
var NewRedisChatMessageHistory = redis.NewRedisChatMessageHistory
