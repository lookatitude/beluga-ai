package mongodb

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore/internal/filterutil"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func init() {
	vectorstore.Register("mongodb", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
		return NewFromConfig(cfg)
	})
}

// HTTPClient abstracts HTTP calls.
//
// Deprecated: the store no longer speaks HTTP. MongoDB retired the Atlas
// Data API on 30 September 2025; the store now uses the official Go driver.
// The type is kept only so existing code that references it still compiles.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Collection is the subset of *mongo.Collection used by the store. It lets
// tests substitute a fake; production code passes a *mongo.Collection.
type Collection interface {
	BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...options.Lister[options.BulkWriteOptions]) (*mongo.BulkWriteResult, error)
	Aggregate(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) (*mongo.Cursor, error)
	DeleteMany(ctx context.Context, filter any, opts ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error)
}

// Store is a VectorStore backed by MongoDB Atlas Vector Search, using the
// official Go driver and the $vectorSearch aggregation stage.
type Store struct {
	coll       Collection
	client     *mongo.Client // owned client, closed by Close; nil if caller-supplied
	initErr    error
	database   string
	collection string
	index      string
}

// Compile-time interface check.
var _ vectorstore.VectorStore = (*Store)(nil)

// Option configures a Store.
type Option func(*Store)

// WithDatabase sets the database name.
func WithDatabase(name string) Option {
	return func(s *Store) { s.database = name }
}

// WithCollection sets the collection name.
func WithCollection(name string) Option {
	return func(s *Store) { s.collection = name }
}

// WithIndex sets the vector search index name.
func WithIndex(name string) Option {
	return func(s *Store) { s.index = name }
}

// WithAPIKey set the Atlas Data API key.
//
// Deprecated: the Atlas Data API was retired; this option is ignored.
// Put credentials in the MongoDB connection string instead.
func WithAPIKey(string) Option {
	return func(*Store) {}
}

// WithHTTPClient set the Data API HTTP client.
//
// Deprecated: the Atlas Data API was retired; this option is ignored.
func WithHTTPClient(HTTPClient) Option {
	return func(*Store) {}
}

func defaults() *Store {
	return &Store{
		database:   "beluga",
		collection: "documents",
		index:      "vector_index",
	}
}

// NewWithCollection creates a Store over an existing collection handle (for
// example client.Database("db").Collection("docs")). The caller owns the
// client. WithDatabase and WithCollection are ignored.
func NewWithCollection(coll Collection, opts ...Option) *Store {
	s := defaults()
	for _, opt := range opts {
		opt(s)
	}
	s.coll = coll
	if coll == nil {
		s.initErr = core.NewError("mongodb.new", core.ErrInvalidInput, "collection is required", nil)
	}
	return s
}

// New creates a Store from a MongoDB connection string
// ("mongodb://..." or "mongodb+srv://..."). The driver connects lazily; call
// Close to release the client.
//
// A URL that is not a MongoDB connection string (such as a retired Atlas
// Data API endpoint) yields a Store whose methods all return a
// core.ErrInvalidInput error explaining the migration. Prefer
// NewFromConfig or NewWithCollection, which report errors directly.
func New(uri string, opts ...Option) *Store {
	s, err := newFromURI(uri, opts...)
	if err != nil {
		s = defaults()
		s.initErr = err
	}
	return s
}

// NewFromConfig creates a Store from a ProviderConfig. BaseURL is the
// MongoDB connection string.
func NewFromConfig(cfg config.ProviderConfig) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("mongodb: base_url is required")
	}
	var opts []Option
	if db, ok := config.GetOption[string](cfg, "database"); ok {
		opts = append(opts, WithDatabase(db))
	}
	if col, ok := config.GetOption[string](cfg, "collection"); ok {
		opts = append(opts, WithCollection(col))
	}
	if idx, ok := config.GetOption[string](cfg, "index"); ok {
		opts = append(opts, WithIndex(idx))
	}
	return newFromURI(cfg.BaseURL, opts...)
}

func newFromURI(uri string, opts ...Option) (*Store, error) {
	if !strings.HasPrefix(uri, "mongodb://") && !strings.HasPrefix(uri, "mongodb+srv://") {
		return nil, core.NewError("mongodb.new", core.ErrInvalidInput,
			"base_url must be a MongoDB connection string (mongodb:// or mongodb+srv://); "+
				"the Atlas Data API was retired on 2025-09-30 and is no longer supported", nil)
	}
	s := defaults()
	for _, opt := range opts {
		opt(s)
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, core.NewError("mongodb.new", core.ErrInvalidInput, "connect", err)
	}
	s.client = client
	s.coll = client.Database(s.database).Collection(s.collection)
	return s, nil
}

