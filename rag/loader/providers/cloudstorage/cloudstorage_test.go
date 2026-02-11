package cloudstorage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		bucket   string
		key      string
		region   string
		expected string
	}{
		{
			name:     "s3 url",
			provider: "s3",
			bucket:   "my-bucket",
			key:      "path/file.txt",
			region:   "us-west-2",
			expected: "https://my-bucket.s3.us-west-2.amazonaws.com/path/file.txt",
		},
		{
			name:     "gcs url",
			provider: "gcs",
			bucket:   "gcs-bucket",
			key:      "doc.pdf",
			region:   "us-east-1",
			expected: "https://storage.googleapis.com/gcs-bucket/doc.pdf",
		},
		{
			name:     "azure url",
			provider: "azure",
			bucket:   "container",
			key:      "blob/data.csv",
			region:   "us-east-1",
			expected: "https://container.blob.core.windows.net/blob/data.csv",
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			bucket:   "bucket",
			key:      "key",
			region:   "us-east-1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Loader{region: tt.region}
			result := l.buildURL(tt.provider, tt.bucket, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadEdgeCases(t *testing.T) {
	// Note: buildURL returning empty is tested via TestBuildURL.
	// In the actual Load flow, parseSource validates schemes before buildURL is called,
	// so the "unknown provider" path in buildURL is defensive programming
	// and covered by the unit test.

	t.Run("s3 with auth headers", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer s3-token", req.Header.Get("Authorization"))
			w := httptest.NewRecorder()
			w.Write([]byte("s3 authed content"))
			return w.Result(), nil
		}}
		l := &Loader{
			httpClient: &http.Client{Transport: transport},
			accessKey:  "s3-token",
			region:     "us-east-1",
		}

		docs, err := l.Load(context.Background(), "s3://bucket/file.txt")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "s3 authed content", docs[0].Content)
	})

	t.Run("azure with auth headers", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer azure-token", req.Header.Get("Authorization"))
			assert.Equal(t, "BlockBlob", req.Header.Get("x-ms-blob-type"))
			w := httptest.NewRecorder()
			w.Write([]byte("azure authed content"))
			return w.Result(), nil
		}}
		l := &Loader{
			httpClient: &http.Client{Transport: transport},
			accessKey:  "azure-token",
			region:     "us-east-1",
		}

		docs, err := l.Load(context.Background(), "az://container/blob.txt")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "azure authed content", docs[0].Content)
	})

	t.Run("http client error", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			return nil, http.ErrHandlerTimeout
		}}
		l := &Loader{
			httpClient: &http.Client{Transport: transport},
			region:     "us-east-1",
		}

		_, err := l.Load(context.Background(), "s3://bucket/file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetch")
	})

	t.Run("filename without path", func(t *testing.T) {
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			w.Write([]byte("content"))
			return w.Result(), nil
		}}
		l := &Loader{
			httpClient: &http.Client{Transport: transport},
			region:     "us-east-1",
		}

		docs, err := l.Load(context.Background(), "s3://bucket/file.txt")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "file.txt", docs[0].Metadata["filename"])
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		l := &Loader{
			httpClient: &http.Client{},
			region:     "us-east-1",
		}

		_, err := l.Load(ctx, "s3://bucket/file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("read body error", func(t *testing.T) {
		// Create a response with a body that errors on read
		transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			w.WriteHeader(http.StatusOK)
			resp := w.Result()
			// Replace body with error reader
			resp.Body = &errorReader{}
			return resp, nil
		}}
		l := &Loader{
			httpClient: &http.Client{Transport: transport},
			region:     "us-east-1",
		}

		_, err := l.Load(context.Background(), "s3://bucket/file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read")
	})
}

func TestNewWithOptions(t *testing.T) {
	t.Run("custom timeout", func(t *testing.T) {
		cfg := config.ProviderConfig{
			Timeout: 30 * time.Second,
		}
		l, err := New(cfg)
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, l.httpClient.Timeout)
	})

	t.Run("custom region", func(t *testing.T) {
		cfg := config.ProviderConfig{
			Options: map[string]any{
				"region": "eu-west-1",
			},
		}
		l, err := New(cfg)
		require.NoError(t, err)
		assert.Equal(t, "eu-west-1", l.region)
	})

	t.Run("secret key option", func(t *testing.T) {
		cfg := config.ProviderConfig{
			APIKey: "access-key",
			Options: map[string]any{
				"secret_key": "secret-key",
			},
		}
		l, err := New(cfg)
		require.NoError(t, err)
		assert.Equal(t, "access-key", l.accessKey)
		assert.Equal(t, "secret-key", l.secretKey)
	})

	t.Run("default region", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)
		assert.Equal(t, "us-east-1", l.region)
	})

	t.Run("default timeout", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)
		assert.Equal(t, 60*time.Second, l.httpClient.Timeout)
	})
}

func TestParseSourceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantErr  string
	}{
		{
			name:    "s3 with empty key",
			source:  "s3://bucket/",
			wantErr: "invalid S3 URL",
		},
		{
			name:    "gs with empty key",
			source:  "gs://bucket/",
			wantErr: "invalid GCS URL",
		},
		{
			name:    "az with empty key",
			source:  "az://container/",
			wantErr: "invalid Azure URL",
		},
		{
			name:    "s3 missing key entirely",
			source:  "s3://bucket",
			wantErr: "invalid S3 URL",
		},
		{
			name:    "unsupported ftp scheme",
			source:  "ftp://server/file.txt",
			wantErr: "unsupported URL scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := parseSource(tt.source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// testTransport is a custom HTTP transport for testing.
type testTransport struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.handler(req)
}

// errorReader is an io.ReadCloser that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, http.ErrBodyReadAfterClose
}

func (e *errorReader) Close() error {
	return nil
}
