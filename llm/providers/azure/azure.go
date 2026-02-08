// Package azure provides the Azure OpenAI LLM provider for the Beluga AI framework.
// Azure OpenAI uses a different authentication scheme (api-key header) and URL
// structure (per-deployment endpoints with api-version query param) compared to
// the standard OpenAI API, but the request/response format is otherwise identical.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/azure"
//
//	model, err := llm.New("azure", config.ProviderConfig{
//	    Model:   "gpt-4o",
//	    APIKey:  "...",
//	    BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
//	    Options: map[string]any{"api_version": "2024-10-21"},
//	})
package azure

import (
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/openai/openai-go/option"
)

const defaultAPIVersion = "2024-10-21"

func init() {
	llm.Register("azure", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Azure OpenAI ChatModel.
// The cfg.BaseURL must be set to the Azure deployment endpoint, e.g.:
//
//	https://{resource}.openai.azure.com/openai/deployments/{deployment}
//
// The api-version is read from cfg.Options["api_version"] (default: "2024-10-21").
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("azure: base_url is required (format: https://{resource}.openai.azure.com/openai/deployments/{deployment})")
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}

	apiVersion := defaultAPIVersion
	if v, ok := config.GetOption[string](cfg, "api_version"); ok && v != "" {
		apiVersion = v
	}

	// Azure uses api-key header instead of Bearer token, and requires
	// api-version query parameter on all requests. We use middleware to
	// strip the default Authorization header and inject Azure-specific headers.
	apiKey := cfg.APIKey
	cfg.APIKey = "" // Prevent openaicompat from setting Bearer auth
	extraOpts := []option.RequestOption{
		option.WithMiddleware(func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
			req.Header.Del("Authorization")
			req.Header.Set("api-key", apiKey)
			q := req.URL.Query()
			q.Set("api-version", apiVersion)
			req.URL.RawQuery = q.Encode()
			return next(req)
		}),
	}

	return openaicompat.NewWithOptions(cfg, extraOpts...)
}
