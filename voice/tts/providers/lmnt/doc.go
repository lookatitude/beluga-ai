// Package lmnt provides the LMNT TTS provider for the Beluga AI voice pipeline.
// It uses the LMNT Text-to-Speech API for low-latency voice synthesis.
//
// # Registration
//
// This package registers itself as "lmnt" with the tts registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/lmnt"
//
// # Usage
//
//	engine, err := tts.New("lmnt", tts.Config{
//	    Voice: "lily",
//	    Extra: map[string]any{"api_key": "..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello!")
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — LMNT API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default voice is "lily". Speed and output format are configurable
// through [tts.Config].
//
// # Exported Types
//
//   - [Engine] — implements tts.TTS using LMNT
//   - [New] — constructor accepting tts.Config
package lmnt
