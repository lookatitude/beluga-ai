// Package fish provides the Fish Audio TTS provider for the Beluga AI voice
// pipeline. It uses the Fish Audio Text-to-Speech API for voice synthesis.
//
// # Registration
//
// This package registers itself as "fish" with the tts registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/fish"
//
// # Usage
//
//	engine, err := tts.New("fish", tts.Config{
//	    Voice: "default",
//	    Extra: map[string]any{"api_key": "..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Fish Audio API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default voice is "default". Voice is used as the reference_id in the
// Fish Audio API.
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using Fish Audio
//   - [New] — constructor accepting tts.Config
package fish
