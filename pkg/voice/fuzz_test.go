package voice

import (
	"encoding/base64"
	"testing"
)

// FuzzBase64AudioDecode fuzz tests base64 decoding of audio data.
func FuzzBase64AudioDecode(f *testing.F) {
	// Add some seed inputs
	f.Add("dGVzdA==")         // "test"
	f.Add("")                 // empty
	f.Add("invalid base64!")  // invalid
	f.Add("YWJjZGVmZ2hpams=") // longer valid base64

	f.Fuzz(func(t *testing.T, data string) {
		// This should not panic
		result, err := base64.StdEncoding.DecodeString(data)
		// We don't care about the result, just that it doesn't crash
		_ = result
		_ = err
	})
}

// FuzzAudioFormatParsing fuzz tests various audio format string parsing.
func FuzzAudioFormatParsing(f *testing.F) {
	// Add some seed inputs
	f.Add("pcm_16k_16bit_mono")
	f.Add("opus_48k")
	f.Add("g722")
	f.Add("wav_44k_stereo")
	f.Add("")
	f.Add("invalid_format_123")

	f.Fuzz(func(t *testing.T, format string) {
		// Parse format components (sample rate, channels, etc.)
		// This is a simplified version of what various parsers do
		parts := make([]string, 0)
		current := ""
		for _, r := range format {
			if r == '_' || r == '-' {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			} else {
				current += string(r)
			}
		}
		if current != "" {
			parts = append(parts, current)
		}

		// Try to extract numeric values
		for _, part := range parts {
			// This should not panic when parsing
			_ = len(part)
		}
	})
}
