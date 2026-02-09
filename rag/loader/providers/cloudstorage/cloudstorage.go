// Package cloudstorage provides a DocumentLoader that loads files from cloud
// storage services (S3, GCS, Azure Blob). It detects the provider by URL prefix
// (s3://, gs://, az://) and uses direct HTTP calls with pre-signed URLs or
// service-specific APIs.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/cloudstorage"
//
//	l, err := loader.New("cloudstorage", config.ProviderConfig{
//	    APIKey: "your-access-key",
//	    Options: map[string]any{
//	        "secret_key": "your-secret-key",
//	        "region":     "us-east-1",
//	    },
//	})
//	docs, err := l.Load(ctx, "s3://bucket/path/to/file.txt")
package cloudstorage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	loader.Register("cloudstorage", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// Loader loads files from cloud storage services.
type Loader struct {
	httpClient *http.Client
	accessKey  string
	secretKey  string
	region     string
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new cloud storage document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	secretKey, _ := config.GetOption[string](cfg, "secret_key")
	region, _ := config.GetOption[string](cfg, "region")
	if region == "" {
		region = "us-east-1"
	}

	return &Loader{
		httpClient: &http.Client{Timeout: timeout},
		accessKey:  cfg.APIKey,
		secretKey:  secretKey,
		region:     region,
	}, nil
}

// parseSource parses the storage URL and returns provider, bucket, and key.
func parseSource(source string) (provider, bucket, key string, err error) {
	switch {
	case strings.HasPrefix(source, "s3://"):
		parts := strings.SplitN(strings.TrimPrefix(source, "s3://"), "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			return "", "", "", fmt.Errorf("cloudstorage: invalid S3 URL %q (expected s3://bucket/key)", source)
		}
		return "s3", parts[0], parts[1], nil

	case strings.HasPrefix(source, "gs://"):
		parts := strings.SplitN(strings.TrimPrefix(source, "gs://"), "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			return "", "", "", fmt.Errorf("cloudstorage: invalid GCS URL %q (expected gs://bucket/key)", source)
		}
		return "gcs", parts[0], parts[1], nil

	case strings.HasPrefix(source, "az://"):
		parts := strings.SplitN(strings.TrimPrefix(source, "az://"), "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			return "", "", "", fmt.Errorf("cloudstorage: invalid Azure URL %q (expected az://container/blob)", source)
		}
		return "azure", parts[0], parts[1], nil

	default:
		return "", "", "", fmt.Errorf("cloudstorage: unsupported URL scheme in %q (expected s3://, gs://, or az://)", source)
	}
}

// buildURL constructs the HTTP URL for the storage object.
func (l *Loader) buildURL(provider, bucket, key string) string {
	switch provider {
	case "s3":
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, l.region, key)
	case "gcs":
		return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, key)
	case "azure":
		// Azure Blob requires storage account in Options; fallback to bucket as account.
		return fmt.Sprintf("https://%s.blob.core.windows.net/%s", bucket, key)
	default:
		return ""
	}
}

// Load downloads a file from cloud storage and returns it as a document.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("cloudstorage: source URL is required")
	}

	provider, bucket, key, err := parseSource(source)
	if err != nil {
		return nil, err
	}

	url := l.buildURL(provider, bucket, key)
	if url == "" {
		return nil, fmt.Errorf("cloudstorage: unable to build URL for %q", source)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("cloudstorage: create request: %w", err)
	}

	// Add auth headers for GCS and Azure.
	switch provider {
	case "gcs":
		if l.accessKey != "" {
			req.Header.Set("Authorization", "Bearer "+l.accessKey)
		}
	case "azure":
		if l.accessKey != "" {
			req.Header.Set("x-ms-blob-type", "BlockBlob")
			req.Header.Set("Authorization", "Bearer "+l.accessKey)
		}
	case "s3":
		if l.accessKey != "" {
			// Simplified auth header for S3 - in production, use AWS SDK signing.
			req.Header.Set("Authorization", "Bearer "+l.accessKey)
		}
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudstorage: fetch %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloudstorage: fetch %q failed (status %d): %s", source, resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cloudstorage: read %q: %w", source, err)
	}

	content := string(data)
	if content == "" {
		return nil, nil
	}

	// Extract filename from key.
	filename := key
	if idx := strings.LastIndex(key, "/"); idx >= 0 {
		filename = key[idx+1:]
	}

	meta := map[string]any{
		"source":   source,
		"loader":   "cloudstorage",
		"provider": provider,
		"bucket":   bucket,
		"key":      key,
		"filename": filename,
	}

	return []schema.Document{{
		ID:       source,
		Content:  content,
		Metadata: meta,
	}}, nil
}
