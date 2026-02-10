// Package s2s provides the speech-to-speech (S2S) interface and provider
// registry for the Beluga AI voice pipeline. S2S providers handle native
// audio-in/audio-out via their own transport (WebRTC, WebSocket), bypassing
// the STT → LLM → TTS cascade for lower latency.
//
// # Core Interface
//
// The [S2S] interface provides a single method to start a bidirectional audio
// session:
//
//	type S2S interface {
//	    Start(ctx context.Context, opts ...Option) (Session, error)
//	}
//
// The [Session] interface represents an active bidirectional audio connection:
//
//	type Session interface {
//	    SendAudio(ctx context.Context, audio []byte) error
//	    SendText(ctx context.Context, text string) error
//	    SendToolResult(ctx context.Context, result schema.ToolResult) error
//	    Recv() <-chan SessionEvent
//	    Interrupt(ctx context.Context) error
//	    Close() error
//	}
//
// # Session Events
//
// Events received from the session channel are typed by [SessionEventType]:
//
//   - [EventAudioOutput] — model-generated audio
//   - [EventTextOutput] — model-generated text
//   - [EventTranscript] — user speech transcript
//   - [EventToolCall] — tool invocation request
//   - [EventTurnEnd] — end of conversational turn
//   - [EventError] — error occurred
//
// # Registry Pattern
//
// Providers register via [Register] in their init() function and are created
// with [New]. Use [List] to discover available providers.
//
//	import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
//
//	engine, err := s2s.New("openai_realtime", s2s.Config{Voice: "alloy"})
//	session, err := engine.Start(ctx)
//	defer session.Close()
//
//	session.SendAudio(ctx, audioChunk)
//	for event := range session.Recv() {
//	    switch event.Type {
//	    case s2s.EventAudioOutput:
//	        playAudio(event.Audio)
//	    case s2s.EventToolCall:
//	        handleToolCall(event.ToolCall)
//	    }
//	}
//
// # Frame Processor Integration
//
// Use [AsFrameProcessor] to wrap an S2S engine as a voice.FrameProcessor for
// integration with the cascading or hybrid pipeline.
//
// # Hooks
//
// The [Hooks] struct provides callbacks for S2S-specific events: OnTurn,
// OnInterrupt, OnToolCall, and OnError. Use [ComposeHooks] to merge hooks.
//
// # Available Providers
//
//   - openai_realtime — OpenAI Realtime API (voice/s2s/providers/openai)
//   - gemini_live — Google Gemini Live API (voice/s2s/providers/gemini)
//   - nova — Amazon Nova Sonic via Bedrock (voice/s2s/providers/nova)
package s2s
