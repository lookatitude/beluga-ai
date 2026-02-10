// Package playht provides the PlayHT TTS provider for the Beluga AI voice
// pipeline. It uses the PlayHT Text-to-Speech API for voice synthesis.
//
// # Registration
//
// This package registers itself as "playht" with the tts registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"
//
// # Usage
//
//	engine, err := tts.New("playht", tts.Config{
//	    Voice: "s3://voice-cloning-zero-shot/...",
//	    Extra: map[string]any{"api_key": "...", "user_id": "..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — PlayHT API key (required)
//   - user_id — PlayHT user ID (required)
//   - base_url — Custom API base URL (optional)
//
// Speed and output format are configurable through [tts.Config]. Default output
// format is MP3.
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using PlayHT
//   - [New] — constructor accepting tts.Config
package playht
