package audio

// Codec provides audio codec utilities.
type Codec struct{}

// NewCodec creates a new codec instance.
func NewCodec() *Codec {
	return &Codec{}
}

// SupportedCodecs returns a list of supported audio codecs.
func (c *Codec) SupportedCodecs() []string {
	return []string{"pcm", "opus", "mp3", "wav"}
}

// IsSupported checks if a codec is supported.
func (c *Codec) IsSupported(codec string) bool {
	supported := c.SupportedCodecs()
	for _, s := range supported {
		if s == codec {
			return true
		}
	}
	return false
}
