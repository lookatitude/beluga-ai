package cloudstorage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FI-04: a bucket containing '#', '?', '@' or ':' must not move the host,
// and credentials must never be sent anywhere else.
func TestLoad_HostInjectionRejected(t *testing.T) {
	sent := false
	transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
		sent = true
		return httptest.NewRecorder().Result(), nil
	}}
	l := &Loader{httpClient: &http.Client{Transport: transport}, accessKey: "tok", secretKey: "s", region: "us-east-1"}

	for _, src := range []string{
		"s3://169.254.169.254#/latest/meta-data",
		"s3://evil.com?/x",
		"s3://user@evil.com/x",
		"s3://evil.com:80/x",
		"gs://evil.com#/x",
		"az://evil.com#/c/b",
		"az://UPPER/c/b",
		"s3://ab/x",
	} {
		_, err := l.Load(context.Background(), src)
		require.Error(t, err, src)
		var ce *core.Error
		require.True(t, errors.As(err, &ce), src)
		assert.Equal(t, core.ErrInvalidInput, ce.Code, src)
	}
	assert.False(t, sent, "no request may be sent for an invalid source")
}

func TestBuildURL_KeyCannotChangeHost(t *testing.T) {
	l := &Loader{region: "us-east-1"}
	for _, tt := range []struct{ provider, bucket, host string }{
		{"s3", "bucket", "bucket.s3.us-east-1.amazonaws.com"},
		{"gcs", "bucket", "storage.googleapis.com"},
		{"azure", "account", "account.blob.core.windows.net"},
	} {
		raw := l.buildURL(tt.provider, tt.bucket, "a#b?c=@evil.com/../d")
		u, err := url.Parse(raw)
		require.NoError(t, err)
		assert.Equal(t, tt.host, u.Host)
		assert.Empty(t, u.RawQuery)
		assert.Empty(t, u.Fragment)
	}
}

func TestHostAllowed(t *testing.T) {
	assert.True(t, hostAllowed("s3", "b.s3.eu-west-1.amazonaws.com", "eu-west-1"))
	assert.False(t, hostAllowed("s3", "b.s3.eu-west-1.amazonaws.com", "us-east-1"))
	assert.False(t, hostAllowed("s3", "169.254.169.254", ""))
	assert.True(t, hostAllowed("gcs", "storage.googleapis.com", ""))
	assert.False(t, hostAllowed("gcs", "storage.googleapis.com.evil", ""))
	assert.True(t, hostAllowed("azure", "acct.blob.core.windows.net", ""))
	assert.False(t, hostAllowed("x", "acct.blob.core.windows.net", ""))
}

func TestCheckRedirect(t *testing.T) {
	mk := func(raw string) *http.Request {
		u, _ := url.Parse(raw)
		return &http.Request{URL: u}
	}
	require.NoError(t, checkRedirect(mk("https://b.s3.us-west-2.amazonaws.com/k"), nil))
	require.Error(t, checkRedirect(mk("https://evil.com/k"), nil))
	require.Error(t, checkRedirect(mk("http://storage.googleapis.com/k"), nil))
	require.Error(t, checkRedirect(mk("https://storage.googleapis.com/k"), make([]*http.Request, 5)))
}

func TestLoad_SizeLimit(t *testing.T) {
	transport := &testTransport{handler: func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		_, _ = w.WriteString(strings.Repeat("x", 11))
		return w.Result(), nil
	}}
	l := &Loader{httpClient: &http.Client{Transport: transport}, region: "us-east-1", maxBytes: 10}
	_, err := l.Load(context.Background(), "s3://bucket/k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")

	l.maxBytes = 11
	docs, err := l.Load(context.Background(), "s3://bucket/k")
	require.NoError(t, err)
	require.Len(t, docs, 1)
}

func TestLoad_S3RequiresSecretWithAccessKey(t *testing.T) {
	l := &Loader{httpClient: &http.Client{}, accessKey: "ak", region: "us-east-1"}
	_, err := l.Load(context.Background(), "s3://bucket/k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret_key")
}

func TestNew_OptionsAndValidation(t *testing.T) {
	l, err := New(config.ProviderConfig{Options: map[string]any{"max_bytes": float64(5), "session_token": "st"}})
	require.NoError(t, err)
	assert.Equal(t, int64(5), l.maxBytes)
	assert.Equal(t, "st", l.sessionToken)
	assert.NotNil(t, l.httpClient.CheckRedirect)

	_, err = New(config.ProviderConfig{Options: map[string]any{"region": "us-east-1.evil.com/"}})
	require.Error(t, err)
}
