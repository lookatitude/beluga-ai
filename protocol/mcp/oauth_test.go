package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOAuthProvider_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  OAuthConfig
		wantErr string
	}{
		{
			name:    "missing client ID",
			config:  OAuthConfig{AuthURL: "https://auth.example.com", TokenURL: "https://token.example.com"},
			wantErr: "client ID is required",
		},
		{
			name:    "missing auth URL",
			config:  OAuthConfig{ClientID: "id", TokenURL: "https://token.example.com"},
			wantErr: "auth URL is required",
		},
		{
			name:    "missing token URL",
			config:  OAuthConfig{ClientID: "id", AuthURL: "https://auth.example.com"},
			wantErr: "token URL is required",
		},
		{
			name: "valid config",
			config: OAuthConfig{
				ClientID: "id",
				AuthURL:  "https://auth.example.com/authorize",
				TokenURL: "https://token.example.com/token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOAuthProvider(tt.config)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOAuthProvider_Authorize(t *testing.T) {
	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID:    "test-client",
		AuthURL:     "https://auth.example.com/authorize",
		TokenURL:    "https://token.example.com/token",
		Scopes:      []string{"read", "write"},
		RedirectURL: "https://localhost/callback",
	})
	if err != nil {
		t.Fatalf("NewOAuthProvider: %v", err)
	}

	authURL, err := provider.Authorize(context.Background(), "test-state")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if !strings.Contains(authURL, "response_type=code") {
		t.Error("missing response_type=code")
	}
	if !strings.Contains(authURL, "client_id=test-client") {
		t.Error("missing client_id")
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Error("missing state")
	}
	if !strings.Contains(authURL, "scope=read+write") {
		t.Error("missing scopes")
	}
	if !strings.Contains(authURL, "redirect_uri=") {
		t.Error("missing redirect_uri")
	}
}

func TestOAuthProvider_Authorize_WithPKCE(t *testing.T) {
	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID: "test-client",
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: "https://token.example.com/token",
		PKCE:     true,
	})
	if err != nil {
		t.Fatalf("NewOAuthProvider: %v", err)
	}

	authURL, err := provider.Authorize(context.Background(), "state")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if !strings.Contains(authURL, "code_challenge=") {
		t.Error("missing code_challenge with PKCE")
	}
	if !strings.Contains(authURL, "code_challenge_method=S256") {
		t.Error("missing code_challenge_method")
	}
}

func TestOAuthProvider_Exchange(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "test-code" {
			t.Errorf("expected code=test-code, got %q", r.Form.Get("code"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-123",
			"token_type":    "Bearer",
			"refresh_token": "refresh-456",
			"expires_in":    3600,
			"scope":         "read write",
		})
	}))
	defer ts.Close()

	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com/authorize",
		TokenURL:     ts.URL,
	})
	if err != nil {
		t.Fatalf("NewOAuthProvider: %v", err)
	}

	token, err := provider.Exchange(context.Background(), "test-code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}

	if token.AccessToken != "access-123" {
		t.Errorf("expected access token 'access-123', got %q", token.AccessToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("expected token type 'Bearer', got %q", token.TokenType)
	}
	if token.RefreshToken != "refresh-456" {
		t.Errorf("expected refresh token 'refresh-456', got %q", token.RefreshToken)
	}
	if token.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt")
	}
	if len(token.Scopes) != 2 || token.Scopes[0] != "read" {
		t.Errorf("expected scopes [read write], got %v", token.Scopes)
	}
}

func TestOAuthProvider_Exchange_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer ts.Close()

	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID: "test-client",
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.Exchange(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error for bad status code")
	}
}

func TestOAuthProvider_Exchange_EmptyAccessToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "",
			"token_type":   "Bearer",
		})
	}))
	defer ts.Close()

	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID: "test-client",
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.Exchange(context.Background(), "code")
	if err == nil {
		t.Fatal("expected error for empty access token")
	}
}

func TestOAuthProvider_Refresh(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("expected refresh_token=old-refresh, got %q", r.Form.Get("refresh_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"token_type":   "Bearer",
			"expires_in":   7200,
		})
	}))
	defer ts.Close()

	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID: "test-client",
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	token, err := provider.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if token.AccessToken != "new-access" {
		t.Errorf("expected 'new-access', got %q", token.AccessToken)
	}
}

func TestOAuthProvider_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer ts.Close()

	provider, err := NewOAuthProvider(OAuthConfig{
		ClientID: "test-client",
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = provider.Exchange(ctx, "code")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{name: "zero time not expired", expiresAt: time.Time{}, want: false},
		{name: "future not expired", expiresAt: time.Now().Add(time.Hour), want: false},
		{name: "past is expired", expiresAt: time.Now().Add(-time.Hour), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{ExpiresAt: tt.expiresAt}
			if got := token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithOAuthHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 60 * time.Second}
	provider, err := NewOAuthProvider(
		OAuthConfig{
			ClientID: "test",
			AuthURL:  "https://auth.example.com",
			TokenURL: "https://token.example.com",
		},
		WithOAuthHTTPClient(customClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	// Verify the provider was created (we can't easily inspect the http client,
	// but we verify it doesn't error).
	_ = provider
}
