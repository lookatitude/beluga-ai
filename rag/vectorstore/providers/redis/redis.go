// Package redis provides a VectorStore backed by Redis with the RediSearch module.
// It uses Redis hashes to store documents and RediSearch's vector similarity
// search for retrieval.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"
//
//	store, err := vectorstore.New("redis", config.ProviderConfig{
//	    BaseURL: "localhost:6379",
//	    Options: map[string]any{
//	        "index":     "idx:documents",
//	        "prefix":    "doc:",
//	        "dimension": float64(1536),
//	    },
//	})
package redis

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	vectorstore.Register("redis", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// RedisClient abstracts the Redis client for testability.
type RedisClient interface {
	HSet(ctx context.Context, key string, values ...any) *goredis.IntCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
	Do(ctx context.Context, args ...any) *goredis.Cmd
	Close() error
}

// Store is a VectorStore backed by Redis with RediSearch.
type Store struct {
	client    RedisClient
	index     string
	prefix    string
	dimension int
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithIndex sets the RediSearch index name.
func WithIndex(name string) Option {
	return func(s *Store) { s.index = name }
}

// WithPrefix sets the key prefix for document hashes.
func WithPrefix(prefix string) Option {
	return func(s *Store) { s.prefix = prefix }
}

// WithDimension sets the vector dimension.
func WithDimension(dim int) Option {
	return func(s *Store) { s.dimension = dim }
}

// WithClient sets a custom Redis client.
func WithClient(c RedisClient) Option {
	return func(s *Store) { s.client = c }
}

// New creates a new Redis Store.
func New(addr string, opts ...Option) *Store {
	s := &Store{
		index:     "idx:documents",
		prefix:    "doc:",
		dimension: 1536,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Create default client only if none was provided via options.
	if s.client == nil {
		s.client = goredis.NewClient(&goredis.Options{
			Addr: addr,
		})
	}

	return s
}

// NewFromConfig creates a Store from a ProviderConfig.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	addr := cfg.BaseURL
	if addr == "" {
		addr = "localhost:6379"
	}

	var opts []Option
	if idx, ok := config.GetOption[string](cfg, "index"); ok {
		opts = append(opts, WithIndex(idx))
	}
	if prefix, ok := config.GetOption[string](cfg, "prefix"); ok {
		opts = append(opts, WithPrefix(prefix))
	}
	if dim, ok := config.GetOption[float64](cfg, "dimension"); ok {
		opts = append(opts, WithDimension(int(dim)))
	}

	return New(addr, opts...), nil
}

// EnsureIndex creates the RediSearch index if it does not exist.
func (s *Store) EnsureIndex(ctx context.Context) error {
	err := s.client.Do(ctx,
		"FT.CREATE", s.index,
		"ON", "HASH",
		"PREFIX", "1", s.prefix,
		"SCHEMA",
		"content", "TEXT",
		"embedding", "VECTOR", "FLAT", "6",
		"TYPE", "FLOAT32",
		"DIM", s.dimension,
		"DISTANCE_METRIC", "COSINE",
	).Err()

	if err != nil && strings.Contains(err.Error(), "Index already exists") {
		return nil
	}
	return err
}

// Add inserts documents with their embeddings into the store.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if len(docs) != len(embeddings) {
		return fmt.Errorf("redis: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}

	for i, doc := range docs {
		key := s.prefix + doc.ID
		fields := []any{
			"content", doc.Content,
			"embedding", float32ToBytes(embeddings[i]),
		}

		for k, v := range doc.Metadata {
			fields = append(fields, k, fmt.Sprintf("%v", v))
		}

		if err := s.client.HSet(ctx, key, fields...).Err(); err != nil {
			return fmt.Errorf("redis: hset %s: %w", key, err)
		}
	}

	return nil
}

// Search finds the k most similar documents to the query vector.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build the FT.SEARCH query for KNN vector search.
	filterExpr := "*"
	if len(cfg.Filter) > 0 {
		filters := make([]string, 0, len(cfg.Filter))
		for key, val := range cfg.Filter {
			filters = append(filters, fmt.Sprintf("@%s:{%v}", key, val))
		}
		filterExpr = strings.Join(filters, " ")
	}

	queryStr := fmt.Sprintf("(%s)=>[KNN %d @embedding $BLOB AS score]", filterExpr, k)

	args := []any{
		"FT.SEARCH", s.index, queryStr,
		"PARAMS", "2", "BLOB", float32ToBytes(query),
		"SORTBY", "score",
		"LIMIT", "0", strconv.Itoa(k),
		"DIALECT", "2",
	}

	result := s.client.Do(ctx, args...)
	if result.Err() != nil {
		return nil, fmt.Errorf("redis: search: %w", result.Err())
	}

	// Parse FT.SEARCH response.
	return parseFTSearchResult(result, s.prefix, cfg.Threshold)
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = s.prefix + id
	}

	return s.client.Del(ctx, keys...).Err()
}

// float32ToBytes converts a float32 slice to a byte slice for Redis.
func float32ToBytes(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// parseFTSearchResult parses the FT.SEARCH response into documents.
func parseFTSearchResult(cmd *goredis.Cmd, prefix string, threshold float64) ([]schema.Document, error) {
	raw, err := cmd.Slice()
	if err != nil {
		return nil, fmt.Errorf("redis: parse search result: %w", err)
	}

	if len(raw) < 1 {
		return nil, nil
	}

	// First element is the total count.
	total, ok := raw[0].(int64)
	if !ok {
		return nil, fmt.Errorf("redis: unexpected result format")
	}

	if total == 0 {
		return nil, nil
	}

	docs := make([]schema.Document, 0, total)

	// Results come in pairs: [key, [field, value, field, value, ...]]
	for i := 1; i < len(raw)-1; i += 2 {
		key, ok := raw[i].(string)
		if !ok {
			continue
		}

		fields, ok := raw[i+1].([]any)
		if !ok {
			continue
		}

		doc := schema.Document{
			ID:       strings.TrimPrefix(key, prefix),
			Metadata: make(map[string]any),
		}

		for j := 0; j < len(fields)-1; j += 2 {
			fieldName, ok := fields[j].(string)
			if !ok {
				continue
			}
			fieldVal := fields[j+1]

			switch fieldName {
			case "content":
				if v, ok := fieldVal.(string); ok {
					doc.Content = v
				}
			case "score":
				if v, ok := fieldVal.(string); ok {
					if score, err := strconv.ParseFloat(v, 64); err == nil {
						doc.Score = 1.0 - score // Convert cosine distance to similarity.
					}
				}
			case "embedding":
				// Skip binary embedding data.
			default:
				if v, ok := fieldVal.(string); ok {
					doc.Metadata[fieldName] = v
				}
			}
		}

		if threshold > 0 && doc.Score < threshold {
			continue
		}

		docs = append(docs, doc)
	}

	return docs, nil
}
