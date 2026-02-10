package voice

// FrameType identifies the kind of data carried by a Frame.
type FrameType string

const (
	// FrameAudio carries raw audio data (PCM, opus, etc.).
	// Metadata typically includes "sample_rate", "encoding", "channels".
	FrameAudio FrameType = "audio"

	// FrameText carries a text fragment (transcript, LLM output, etc.).
	FrameText FrameType = "text"

	// FrameControl carries control signals such as start, stop, interrupt,
	// and end-of-utterance markers.
	FrameControl FrameType = "control"

	// FrameImage carries an image or video frame for multimodal pipelines.
	FrameImage FrameType = "image"
)

// Control signal constants for FrameControl frames. These are stored in
// the Frame's Metadata under the "signal" key.
const (
	SignalStart          = "start"
	SignalStop           = "stop"
	SignalInterrupt      = "interrupt"
	SignalEndOfUtterance = "end_of_utterance"
)

// Frame is the atomic unit of data flowing through a voice pipeline.
// Each frame carries typed data and optional metadata describing its contents.
type Frame struct {
	// Type identifies the kind of data in this frame.
	Type FrameType

	// Data holds the raw payload. For audio frames this is PCM/opus bytes,
	// for text frames it is UTF-8 text, for control frames it may be empty.
	Data []byte

	// Metadata holds additional properties such as sample_rate, encoding,
	// language, signal type, or any provider-specific attributes.
	Metadata map[string]any
}

// NewAudioFrame creates an audio frame with the given data and sample rate.
func NewAudioFrame(data []byte, sampleRate int) Frame {
	return Frame{
		Type: FrameAudio,
		Data: data,
		Metadata: map[string]any{
			"sample_rate": sampleRate,
		},
	}
}

// NewTextFrame creates a text frame from a string.
func NewTextFrame(text string) Frame {
	return Frame{
		Type: FrameText,
		Data: []byte(text),
	}
}

// NewControlFrame creates a control frame with the given signal.
func NewControlFrame(signal string) Frame {
	return Frame{
		Type: FrameControl,
		Metadata: map[string]any{
			"signal": signal,
		},
	}
}

// NewImageFrame creates an image frame with the given data and content type.
func NewImageFrame(data []byte, contentType string) Frame {
	return Frame{
		Type: FrameImage,
		Data: data,
		Metadata: map[string]any{
			"content_type": contentType,
		},
	}
}

// Signal returns the control signal from a control frame's metadata.
// Returns an empty string if the frame is not a control frame or has no signal.
func (f Frame) Signal() string {
	if f.Type != FrameControl || f.Metadata == nil {
		return ""
	}
	s, _ := f.Metadata["signal"].(string)
	return s
}

// Text returns the text content of a text frame as a string.
// Returns an empty string if the frame has no data.
func (f Frame) Text() string {
	return string(f.Data)
}
