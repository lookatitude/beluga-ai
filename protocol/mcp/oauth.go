package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// maxOAuthResponseSize is the maximum size of an OAuth token response body (1 MB).
const maxOAuthResponseSize = 1 << 20

// Token represents an OAuth 2.0 access token with optional refresh capability.
type Token struct {
	// AccessToken is the token used to authenticate requests.
	AccessToken string `json:"access_token"`

	// TokenType is the type of token (typically "Bearer").
	TokenType string `json:"token_type"`

	// RefreshToken is an optional token used to obtain new access tokens.
	RefreshToken string `json:"refresh_token,omitempty"`

	// ExpiresAt is the time at which the access token expires.
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// Scopes are the granted scopes for this token.
	Scopes []string `json:"scopes,omitempty"`
}

// IsExpired reports whether the token has expired. A token with a zero ExpiresAt
// is never considered expired.
func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(t.ExpiresAt)
}

// OAuthConfig holds the configuration for an OAuth 2.0 provider.
type OAuthConfig struct {
	// ClientID is the application's client identifier.
	ClientID string

	// ClientSecret is the application's client secret. May be empty for
	// public clients using PKCE.
	ClientSecret string

	// AuthURL is the authorization endpoint URL.
	AuthURL string

	// TokenURL is the token endpoint URL.
	TokenURL string

	// Scopes are the OAuth scopes to request.
	Scopes []string

	// PKCE enables Proof Key for Code Exchange (RFC 7636).
	PKCE bool

	// RedirectURL is the URL to redirect to after authorization.
	RedirectURL string
}

// OAuthProvider handles the OAuth 2.0 authorization flow for MCP servers.
type OAuthProvider interface {
	// Authorize returns the authorization URL that the user should visit.
	// The state parameter is used for CSRF protection and must be verified
	// when handling the callback.
	Authorize(ctx context.Context, state string) (authURL string, err error)

	// Exchange trades an authorization code for an access token.
	Exchange(ctx context.Context, code string) (*Token, error)

	// Refresh obtains a new access token using a refresh token.
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
}

// pkceParams holds PKCE challenge parameters.
type pkceParams struct {
	verifier  string
	challenge string
	method    string
}

// oauthProvider is the default implementation of OAuthProvider.
type oauthProvider struct {
	config     OAuthConfig
	httpClient *http.Client
	pkce       *pkceParams
}

// Compile-time interface check.
var _ OAuthProvider = (*oauthProvider)(nil)

// OAuthOption configures an OAuthProvider.
type OAuthOption func(*oauthProviderOptions)

type oauthProviderOptions struct {
	httpClient *http.Client
}

// WithOAuthHTTPClient sets a custom HTTP client for the OAuth provider.
func WithOAuthHTTPClient(c *http.Client) OAuthOption {
	return func(o *oauthProviderOptions) {
		o.httpClient = c
	}
}

// NewOAuthProvider creates a new OAuthProvider with the given configuration.
func NewOAuthProvider(config OAuthConfig, opts ...OAuthOption) (OAuthProvider, error) {
	if config.ClientID == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "mcp/oauth: client ID is required")
	}
	if config.AuthURL == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "mcp/oauth: auth URL is required")
	}
	if config.TokenURL == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "mcp/oauth: token URL is required")
	}

	o := &oauthProviderOptions{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(o)
	}

	p := &oauthProvider{
		config:     config,
		httpClient: o.httpClient,
	}

	if config.PKCE {
		pkce, err := generatePKCE()
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "mcp/oauth: generate PKCE: %w", err)
		}
		p.pkce = pkce
	}

	return p, nil
}

// Authorize returns the authorization URL that the user should visit.
func (p *oauthProvider) Authorize(_ context.Context, state string) (string, error) {
	u, err := url.Parse(p.config.AuthURL)
	if err != nil {
		return "", core.Errorf(core.ErrInvalidInput, "mcp/oauth: parse auth URL: %w", err)
	}

	params := u.Query()
	params.Set("response_type", "code")
	params.Set("client_id", p.config.ClientID)
	params.Set("state", state)

	if len(p.config.Scopes) > 0 {
		params.Set("scope", strings.Join(p.config.Scopes, " "))
	}

	if p.config.RedirectURL != "" {
		params.Set("redirect_uri", p.config.RedirectURL)
	}

	if p.pkce != nil {
		params.Set("code_challenge", p.pkce.challenge)
		params.Set("code_challenge_method", p.pkce.method)
	}

	u.RawQuery = params.Encode()
	return u.String(), nil
}

// Exchange trades an authorization code for an access token.
func (p *oauthProvider) Exchange(ctx context.Context, code string) (*Token, error) {
	data := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
		"client_id":  {p.config.ClientID},
	}

	if p.config.ClientSecret != "" {
		data.Set("client_secret", p.config.ClientSecret)
	}

	if p.config.RedirectURL != "" {
		data.Set("redirect_uri", p.config.RedirectURL)
	}

	if p.pkce != nil {
		data.Set("code_verifier", p.pkce.verifier)
	}

	return p.tokenRequest(ctx, data)
}

// Refresh obtains a new access token using a refresh token.
func (p *oauthProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.config.ClientID},
	}

	if p.config.ClientSecret != "" {
		data.Set("client_secret", p.config.ClientSecret)
	}

	return p.tokenRequest(ctx, data)
}

// tokenRequest performs a token endpoint request and parses the response.
func (p *oauthProvider) tokenRequest(ctx context.Context, data url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "mcp/oauth: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/oauth: token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseSize))
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/oauth: read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/oauth: token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "mcp/oauth: decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/oauth: empty access token in response")
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	if tokenResp.Scope != "" {
		token.Scopes = strings.Split(tokenResp.Scope, " ")
	}

	return token, nil
}

// generatePKCE creates a new PKCE code verifier and challenge using S256.
func generatePKCE() (*pkceParams, error) {
	// Generate 32 bytes of random data for the verifier.
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "generate verifier: %w", err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// S256: challenge = BASE64URL(SHA256(verifier))
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &pkceParams{
		verifier:  verifier,
		challenge: challenge,
		method:    "S256",
	}, nil
}
