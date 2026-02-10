// Package groq provides the Groq TTS provider for the Beluga AI voice pipeline.
// It uses the Groq TTS endpoint (OpenAI-compatible API) for voice synthesis.
//
// # Registration
//
// This package registers itself as "groq" with the tts registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/groq"
//
// # Usage
//
//	engine, err := tts.New("groq", tts.Config{
//	    Voice: "aura-asteria-en",
//	    Extra: map[string]any{"api_key": "gsk-..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Groq API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default voice is "aura-asteria-en" and the default model is "playai-tts".
// Speed and output format are configurable through [tts.Config].
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using Groq
//   - [New] — constructor accepting tts.Config
package groq
