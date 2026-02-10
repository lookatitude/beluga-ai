// Package smallest provides the Smallest.ai TTS provider for the Beluga AI
// voice pipeline. It uses the Smallest.ai Text-to-Speech API for low-latency
// voice synthesis.
//
// # Registration
//
// This package registers itself as "smallest" with the tts registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/smallest"
//
// # Usage
//
//	engine, err := tts.New("smallest", tts.Config{
//	    Voice: "emily",
//	    Extra: map[string]any{"api_key": "..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Smallest.ai API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default voice is "emily" and the default model is "lightning". Speed is
// configurable through [tts.Config].
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using Smallest.ai
//   - [New] — constructor accepting tts.Config
package smallest
