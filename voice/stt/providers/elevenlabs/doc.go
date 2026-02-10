// Package elevenlabs provides the ElevenLabs Scribe STT provider for the
// Beluga AI voice pipeline. It uses the ElevenLabs Speech-to-Text API for
// batch transcription via multipart form upload.
//
// # Registration
//
// This package registers itself as "elevenlabs" with the stt registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/elevenlabs"
//
// # Usage
//
//	engine, err := stt.New("elevenlabs", stt.Config{
//	    Extra: map[string]any{"api_key": "xi-..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
// ElevenLabs Scribe does not support native streaming. TranscribeStream falls
// back to transcribing each audio chunk independently.
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — ElevenLabs API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default model is "scribe_v1".
//
// # Exported Types
//
//   - [Engine] — implements stt.STT using ElevenLabs Scribe
//   - [New] — constructor accepting stt.Config
package elevenlabs
