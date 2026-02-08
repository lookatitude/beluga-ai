package hitl

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogNotifier(t *testing.T) {
	logger := slog.Default()
	n := NewLogNotifier(logger)
	if n == nil {
		t.Fatal("expected non-nil notifier")
	}

	err := n.Notify(context.Background(), InteractionRequest{
		ID:         "test-123",
		Type:       TypeApproval,
		ToolName:   "delete_user",
		Confidence: 0.7,
		RiskLevel:  RiskIrreversible,
		Description: "test reason",
	})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
}

func TestLogNotifier_NilLogger(t *testing.T) {
	n := NewLogNotifier(nil)
	if n == nil {
		t.Fatal("expected non-nil notifier even with nil logger")
	}
	if n.logger == nil {
		t.Fatal("expected default logger")
	}
}

func TestWebhookNotifier(t *testing.T) {
	var received InteractionRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	n := NewWebhookNotifier(ts.URL)

	req := InteractionRequest{
		ID:       "webhook-test",
		Type:     TypeApproval,
		ToolName: "delete_user",
		Input:    map[string]any{"user_id": "456"},
	}

	err := n.Notify(context.Background(), req)
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if received.ID != "webhook-test" {
		t.Errorf("expected ID 'webhook-test', got %q", received.ID)
	}
	if received.ToolName != "delete_user" {
		t.Errorf("expected tool 'delete_user', got %q", received.ToolName)
	}
}

func TestWebhookNotifier_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	n := NewWebhookNotifier(ts.URL)
	err := n.Notify(context.Background(), InteractionRequest{ID: "test"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestWebhookNotifier_ConnectionError(t *testing.T) {
	n := NewWebhookNotifier("http://localhost:1")
	err := n.Notify(context.Background(), InteractionRequest{ID: "test"})
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestWebhookNotifierWithClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	n := NewWebhookNotifierWithClient(ts.URL, client)
	if n == nil {
		t.Fatal("expected non-nil notifier")
	}
	if n.client != client {
		t.Error("expected custom client to be set")
	}

	err := n.Notify(context.Background(), InteractionRequest{ID: "custom-client"})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
}
