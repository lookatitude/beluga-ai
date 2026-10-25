package composio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecute_EscapesActionName: a hostile action name from the API must not
// escape the /actions/{name}/execute path segment.
func TestExecute_EscapesActionName(t *testing.T) {
	var gotRawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(executeResponse{Data: "ok", Successful: true})
	}))
	defer srv.Close()

	c, err := New(WithBaseURL(srv.URL), WithAPIKey("cmp-test"))
	require.NoError(t, err)
	ct := &composioTool{client: c, info: actionInfo{Name: "../../admin/keys?x=1#"}}
	_, err = ct.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/actions/..%2F..%2Fadmin%2Fkeys%3Fx=1%23/execute", gotRawPath)
}

// TestExecute_OutputIsJSON: structured output must be JSON, not Go %v syntax.
func TestExecute_OutputIsJSON(t *testing.T) {
	tests := []struct {
		name string
		data any
		want string
	}{
		{"object", map[string]any{"a": 1, "b": []any{"x"}}, `{"a":1,"b":["x"]}`},
		{"string", "plain text", "plain text"},
		{"null", nil, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(executeResponse{Data: tt.data, Successful: true})
			}))
			defer srv.Close()
			c, err := New(WithBaseURL(srv.URL), WithAPIKey("cmp-test"))
			require.NoError(t, err)
			res, err := (&composioTool{client: c, info: actionInfo{Name: "a"}}).Execute(context.Background(), nil)
			require.NoError(t, err)
			require.Len(t, res.Content, 1)
			assert.Equal(t, tt.want, res.Content[0].(schema.TextPart).Text)
		})
	}
}

func TestFormatOutput_Unmarshalable(t *testing.T) {
	_, err := formatOutput(func() {})
	assert.Error(t, err)
}
