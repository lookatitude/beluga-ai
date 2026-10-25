// Package nova provides the Amazon Nova Sonic S2S provider for the Beluga AI
// voice pipeline.
//
// The provider uses the Bedrock Runtime InvokeModelWithBidirectionalStream
// API through aws-sdk-go-v2 (HTTP/2 event stream, SigV4-signed). Earlier
// releases dialed an unsigned WebSocket at a non-existent
// ".../converse-stream" endpoint; that transport was removed.
//
// # Registration
//
// This package registers itself as "nova" with the s2s registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/voice/s2s/providers/nova"
//
// # Usage
//
//	engine, err := s2s.New("nova", s2s.Config{
//	    Model: "amazon.nova-sonic-v1:0",
//	    Voice: "matthew",
//	    Extra: map[string]any{"region": "us-east-1"},
//	})
//	if err != nil {
//	    return err
//	}
//	session, err := engine.Start(ctx)
//	if err != nil {
//	    return err
//	}
//	defer session.Close()
//
// # Protocol
//
// Start sends sessionStart, promptStart (24 kHz LPCM audio output, text and
// tool-use output configuration, tools), the system prompt as a SYSTEM text
// content block when Instructions is set, and opens an interactive USER
// AUDIO content block. SendAudio streams 16-bit mono LPCM (16 kHz unless
// Config.SampleRate says otherwise) on that block. SendText and
// SendToolResult send their own content blocks. Close sends contentEnd,
// promptEnd and sessionEnd.
//
// Output maps to session events: USER textOutput becomes EventTranscript,
// FINAL-stage ASSISTANT textOutput becomes EventTextOutput (SPECULATIVE
// previews are dropped), audioOutput becomes EventAudioOutput, toolUse
// becomes EventToolCall, and contentEnd with stopReason END_TURN or
// INTERRUPTED becomes EventTurnEnd. Barge-in is detected by Nova from the
// audio stream, so Interrupt is a no-op.
//
// # Configuration
//
// Configuration in Config.Extra:
//
//   - region — AWS region (optional, defaults to "us-east-1")
//   - access_key_id, secret_access_key, session_token — static AWS
//     credentials (optional; the default AWS credential chain is used
//     otherwise)
//   - base_url — Bedrock Runtime endpoint override, http(s) only (optional)
//
// The default model is "amazon.nova-sonic-v1:0" and the default voice is
// "matthew".
//
// # Exported Types
//
//   - [Engine] — implements s2s.S2S using Amazon Nova Sonic via Bedrock
//   - [New] — constructor accepting s2s.Config
package nova
