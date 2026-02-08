package voice

import "testing"

func TestNewAudioFrame(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	f := NewAudioFrame(data, 16000)

	if f.Type != FrameAudio {
		t.Errorf("Type = %q, want %q", f.Type, FrameAudio)
	}
	if string(f.Data) != string(data) {
		t.Errorf("Data mismatch")
	}
	if sr, ok := f.Metadata["sample_rate"].(int); !ok || sr != 16000 {
		t.Errorf("sample_rate = %v, want 16000", f.Metadata["sample_rate"])
	}
}

func TestNewTextFrame(t *testing.T) {
	f := NewTextFrame("hello world")

	if f.Type != FrameText {
		t.Errorf("Type = %q, want %q", f.Type, FrameText)
	}
	if f.Text() != "hello world" {
		t.Errorf("Text() = %q, want %q", f.Text(), "hello world")
	}
}

func TestNewControlFrame(t *testing.T) {
	tests := []struct {
		signal string
	}{
		{SignalStart},
		{SignalStop},
		{SignalInterrupt},
		{SignalEndOfUtterance},
	}

	for _, tt := range tests {
		t.Run(tt.signal, func(t *testing.T) {
			f := NewControlFrame(tt.signal)
			if f.Type != FrameControl {
				t.Errorf("Type = %q, want %q", f.Type, FrameControl)
			}
			if f.Signal() != tt.signal {
				t.Errorf("Signal() = %q, want %q", f.Signal(), tt.signal)
			}
		})
	}
}

func TestNewImageFrame(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF}
	f := NewImageFrame(data, "image/jpeg")

	if f.Type != FrameImage {
		t.Errorf("Type = %q, want %q", f.Type, FrameImage)
	}
	if ct, ok := f.Metadata["content_type"].(string); !ok || ct != "image/jpeg" {
		t.Errorf("content_type = %v, want %q", f.Metadata["content_type"], "image/jpeg")
	}
}

func TestFrameSignalNonControl(t *testing.T) {
	f := NewTextFrame("test")
	if s := f.Signal(); s != "" {
		t.Errorf("Signal() on text frame = %q, want empty", s)
	}

	f2 := Frame{Type: FrameControl} // nil metadata
	if s := f2.Signal(); s != "" {
		t.Errorf("Signal() with nil metadata = %q, want empty", s)
	}
}

func TestFrameTextEmpty(t *testing.T) {
	f := Frame{}
	if f.Text() != "" {
		t.Errorf("Text() on empty frame = %q, want empty", f.Text())
	}
}