// Close disconnects the client created by New or NewFromConfig. It is a
// no-op for a Store built with NewWithCollection.
func (s *Store) Close(ctx context.Context) error {
	if s.client == nil {
		return nil
	}
	err := s.client.Disconnect(ctx)
	s.client = nil
	return err
}

// Add upserts documents with their embeddings, keyed by document ID.
func (s *Store) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if s.initErr != nil {
		return s.initErr
	}
	if len(docs) != len(embeddings) {
		return fmt.Errorf("mongodb: docs length (%d) != embeddings length (%d)", len(docs), len(embeddings))
	}
	if len(docs) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(docs))
	for i, doc := range docs {
		d := bson.M{
			"_id":       doc.ID,
			"content":   doc.Content,
			"embedding": float32SliceToFloat64(embeddings[i]),
		}
		if doc.Metadata != nil {
			d["metadata"] = doc.Metadata
		}
		models[i] = mongo.NewReplaceOneModel().
			SetFilter(bson.M{"_id": doc.ID}).
			SetReplacement(d).
			SetUpsert(true)
	}
	if _, err := s.coll.BulkWrite(ctx, models); err != nil {
		return core.NewError("mongodb.add", core.ErrProviderDown, "bulk write", err)
	}
	return nil
}

// Search finds the k most similar documents to the query vector using
// $vectorSearch. The similarity function is defined by the Atlas vector
// index, so SearchConfig.Strategy is not used. Filter keys must be plain
// identifiers and values scalars; they match fields under "metadata" and
// must be declared as filter fields in the index.
func (s *Store) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	if s.initErr != nil {
		return nil, s.initErr
	}
	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	pipeline, err := buildPipeline(s.index, query, k, cfg.Filter)
	if err != nil {
		return nil, err
	}

	cur, err := s.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, core.NewError("mongodb.search", core.ErrProviderDown, "aggregate", err)
	}
	var rows []searchRow
	if err := cur.All(ctx, &rows); err != nil {
		return nil, core.NewError("mongodb.search", core.ErrProviderDown, "decode results", err)
	}

	docs := make([]schema.Document, 0, len(rows))
	for _, r := range rows {
		if cfg.Threshold > 0 && r.Score < cfg.Threshold {
			continue
		}
		docs = append(docs, schema.Document{
			ID:       r.ID,
			Content:  r.Content,
			Score:    r.Score,
			Metadata: r.Metadata,
		})
	}
	return docs, nil
}

// searchRow is one $vectorSearch result.
type searchRow struct {
	ID       string         `bson:"_id"`
	Content  string         `bson:"content"`
	Metadata map[string]any `bson:"metadata"`
	Score    float64        `bson:"score"`
}

// buildPipeline builds the $vectorSearch aggregation pipeline.
func buildPipeline(index string, query []float32, k int, filter map[string]any) (mongo.Pipeline, error) {
	keys, err := filterutil.Validate("mongodb.search", filter)
	if err != nil {
		return nil, err
	}
	vs := bson.D{
		{Key: "index", Value: index},
		{Key: "path", Value: "embedding"},
		{Key: "queryVector", Value: float32SliceToFloat64(query)},
		{Key: "numCandidates", Value: k * 10},
		{Key: "limit", Value: k},
	}
	if len(keys) > 0 {
		f := bson.D{}
		for _, key := range keys {
			// Wrap in $eq so a value can never act as an operator.
			f = append(f, bson.E{Key: "metadata." + key, Value: bson.D{{Key: "$eq", Value: filter[key]}}})
		}
		vs = append(vs, bson.E{Key: "filter", Value: f})
	}
	return mongo.Pipeline{
		{{Key: "$vectorSearch", Value: vs}},
		{{Key: "$project", Value: bson.D{
			{Key: "content", Value: 1},
			{Key: "metadata", Value: 1},
			{Key: "score", Value: bson.D{{Key: "$meta", Value: "vectorSearchScore"}}},
		}}},
	}, nil
}

// Delete removes documents with the given IDs from the store.
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if s.initErr != nil {
		return s.initErr
	}
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.coll.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}}); err != nil {
		return core.NewError("mongodb.delete", core.ErrProviderDown, "delete", err)
	}
	return nil
}

// float32SliceToFloat64 converts []float32 to []float64 for BSON encoding.
func float32SliceToFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}
