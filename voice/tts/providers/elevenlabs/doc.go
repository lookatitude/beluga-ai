// Package elevenlabs provides the ElevenLabs TTS provider for the Beluga AI
// voice pipeline. It uses the ElevenLabs Text-to-Speech API for high-quality
// voice synthesis.
//
// # Registration
//
// This package registers itself as "elevenlabs" with the tts registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
//
// # Usage
//
//	engine, err := tts.New("elevenlabs", tts.Config{
//	    Voice: "rachel",
//	    Extra: map[string]any{"api_key": "xi-..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello, world!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — ElevenLabs API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default voice is "Rachel" (21m00Tcm4TlvDq8ikWAM) and the default model
// is "eleven_monolingual_v1". Output format defaults to audio/mpeg.
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using ElevenLabs
//   - [New] — constructor accepting tts.Config
package elevenlabs
