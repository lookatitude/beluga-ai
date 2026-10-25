package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertCode(t *testing.T, err error, code core.ErrorCode) {
	t.Helper()
	var ce *core.Error
	require.True(t, errors.As(err, &ce), "want core.Error, got %v", err)
	assert.Equal(t, code, ce.Code)
}

// FI-01: a quote in a filter value must not break out of the literal.
func TestBuildWhereClause_ValueInjection(t *testing.T) {
	payload := `x"}]},{path:["tenant"],operator:NotEqual,valueText:"y`
	where, err := buildWhereClause(map[string]any{"k": payload})
	require.NoError(t, err)
	assert.NotContains(t, where, `valueText:"x"}`)
	// The whole payload is one escaped JSON string literal.
	lit, _ := json.Marshal(payload)
	assert.Equal(t, `, where: {path:["k"],operator:Equal,valueText:`+string(lit)+`}`, where)
}

func TestBuildWhereClause_TypedValues(t *testing.T) {
	where, err := buildWhereClause(map[string]any{"a": true, "b": 3, "c": 1.5, "d": "s"})
	require.NoError(t, err)
	assert.Equal(t, `, where: {operator:And,operands:[`+
		`{path:["a"],operator:Equal,valueBoolean:true},`+
		`{path:["b"],operator:Equal,valueInt:3},`+
		`{path:["c"],operator:Equal,valueNumber:1.5},`+
		`{path:["d"],operator:Equal,valueText:"s"}]}`, where)
}

func TestBuildWhereClause_InvalidKeyOrValue(t *testing.T) {
	for _, f := range []map[string]any{
		{`k"]`: "x"},
		{"a b": "x"},
		{"k": map[string]any{"x": 1}},
	} {
		_, err := buildWhereClause(f)
		assertCode(t, err, core.ErrInvalidInput)
	}
}

func TestStore_Search_InvalidFilterRejectedBeforeRequest(t *testing.T) {
	called := false
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	defer srv.Close()
	_, err := store.Search(context.Background(), []float32{1}, 1,
		vectorstore.WithFilter(map[string]any{`x"`: "y"}))
	assertCode(t, err, core.ErrInvalidInput)
	assert.False(t, called)
}

// FI-01: GraphQL errors arrive as HTTP 200 with a null class; they must
// surface instead of returning (nil, nil).
func TestStore_Search_GraphQLErrorsSurface(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"Get":{"TestDoc":null}},"errors":[{"message":"invalid where filter"}]}`))
	})
	defer srv.Close()
	_, err := store.Search(context.Background(), []float32{1}, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid where filter")
}

// FI-02: IDs sharing a 16-byte prefix must map to distinct UUIDs.
func TestUUIDFromID_NoPrefixCollision(t *testing.T) {
	a := uuidFromID("document-chunk-0001")
	b := uuidFromID("document-chunk-0002")
	assert.NotEqual(t, a, b)
	assert.Equal(t, a, uuidFromID("document-chunk-0001"))
	assert.Equal(t, byte('5'), a[14], "UUID version must be 5")
}

// FI-34: per-object batch errors in a 200 body must fail Add.
func TestStore_Add_BatchObjectErrors(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"u1","result":{}},{"id":"u2","result":{"errors":{"error":[{"message":"vector dimension mismatch"}]}}}]`))
	})
	defer srv.Close()
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "a"}, {ID: "b"}}, [][]float32{{1}, {2}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vector dimension mismatch")
}

func TestStore_Add_BatchUnparseable(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})
	defer srv.Close()
	err := store.Add(context.Background(), []schema.Document{{ID: "a"}}, [][]float32{{1}})
	require.Error(t, err)
}

// FI-20 (weaviate): metadata must not overwrite managed properties.
func TestStore_Add_ReservedMetadataKey(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request must not be sent")
	})
	defer srv.Close()
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "a", Metadata: map[string]any{"_beluga_id": "other"}}}, [][]float32{{1}})
	assertCode(t, err, core.ErrInvalidInput)
}

func TestStore_InvalidClassRejected(t *testing.T) {
	store := New("http://127.0.0.1:1", WithClass(`Doc(limit:1){x}}`))
	_, err := store.Search(context.Background(), []float32{1}, 1)
	assertCode(t, err, core.ErrInvalidInput)
	assertCode(t, store.Add(context.Background(), nil, nil), core.ErrInvalidInput)
	assertCode(t, store.Delete(context.Background(), []string{"a"}), core.ErrInvalidInput)
}

func TestNew_DefaultClientHasTimeout(t *testing.T) {
	store := New("http://localhost")
	c, ok := store.client.(*http.Client)
	require.True(t, ok)
	assert.Equal(t, defaultTimeout, c.Timeout)
}
