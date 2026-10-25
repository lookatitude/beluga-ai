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
//	import _ "github.com/lookatitude/beluga-ai/v2/voice/s2s/providers/openai"
//
// # Usage
//
//	engine, err := s2s.New("openai_realtime", s2s.Config{
//	    Voice: "alloy",
//	    Model: "gpt-realtime-2.1",
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
// The default model is "gpt-realtime-2.1" and the default voice is
// "alloy". Instructions and tools are passed through [s2s.Config] fields.
// Audio uses 24 kHz PCM16 ("audio/pcm") with server-side VAD for turn
// detection. Sessions use the GA Realtime API (session.type "realtime", no
// OpenAI-Beta header); both GA (response.output_audio.*) and preview
// (response.audio.*) server event names are understood.
//
// # Exported Types
//
//   - [Engine] — implements s2s.S2S using OpenAI Realtime
//   - [New] — constructor accepting s2s.Config
package openai
