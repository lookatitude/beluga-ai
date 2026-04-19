package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// OPAPolicy implements authorization via an external Open Policy Agent (OPA)
// server. It sends authorization requests to a configured OPA endpoint and
// evaluates the response.
//
// OPAPolicy is safe for concurrent use.
type OPAPolicy struct {
	name     string
	endpoint string
	client   *http.Client
	timeout  time.Duration
}

// OPAOption configures an OPAPolicy.
type OPAOption func(*OPAPolicy)

// WithTimeout sets the request timeout for OPA calls. Default is 5 seconds.
func WithTimeout(timeout time.Duration) OPAOption {
	return func(p *OPAPolicy) {
		p.timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client. Default creates a new client.
func WithHTTPClient(client *http.Client) OPAOption {
	return func(p *OPAPolicy) {
		p.client = client
	}
}

// NewOPAPolicy creates a new OPA policy that delegates authorization decisions
// to an external OPA endpoint. The endpoint should be the full URL to the OPA
// policy endpoint (e.g., "http://localhost:8181/v1/data/authz/allow").
func NewOPAPolicy(name string, endpoint string, opts ...OPAOption) *OPAPolicy {
	p := &OPAPolicy{
		name:     name,
		endpoint: endpoint,
		timeout:  5 * time.Second,
		client:   &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the policy name.
func (p *OPAPolicy) Name() string { return p.name }

// opaRequest is the JSON structure sent to OPA.
type opaRequest struct {
	Subject    string `json:"subject"`
	Permission string `json:"permission"`
	Resource   string `json:"resource"`
}

// opaResponse is the JSON structure received from OPA.
type opaResponse struct {
	Result bool `json:"result"`
}

// Authorize sends an authorization request to the OPA endpoint and returns
// the decision. Returns (false, error) if the OPA call fails.
func (p *OPAPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	// Create a context with timeout for the OPA request.
	opaCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Build the request body.
	req := opaRequest{
		Subject:    subject,
		Permission: string(permission),
		Resource:   resource,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return false, core.Errorf(core.ErrInvalidInput, "auth/opa: failed to marshal request: %w", err)
	}

	// Create and execute the HTTP request.
	httpReq, err := http.NewRequestWithContext(opaCtx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return false, core.Errorf(core.ErrInvalidInput, "auth/opa: failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return false, core.Errorf(core.ErrProviderDown, "auth/opa: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, core.Errorf(core.ErrProviderDown, "auth/opa: OPA returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, core.Errorf(core.ErrProviderDown, "auth/opa: failed to read response body: %w", err)
	}

	var opaResp opaResponse
	if err := json.Unmarshal(respBody, &opaResp); err != nil {
		return false, core.Errorf(core.ErrProviderDown, "auth/opa: failed to unmarshal response: %w", err)
	}

	return opaResp.Result, nil
}

// Ensure OPAPolicy implements Policy at compile time.
var _ Policy = (*OPAPolicy)(nil)

func init() {
	Register("opa", func(cfg Config) (Policy, error) {
		endpoint, ok := cfg.Extra["endpoint"].(string)
		if !ok {
			return nil, core.Errorf(core.ErrInvalidInput, "auth: OPA policy requires 'endpoint' in config")
		}
		if endpoint == "" {
			return nil, core.Errorf(core.ErrInvalidInput, "auth: OPA endpoint must not be empty")
		}

		var opts []OPAOption
		if timeout, ok := cfg.Extra["timeout"].(float64); ok && timeout > 0 {
			opts = append(opts, WithTimeout(time.Duration(timeout)*time.Second))
		}

		return NewOPAPolicy(cfg.Extra["name"].(string), endpoint, opts...), nil
	})
}
