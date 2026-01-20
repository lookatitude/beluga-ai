package livekit

import (
	"testing"
)

// FuzzParseFormatString fuzz tests the parseFormatString function.
func FuzzParseFormatString(f *testing.F) {
	// Add some seed inputs
	f.Add("pcm")
	f.Add("pcm_16k_16bit_mono")
	f.Add("opus")
	f.Add("opus_48k")
	f.Add("g722")
	f.Add("invalid_format")
	f.Add("")

	f.Fuzz(func(t *testing.T, format string) {
		// This should not panic or cause undefined behavior
		result, err := parseFormatString(format)
		// We don't care about the result, just that it doesn't crash
		_ = result
		_ = err
	})
}
