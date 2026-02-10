// Package whisper provides the OpenAI Whisper STT provider for the Beluga AI
// voice pipeline. It uses the OpenAI Audio Transcriptions API for batch
// transcription via multipart form upload.
//
// # Registration
//
// This package registers itself as "whisper" with the stt registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"
//
// # Usage
//
//	engine, err := stt.New("whisper", stt.Config{
//	    Model: "whisper-1",
//	    Extra: map[string]any{"api_key": "sk-..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
// Whisper does not support native streaming. TranscribeStream transcribes
// each audio chunk independently as a batch request.
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — OpenAI API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default model is "whisper-1".
//
// # Exported Types
//
//   - [Engine] — implements stt.STT using OpenAI Whisper
//   - [New] — constructor accepting stt.Config
package whisper
