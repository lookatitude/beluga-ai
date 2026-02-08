package composio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, err := New(WithAPIKey("cmp-test"))
		require.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := New()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key")
	})

	t.Run("with options", func(t *testing.T) {
		c, err := New(
			WithBaseURL("https://custom.composio.dev"),
			WithAPIKey("cmp-test"),
			WithTimeout(5*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, c)
	})
}

func TestListTools(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/actions", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "cmp-test", r.Header.Get("x-api-key"))

			resp := actionsResponse{
				Items: []actionInfo{
					{Name: "github_create_issue", DisplayName: "Create Issue", Description: "Create a GitHub issue", AppName: "github"},
					{Name: "slack_send_message", DisplayName: "Send Message", Description: "Send a Slack message", AppName: "slack"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		c, err := New(WithBaseURL(srv.URL), WithAPIKey("cmp-test"))
		require.NoError(t, err)

		tools, err := c.ListTools(context.Background())
		require.NoError(t, err)
		assert.Len(t, tools, 2)
		assert.Equal(t, "github_create_issue", tools[0].Name())
		assert.Equal(t, "Create a GitHub issue", tools[0].Description())
		assert.Equal(t, "slack_send_message", tools[1].Name())
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"message":"server error"}}`))
		}))
		defer srv.Close()

		c, err := New(WithBaseURL(srv.URL), WithAPIKey("cmp-test"))
		require.NoError(t, err)

		_, err = c.ListTools(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "composio")
	})
}

func TestExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/actions" {
				resp := actionsResponse{
					Items: []actionInfo{
						{Name: "test_action", Description: "Test action"},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}

			assert.Equal(t, "/api/v1/actions/test_action/execute", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			resp := executeResponse{
				Data:       "result data",
				Successful: true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		c, err := New(WithBaseURL(srv.URL), WithAPIKey("cmp-test"))
		require.NoError(t, err)

		tools, err := c.ListTools(context.Background())
		require.NoError(t, err)
		require.Len(t, tools, 1)

		result, err := tools[0].Execute(context.Background(), map[string]any{"key": "value"})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
	})

	t.Run("execution failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/actions" {
				resp := actionsResponse{
					Items: []actionInfo{{Name: "fail_action", Description: "Fail"}},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}

			resp := executeResponse{
				Error:      "permission denied",
				Successful: false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		c, err := New(WithBaseURL(srv.URL), WithAPIKey("cmp-test"))
		require.NoError(t, err)

		tools, err := c.ListTools(context.Background())
		require.NoError(t, err)

		result, err := tools[0].Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}
