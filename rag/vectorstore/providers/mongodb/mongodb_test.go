package mongodb

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCollection records calls and returns canned results.
type fakeCollection struct {
	models   []mongo.WriteModel
	pipeline any
	filter   any
	results  []any
	err      error
}

func (f *fakeCollection) BulkWrite(_ context.Context, models []mongo.WriteModel, _ ...options.Lister[options.BulkWriteOptions]) (*mongo.BulkWriteResult, error) {
	f.models = models
	return &mongo.BulkWriteResult{}, f.err
}

func (f *fakeCollection) Aggregate(_ context.Context, pipeline any, _ ...options.Lister[options.AggregateOptions]) (*mongo.Cursor, error) {
	f.pipeline = pipeline
	if f.err != nil {
		return nil, f.err
	}
	return mongo.NewCursorFromDocuments(f.results, nil, nil)
}

func (f *fakeCollection) DeleteMany(_ context.Context, filter any, _ ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error) {
	f.filter = filter
	return &mongo.DeleteResult{}, f.err
}

var _ Collection = (*fakeCollection)(nil)

func requireCode(t *testing.T, err error, code core.ErrorCode) {
	t.Helper()
	var ce *core.Error
	require.True(t, errors.As(err, &ce), "want core.Error, got %v", err)
	assert.Equal(t, code, ce.Code)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	coll := &fakeCollection{}
	s := NewWithCollection(coll)
	err := s.Add(context.Background(),
		[]schema.Document{{ID: "a", Content: "x", Metadata: map[string]any{"k": "v"}}, {ID: "b"}},
		[][]float32{{1, 2}, {3, 4}})
	require.NoError(t, err)
	require.Len(t, coll.models, 2)
	m, ok := coll.models[0].(*mongo.ReplaceOneModel)
	require.True(t, ok)
	assert.Equal(t, bson.M{"_id": "a"}, m.Filter)
	assert.True(t, *m.Upsert)
	repl := m.Replacement.(bson.M)
	assert.Equal(t, []float64{1, 2}, repl["embedding"])
	assert.Equal(t, map[string]any{"k": "v"}, repl["metadata"])
}

func TestStore_Add_Errors(t *testing.T) {
	s := NewWithCollection(&fakeCollection{})
	require.Error(t, s.Add(context.Background(), []schema.Document{{ID: "a"}}, nil))
	require.NoError(t, s.Add(context.Background(), nil, nil))

	s = NewWithCollection(&fakeCollection{err: errors.New("boom")})
	requireCode(t, s.Add(context.Background(), []schema.Document{{ID: "a"}}, [][]float32{{1}}), core.ErrProviderDown)
}

func TestStore_Search(t *testing.T) {
	coll := &fakeCollection{results: []any{
		bson.D{{Key: "_id", Value: "a"}, {Key: "content", Value: "hello"}, {Key: "score", Value: 0.9},
			{Key: "metadata", Value: bson.D{{Key: "k", Value: "v"}}}},
		bson.D{{Key: "_id", Value: "b"}, {Key: "content", Value: "low"}, {Key: "score", Value: 0.1}},
	}}
	s := NewWithCollection(coll, WithIndex("idx"))
	docs, err := s.Search(context.Background(), []float32{1, 2}, 2,
		vectorstore.WithFilter(map[string]any{"k": "v"}), vectorstore.WithThreshold(0.5))
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "a", docs[0].ID)
	assert.Equal(t, "hello", docs[0].Content)
	assert.InDelta(t, 0.9, docs[0].Score, 1e-9)
	assert.Equal(t, "v", docs[0].Metadata["k"])

	p := coll.pipeline.(mongo.Pipeline)
	vs := p[0][0].Value.(bson.D)
	assert.Equal(t, "$vectorSearch", p[0][0].Key)
	assert.Contains(t, vs, bson.E{Key: "index", Value: "idx"})
	assert.Contains(t, vs, bson.E{Key: "limit", Value: 2})
	assert.Contains(t, vs, bson.E{Key: "filter", Value: bson.D{
		{Key: "metadata.k", Value: bson.D{{Key: "$eq", Value: "v"}}},
	}})
}

