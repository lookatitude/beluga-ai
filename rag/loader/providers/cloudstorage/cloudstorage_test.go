package cloudstorage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistration(t *testing.T) {
	names := loader.List()
	found := false
	for _, n := range names {
		if n == "cloudstorage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("cloudstorage loader not registered")
	}
}

func TestNew(t *testing.T) {
	l, err := New(config.ProviderConfig{})
	require.NoError(t, err)
	assert.NotNil(t, l)
}

func TestParseSource(t *testing.T) {
	tests := []struct {
		source   string
		provider string
		bucket   string
		key      string
		hasError bool
	}{
		{"s3://my-bucket/path/file.txt", "s3", "my-bucket", "path/file.txt", false},
		{"gs://gcs-bucket/doc.pdf", "gcs", "gcs-bucket", "doc.pdf", false},
		{"az://container/blob/data.csv", "azure", "container", "blob/data.csv", false},
		{"s3://bucket-only/", "", "", "", true},
		{"gs://bucket-only", "", "", "", true},
		{"http://example.com/file", "", "", "", true},
		{"", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			provider, bucket, key, err := parseSource(tt.source)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.provider, provider)
				assert.Equal(t, tt.bucket, bucket)
				assert.Equal(t, tt.key, key)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("file content from cloud"))
		}))
		defer srv.Close()

		// Override buildURL to use test server.
		l := &Loader{
			httpClient: srv.Client(),
			region:     "us-east-1",
		}

		// We need to test via the actual URL, so we'll use a custom approach.
		// Instead, let's test the full flow using httptest with a custom transport.
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			w.Write([]byte("cloud file content"))
			return w.Result(), nil
		}}
		l.httpClient = &http.Client{Transport: transport}

		docs, err := l.Load(context.Background(), "s3://test-bucket/path/file.txt")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "cloud file content", docs[0].Content)
		assert.Equal(t, "cloudstorage", docs[0].Metadata["loader"])
		assert.Equal(t, "s3", docs[0].Metadata["provider"])
		assert.Equal(t, "test-bucket", docs[0].Metadata["bucket"])
		assert.Equal(t, "path/file.txt", docs[0].Metadata["key"])
		assert.Equal(t, "file.txt", docs[0].Metadata["filename"])
	})

	t.Run("gcs download", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Host, "storage.googleapis.com")
			w := httptest.NewRecorder()
			w.Write([]byte("gcs content"))
			return w.Result(), nil
		}}
		l := &Loader{httpClient: &http.Client{Transport: transport}, region: "us-east-1"}

		docs, err := l.Load(context.Background(), "gs://gcs-bucket/data.csv")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "gcs content", docs[0].Content)
		assert.Equal(t, "gcs", docs[0].Metadata["provider"])
	})

	t.Run("azure download", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Host, "blob.core.windows.net")
			w := httptest.NewRecorder()
			w.Write([]byte("azure content"))
			return w.Result(), nil
		}}
		l := &Loader{httpClient: &http.Client{Transport: transport}, region: "us-east-1"}

		docs, err := l.Load(context.Background(), "az://mycontainer/path/blob.txt")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "azure content", docs[0].Content)
		assert.Equal(t, "azure", docs[0].Metadata["provider"])
	})

	t.Run("empty source", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("invalid scheme", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "http://example.com/file")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported URL scheme")
	})

	t.Run("download error", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("access denied"))
			return w.Result(), nil
		}}
		l := &Loader{httpClient: &http.Client{Transport: transport}, region: "us-east-1"}

		_, err := l.Load(context.Background(), "s3://bucket/key.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("empty content", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			return w.Result(), nil
		}}
		l := &Loader{httpClient: &http.Client{Transport: transport}, region: "us-east-1"}

		docs, err := l.Load(context.Background(), "s3://bucket/empty.txt")
		require.NoError(t, err)
		assert.Nil(t, docs)
	})

	t.Run("with auth token", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.Header.Get("Authorization"), "Bearer test-token")
			w := httptest.NewRecorder()
			w.Write([]byte("authed content"))
			return w.Result(), nil
		}}
		l := &Loader{
			httpClient: &http.Client{Transport: transport},
			accessKey:  "test-token",
			region:     "us-east-1",
		}

		docs, err := l.Load(context.Background(), "gs://bucket/file.txt")
		require.NoError(t, err)
		require.Len(t, docs, 1)
	})
}

func TestRegistryNew(t *testing.T) {
	l, err := loader.New("cloudstorage", config.ProviderConfig{})
	require.NoError(t, err)
	assert.NotNil(t, l)
}

// testTransport is a custom HTTP transport for testing.
type testTransport struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.handler(req)
}
