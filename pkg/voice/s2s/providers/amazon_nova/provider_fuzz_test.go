package amazon_nova

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// FuzzParseNovaResponse fuzz tests the parseNovaResponse function.
func FuzzParseNovaResponse(f *testing.F) {
	// Add some seed inputs
	f.Add([]byte(`{"output":{"audio":"dGVzdA==","format":{"sample_rate":16000,"channels":1,"encoding":"pcm"}},"voice":{"voice_id":"test","language":"en"},"metadata":{"latency_ms":100}}`))
	f.Add([]byte(`{"output":{"audio":"","format":{"sample_rate":0,"channels":0,"encoding":""}},"voice":{"voice_id":"","language":""},"metadata":{"latency_ms":0}}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		input := &internal.AudioInput{
			Format: internal.AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
		}

		// Create a provider instance for testing
		config := &s2s.Config{}
		p := &AmazonNovaProvider{
			providerName: "amazon_nova",
			config: &AmazonNovaConfig{
				Config: config,
			},
		}

		// This should not panic
		result, err := p.parseNovaResponse(data, input)
		// We don't care about the result, just that it doesn't crash
		_ = result
		_ = err
	})
}
