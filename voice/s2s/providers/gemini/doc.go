// Package gemini provides the Gemini Live S2S provider for the Beluga AI voice
// pipeline. It uses the Google Gemini Live API via WebSocket for bidirectional
// audio streaming with support for text, audio, and tool call events.
//
// # Registration
//
// This package registers itself as "gemini_live" with the s2s registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/gemini"
//
// # Usage
//
//	engine, err := s2s.New("gemini_live", s2s.Config{
//	    Model: "gemini-2.0-flash-exp",
//	    Extra: map[string]any{"api_key": "..."},
//	})
//	session, err := engine.Start(ctx)
//	defer session.Close()
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Google AI API key (required)
//   - base_url — Custom WebSocket endpoint (optional, defaults to Gemini Live production URL)
//
// The default model is "gemini-2.0-flash-exp". Voice, instructions, and tools
// are passed through [s2s.Config] fields.
//
// # Exported Types
//
//   - [Engine] — implements s2s.S2S using Gemini Live
//   - [New] — constructor accepting s2s.Config
package gemini
