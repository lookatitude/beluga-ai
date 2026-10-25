package livekit

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/voice"
	"github.com/lookatitude/beluga-ai/v2/voice/transport"
)

// FI-11: the stub must not report success.
func TestNew_NotImplemented(t *testing.T) {
	for _, cfg := range []transport.Config{{}, {URL: "wss://example", Token: "t"}} {
		tr, err := New(cfg)
		if tr != nil {
			t.Fatal("expected nil transport")
		}
		if !errors.Is(err, transport.ErrNotImplemented) || !errors.Is(err, errors.ErrUnsupported) {
			t.Fatalf("err = %v, want ErrNotImplemented", err)
		}
	}
}

func TestRegistry_NotImplemented(t *testing.T) {
	_, err := transport.New("livekit", transport.Config{URL: "wss://example", Token: "t"})
	if !errors.Is(err, transport.ErrNotImplemented) {
		t.Fatalf("err = %v, want ErrNotImplemented", err)
	}
}

func TestTransportMethods_ReportNotImplemented(t *testing.T) {
	var tr Transport
	if err := tr.Send(context.Background(), voice.Frame{}); !errors.Is(err, transport.ErrNotImplemented) {
		t.Errorf("Send err = %v", err)
	}
	if _, err := tr.AudioOut().Write([]byte("x")); !errors.Is(err, transport.ErrNotImplemented) {
		t.Errorf("AudioOut err = %v", err)
	}
	n := 0
	for _, err := range tr.Recv(context.Background()) {
		n++
		if !errors.Is(err, transport.ErrNotImplemented) {
			t.Errorf("Recv err = %v", err)
		}
	}
	if n != 1 {
		t.Errorf("Recv yielded %d items, want 1", n)
	}
	if err := tr.Close(); err != nil {
		t.Errorf("Close err = %v", err)
	}
}
