// Package cartesia provides the Cartesia TTS provider for the Beluga AI voice
// pipeline. It uses the Cartesia Text-to-Speech API via direct HTTP for batch
// synthesis and streaming.
//
// # Registration
//
// This package registers itself as "cartesia" with the tts registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"
//
// # Usage
//
//	engine, err := tts.New("cartesia", tts.Config{
//	    Voice: "a0e99841-438c-4a64-b679-ae501e7d6091",
//	    Extra: map[string]any{"api_key": "sk-..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello, world!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Cartesia API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default model is "sonic-2" with raw PCM output at 24000 Hz. Voice is
// specified as a Cartesia voice ID.
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using Cartesia
//   - [New] — constructor accepting tts.Config
package cartesia
