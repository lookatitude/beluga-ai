// Package openai provides the OpenAI Realtime S2S provider for the Beluga AI
// voice pipeline. It uses the OpenAI Realtime API via WebSocket for
// bidirectional audio streaming with support for text, audio, tool calls,
// and server-side VAD.
//
// # Registration
//
// This package registers itself as "openai_realtime" with the s2s registry.
// Import it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
//
// # Usage
//
//	engine, err := s2s.New("openai_realtime", s2s.Config{
//	    Voice: "alloy",
//	    Model: "gpt-4o-realtime-preview",
//	    Extra: map[string]any{"api_key": "sk-..."},
//	})
//	session, err := engine.Start(ctx)
//	defer session.Close()
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — OpenAI API key (required)
//   - base_url — Custom WebSocket endpoint (optional, defaults to wss://api.openai.com/v1/realtime)
//
// The default model is "gpt-4o-realtime-preview" and the default voice is
// "alloy". Instructions and tools are passed through [s2s.Config] fields.
// Audio uses PCM16 format with server-side VAD for turn detection.
//
// # Exported Types
//
//   - [Engine] — implements s2s.S2S using OpenAI Realtime
//   - [New] — constructor accepting s2s.Config
package openai
