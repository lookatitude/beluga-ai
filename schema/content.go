package schema

// ContentType identifies the kind of content in a ContentPart.
type ContentType string

const (
	// ContentText represents a text content part.
	ContentText ContentType = "text"
	// ContentImage represents an image content part.
	ContentImage ContentType = "image"
	// ContentAudio represents an audio content part.
	ContentAudio ContentType = "audio"
	// ContentVideo represents a video content part.
	ContentVideo ContentType = "video"
	// ContentFile represents a generic file content part.
	ContentFile ContentType = "file"
	// ContentThinking represents a reasoning/thinking content part from
	// models that expose their chain-of-thought (e.g. OpenAI o-series,
	// Claude with extended thinking).
	ContentThinking ContentType = "thinking"
)

// ContentPart is the interface implemented by all multimodal content types.
// Each message contains a slice of ContentParts, enabling rich multimodal
// conversations that mix text, images, audio, video, and files.
type ContentPart interface {
	// PartType returns the ContentType identifying this part.
	PartType() ContentType
}

// TextPart holds a plain text content part.
type TextPart struct {
	// Text is the textual content.
	Text string
}

// PartType returns ContentText.
func (t TextPart) PartType() ContentType { return ContentText }

// ImagePart holds image data, either inline or via URL.
type ImagePart struct {
	// Data contains the raw image bytes. May be nil if URL is provided.
	Data []byte
	// MimeType is the MIME type of the image (e.g., "image/png", "image/jpeg").
	MimeType string
	// URL is an optional URL pointing to the image. May be empty if Data is provided.
	URL string
}

// PartType returns ContentImage.
func (i ImagePart) PartType() ContentType { return ContentImage }

// AudioPart holds audio data for speech and sound content.
type AudioPart struct {
	// Data contains the raw audio bytes.
	Data []byte
	// Format is the audio encoding format (e.g., "wav", "mp3", "pcm16").
	Format string
	// SampleRate is the audio sample rate in Hz (e.g., 16000, 44100).
	SampleRate int
}

// PartType returns ContentAudio.
func (a AudioPart) PartType() ContentType { return ContentAudio }

// VideoPart holds video data, either inline or via URL.
type VideoPart struct {
	// Data contains the raw video bytes. May be nil if URL is provided.
	Data []byte
	// MimeType is the MIME type of the video (e.g., "video/mp4").
	MimeType string
	// URL is an optional URL pointing to the video. May be empty if Data is provided.
	URL string
}

// PartType returns ContentVideo.
func (v VideoPart) PartType() ContentType { return ContentVideo }

// FilePart holds a generic file attachment.
type FilePart struct {
	// Data contains the raw file bytes.
	Data []byte
	// Name is the filename (e.g., "report.pdf").
	Name string
	// MimeType is the MIME type of the file (e.g., "application/pdf").
	MimeType string
}

// PartType returns ContentFile.
func (f FilePart) PartType() ContentType { return ContentFile }

// Compile-time interface checks.
var _ ContentPart = TextPart{}
var _ ContentPart = ImagePart{}
var _ ContentPart = AudioPart{}
var _ ContentPart = VideoPart{}
var _ ContentPart = FilePart{}
var _ ContentPart = ThinkingPart{}

// ThinkingPart holds reasoning/chain-of-thought content from models that
// expose their internal reasoning process (e.g. OpenAI o-series, Claude
// with extended thinking).
type ThinkingPart struct {
	// Text is the reasoning content produced by the model.
	Text string
	// Signature is the opaque signature required by Anthropic's API when
	// sending a thinking block back in a subsequent turn. Empty for providers
	// that do not use signed thinking blocks.
	Signature string
}

// PartType returns ContentThinking.
func (t ThinkingPart) PartType() ContentType { return ContentThinking }
