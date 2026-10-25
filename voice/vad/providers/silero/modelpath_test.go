//go:build cgo

package silero

import (
	"testing"

	"github.com/lookatitude/beluga-ai/v2/voice"
)

// FI-30: ModelPath must not be silently ignored.
func TestNew_ModelPathRejected(t *testing.T) {
	if _, err := New(Config{ModelPath: "/models/silero_vad.onnx"}); err == nil {
		t.Fatal("expected error for unsupported ModelPath")
	}
	if _, err := voice.NewVAD("silero", map[string]any{"model_path": "/x.onnx"}); err == nil {
		t.Fatal("expected registry error for unsupported model_path")
	}
	if _, err := New(Config{}); err != nil {
		t.Fatalf("energy VAD without ModelPath: %v", err)
	}
}