func TestStore_Search_Errors(t *testing.T) {
	s := NewWithCollection(&fakeCollection{err: errors.New("boom")})
	_, err := s.Search(context.Background(), []float32{1}, 1)
	requireCode(t, err, core.ErrProviderDown)

	// Operator keys and operator-object values must be rejected.
	s = NewWithCollection(&fakeCollection{})
	for _, f := range []map[string]any{{"$where": "1"}, {"k": bson.M{"$ne": "x"}}, {"a.b": 1}} {
		_, err = s.Search(context.Background(), []float32{1}, 1, vectorstore.WithFilter(f))
		requireCode(t, err, core.ErrInvalidInput)
	}

	// Undecodable results.
	s = NewWithCollection(&fakeCollection{results: []any{bson.D{{Key: "_id", Value: 5}}}})
	_, err = s.Search(context.Background(), []float32{1}, 1)
	requireCode(t, err, core.ErrProviderDown)
}

func TestStore_Delete(t *testing.T) {
	coll := &fakeCollection{}
	s := NewWithCollection(coll)
	require.NoError(t, s.Delete(context.Background(), nil))
	assert.Nil(t, coll.filter)
	require.NoError(t, s.Delete(context.Background(), []string{"a", "b"}))
	assert.Equal(t, bson.M{"_id": bson.M{"$in": []string{"a", "b"}}}, coll.filter)

	s = NewWithCollection(&fakeCollection{err: errors.New("boom")})
	requireCode(t, s.Delete(context.Background(), []string{"a"}), core.ErrProviderDown)
}

// FI-14: the retired Atlas Data API must produce a clear error, not
// silent HTTP calls to a dead endpoint.
func TestNew_DataAPIURLRejected(t *testing.T) {
	ctx := context.Background()
	s := New("https://data.mongodb-api.com/app/x/endpoint/data/v1", WithAPIKey("k"), WithHTTPClient(nil))
	requireCode(t, s.Add(ctx, []schema.Document{{ID: "a"}}, [][]float32{{1}}), core.ErrInvalidInput)
	_, err := s.Search(ctx, []float32{1}, 1)
	requireCode(t, err, core.ErrInvalidInput)
	requireCode(t, s.Delete(ctx, []string{"a"}), core.ErrInvalidInput)
	assert.Contains(t, s.Add(ctx, nil, nil).Error(), "Data API")
	require.NoError(t, s.Close(ctx))

	_, err = NewFromConfig(config.ProviderConfig{BaseURL: "https://data.mongodb-api.com/app/x"})
	requireCode(t, err, core.ErrInvalidInput)
}

func TestNewFromConfig(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)

	s, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=10",
		Options: map[string]any{"database": "db", "collection": "c", "index": "i"},
	})
	require.NoError(t, err)
	assert.Equal(t, "db", s.database)
	assert.Equal(t, "c", s.collection)
	assert.Equal(t, "i", s.index)
	require.NotNil(t, s.client)
	require.NoError(t, s.Close(context.Background()))
	require.NoError(t, s.Close(context.Background()))

	_, err = NewFromConfig(config.ProviderConfig{BaseURL: "mongodb://%zz"})
	requireCode(t, err, core.ErrInvalidInput)
}

func TestNew_ConnectionString(t *testing.T) {
	s := New("mongodb+srv://user:pw@cluster.example.invalid/?connect=direct&serverSelectionTimeoutMS=10")
	// mongodb+srv resolves DNS at connect time; either outcome must be a
	// usable store or a typed configuration error, never a panic.
	if s.initErr != nil {
		requireCode(t, s.initErr, core.ErrInvalidInput)
		return
	}
	require.NoError(t, s.Close(context.Background()))
}

func TestNewWithCollection_Nil(t *testing.T) {
	s := NewWithCollection(nil)
	requireCode(t, s.Delete(context.Background(), []string{"a"}), core.ErrInvalidInput)
}

func TestRegistry(t *testing.T) {
	assert.Contains(t, vectorstore.List(), "mongodb")
}
