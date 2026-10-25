package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// pkceTokenServer verifies code_verifier against the challenge registered for
// each code, as a real authorization server would.
type pkceTokenServer struct {
	mu         sync.Mutex
	challenges map[string]string // code -> challenge
	forms      []url.Values
}

func (s *pkceTokenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	s.mu.Lock()
	s.forms = append(s.forms, r.PostForm)
	want := s.challenges[r.PostForm.Get("code")]
	s.mu.Unlock()
	if r.PostForm.Get("grant_type") == "authorization_code" {
		sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != want {
			http.Error(w, "invalid_grant", http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer"})
}

func challengeOf(t *testing.T, authURL string) string {
	t.Helper()
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("code_challenge")
}

func newPKCEProvider(t *testing.T, tokenURL string) *oauthProvider {
	t.Helper()
	p, err := NewOAuthProvider(OAuthConfig{
		ClientID: "c", AuthURL: "https://auth.example/authorize", TokenURL: tokenURL,
		PKCE: true, Resource: "https://mcp.example.com/mcp",
	})
	if err != nil {
		t.Fatal(err)
	}
	return p.(*oauthProvider)
}

func TestOAuth_PKCEVerifierFreshPerAuthorize(t *testing.T) {
	p := newPKCEProvider(t, "https://token.example")
	a1, err := p.Authorize(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := p.Authorize(context.Background(), "s2")
	if challengeOf(t, a1) == "" || challengeOf(t, a1) == challengeOf(t, a2) {
		t.Fatalf("PKCE challenge reused across Authorize calls: %q", challengeOf(t, a1))
	}
}

func TestOAuth_ExchangeWithStateUsesMatchingVerifier(t *testing.T) {
	ts := &pkceTokenServer{challenges: map[string]string{}}
	srv := httptest.NewServer(ts)
	defer srv.Close()
	p := newPKCEProvider(t, srv.URL)

	a1, _ := p.Authorize(context.Background(), "s1")
	a2, _ := p.Authorize(context.Background(), "s2")
	ts.challenges["code1"] = challengeOf(t, a1)
	ts.challenges["code2"] = challengeOf(t, a2)

	// Exchange (no state) is ambiguous with two pending authorizations.
	if _, err := p.Exchange(context.Background(), "code1"); err == nil || !strings.Contains(err.Error(), "ExchangeWithState") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
	var se StateExchanger = p
	if _, err := se.ExchangeWithState(context.Background(), "code2", "s2"); err != nil {
		t.Fatalf("exchange s2: %v", err)
	}
	if _, err := se.ExchangeWithState(context.Background(), "code1", "s1"); err != nil {
		t.Fatalf("exchange s1: %v", err)
	}
	// Verifiers are single-use.
	if _, err := se.ExchangeWithState(context.Background(), "code1", "s1"); err == nil {
		t.Fatal("verifier reused")
	}
}

func TestOAuth_ExchangeRequiresVerifier(t *testing.T) {
	ts := &pkceTokenServer{challenges: map[string]string{}}
	srv := httptest.NewServer(ts)
	defer srv.Close()
	p := newPKCEProvider(t, srv.URL)

	if _, err := p.Exchange(context.Background(), "code"); err == nil || !strings.Contains(err.Error(), "call Authorize first") {
		t.Fatalf("expected missing-verifier error, got %v", err)
	}
	// Single pending authorization: legacy Exchange keeps working.
	a, _ := p.Authorize(context.Background(), "only")
	ts.challenges["code"] = challengeOf(t, a)
	if _, err := p.Exchange(context.Background(), "code"); err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if v := ts.forms[len(ts.forms)-1].Get("code_verifier"); v == "" {
		t.Fatal("code_verifier not sent")
	}
}

func TestOAuth_PendingVerifierExpires(t *testing.T) {
	p := newPKCEProvider(t, "https://token.example")
	now := time.Unix(0, 0)
	p.now = func() time.Time { return now }
	_, _ = p.Authorize(context.Background(), "s")
	now = now.Add(pkcePendingTTL + time.Second)
	if _, err := p.ExchangeWithState(context.Background(), "c", "s"); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expiry, got %v", err)
	}
	_, _ = p.Authorize(context.Background(), "s2")
	now = now.Add(pkcePendingTTL + time.Second)
	if _, err := p.Exchange(context.Background(), "c"); err == nil {
		t.Fatal("expected error for expired single pending authorization")
	}
}

func TestOAuth_PendingVerifiersBounded(t *testing.T) {
	p := newPKCEProvider(t, "https://token.example")
	base := time.Unix(0, 0)
	i := 0
	p.now = func() time.Time { return base.Add(time.Duration(i) * time.Millisecond) }
	for ; i < maxPendingPKCE+10; i++ {
		_, _ = p.Authorize(context.Background(), fmt.Sprintf("s%d", i))
	}
	if len(p.pending) != maxPendingPKCE {
		t.Fatalf("pending = %d, want %d", len(p.pending), maxPendingPKCE)
	}
	if _, ok := p.pending["s0"]; ok {
		t.Fatal("oldest entry not evicted")
	}
}

func TestOAuth_ResourceParameter(t *testing.T) {
	ts := &pkceTokenServer{challenges: map[string]string{}}
	srv := httptest.NewServer(ts)
	defer srv.Close()
	p := newPKCEProvider(t, srv.URL)

	a, _ := p.Authorize(context.Background(), "s")
	u, _ := url.Parse(a)
	if u.Query().Get("resource") != "https://mcp.example.com/mcp" {
		t.Fatalf("authorize URL missing resource: %s", a)
	}
	ts.challenges["code"] = challengeOf(t, a)
	if _, err := p.ExchangeWithState(context.Background(), "code", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Refresh(context.Background(), "rt"); err != nil {
		t.Fatal(err)
	}
	for i, f := range ts.forms {
		if f.Get("resource") != "https://mcp.example.com/mcp" {
			t.Errorf("token request %d missing resource: %v", i, f)
		}
	}

	// Without Resource configured, no resource parameter is sent.
	plain, _ := NewOAuthProvider(OAuthConfig{ClientID: "c", AuthURL: "https://a/x", TokenURL: srv.URL})
	a2, _ := plain.Authorize(context.Background(), "s")
	if strings.Contains(a2, "resource=") {
		t.Fatalf("unexpected resource param: %s", a2)
	}
}
