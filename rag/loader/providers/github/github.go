// Package github provides a DocumentLoader that loads files from GitHub
// repositories via the GitHub API. It implements the loader.DocumentLoader
// interface using direct HTTP calls.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/github"
//
//	l, err := loader.New("github", config.ProviderConfig{
//	    APIKey: "ghp_...",
//	})
//	docs, err := l.Load(ctx, "owner/repo/path/to/file.go")
package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

const defaultBaseURL = "https://api.github.com"

func init() {
	loader.Register("github", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// Loader loads files from GitHub repositories via the API.
type Loader struct {
	client *httpclient.Client
	ref    string
}

// New creates a new GitHub document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(baseURL),
		httpclient.WithHeader("Accept", "application/vnd.github.v3+json"),
		httpclient.WithTimeout(timeout),
	}
	if cfg.APIKey != "" {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.APIKey))
	}

	ref, _ := config.GetOption[string](cfg, "ref")

	return &Loader{
		client: httpclient.New(clientOpts...),
		ref:    ref,
	}, nil
}

// contentResponse is the GitHub contents API response.
type contentResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	DownloadURL string `json:"download_url"`
	HTMLURL     string `json:"html_url"`
}

// Load fetches a file from a GitHub repository. The source format is
// "owner/repo/path/to/file". An optional ref (branch/tag/SHA) can be set
// via Options["ref"] in the config.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("github: source is required (format: owner/repo/path)")
	}

	parts := strings.SplitN(source, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("github: source must be in format owner/repo/path, got %q", source)
	}
	owner, repo, filePath := parts[0], parts[1], parts[2]

	path := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, filePath)
	if l.ref != "" {
		path += "?ref=" + l.ref
	}

	resp, err := httpclient.DoJSON[contentResponse](ctx, l.client, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("github: fetch %q: %w", source, err)
	}

	if resp.Type != "file" {
		return nil, fmt.Errorf("github: %q is a %s, not a file", source, resp.Type)
	}

	content, err := decodeContent(resp.Content, resp.Encoding)
	if err != nil {
		return nil, fmt.Errorf("github: decode %q: %w", source, err)
	}

	if content == "" {
		return nil, nil
	}

	meta := map[string]any{
		"source":       source,
		"loader":       "github",
		"path":         resp.Path,
		"sha":          resp.SHA,
		"size":         resp.Size,
		"html_url":     resp.HTMLURL,
		"download_url": resp.DownloadURL,
	}

	return []schema.Document{{
		ID:       fmt.Sprintf("%s/%s/%s", owner, repo, resp.Path),
		Content:  content,
		Metadata: meta,
	}}, nil
}

// decodeContent decodes the file content based on the encoding.
func decodeContent(content, encoding string) (string, error) {
	if encoding == "base64" || encoding == "" {
		// GitHub returns base64-encoded content with embedded newlines.
		cleaned := strings.ReplaceAll(content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return "", fmt.Errorf("base64 decode: %w", err)
		}
		return string(decoded), nil
	}
	return content, nil
}
