package schema

import "testing"

func TestTextPart_PartType(t *testing.T) {
	p := TextPart{Text: "hello"}
	if got := p.PartType(); got != ContentText {
		t.Errorf("PartType() = %q, want %q", got, ContentText)
	}
}

func TestImagePart_PartType(t *testing.T) {
	p := ImagePart{MimeType: "image/png", URL: "http://example.com/img.png"}
	if got := p.PartType(); got != ContentImage {
		t.Errorf("PartType() = %q, want %q", got, ContentImage)
	}
}

func TestAudioPart_PartType(t *testing.T) {
	p := AudioPart{Format: "wav", SampleRate: 16000}
	if got := p.PartType(); got != ContentAudio {
		t.Errorf("PartType() = %q, want %q", got, ContentAudio)
	}
}

func TestVideoPart_PartType(t *testing.T) {
	p := VideoPart{MimeType: "video/mp4", URL: "http://example.com/vid.mp4"}
	if got := p.PartType(); got != ContentVideo {
		t.Errorf("PartType() = %q, want %q", got, ContentVideo)
	}
}

func TestFilePart_PartType(t *testing.T) {
	p := FilePart{Name: "report.pdf", MimeType: "application/pdf"}
	if got := p.PartType(); got != ContentFile {
		t.Errorf("PartType() = %q, want %q", got, ContentFile)
	}
}

func TestContentPart_Interface(t *testing.T) {
	// All types implement ContentPart.
	parts := []ContentPart{
		TextPart{Text: "text"},
		ImagePart{MimeType: "image/jpeg"},
		AudioPart{Format: "mp3"},
		VideoPart{MimeType: "video/mp4"},
		FilePart{Name: "file.txt"},
	}

	expected := []ContentType{ContentText, ContentImage, ContentAudio, ContentVideo, ContentFile}

	for i, p := range parts {
		if got := p.PartType(); got != expected[i] {
			t.Errorf("parts[%d].PartType() = %q, want %q", i, got, expected[i])
		}
	}
}

func TestContentType_Values(t *testing.T) {
	types := map[ContentType]string{
		ContentText:  "text",
		ContentImage: "image",
		ContentAudio: "audio",
		ContentVideo: "video",
		ContentFile:  "file",
	}
	for ct, want := range types {
		if string(ct) != want {
			t.Errorf("ContentType = %q, want %q", string(ct), want)
		}
	}
}

func TestTextPart_Fields(t *testing.T) {
	p := TextPart{Text: "Hello, world!"}
	if p.Text != "Hello, world!" {
		t.Errorf("Text = %q, want %q", p.Text, "Hello, world!")
	}
}

func TestImagePart_Fields(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47}
	p := ImagePart{
		Data:     data,
		MimeType: "image/png",
		URL:      "http://example.com/img.png",
	}
	if len(p.Data) != 4 {
		t.Errorf("Data len = %d, want 4", len(p.Data))
	}
	if p.MimeType != "image/png" {
		t.Errorf("MimeType = %q, want %q", p.MimeType, "image/png")
	}
	if p.URL != "http://example.com/img.png" {
		t.Errorf("URL = %q, want %q", p.URL, "http://example.com/img.png")
	}
}

func TestAudioPart_Fields(t *testing.T) {
	p := AudioPart{
		Data:       []byte{0x00, 0x01},
		Format:     "pcm16",
		SampleRate: 44100,
	}
	if p.Format != "pcm16" {
		t.Errorf("Format = %q, want %q", p.Format, "pcm16")
	}
	if p.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", p.SampleRate)
	}
}

func TestVideoPart_Fields(t *testing.T) {
	p := VideoPart{
		Data:     []byte{0x00},
		MimeType: "video/webm",
		URL:      "http://example.com/v.webm",
	}
	if p.MimeType != "video/webm" {
		t.Errorf("MimeType = %q, want %q", p.MimeType, "video/webm")
	}
	if p.URL != "http://example.com/v.webm" {
		t.Errorf("URL = %q, want %q", p.URL, "http://example.com/v.webm")
	}
}

func TestFilePart_Fields(t *testing.T) {
	p := FilePart{
		Data:     []byte("file content"),
		Name:     "notes.txt",
		MimeType: "text/plain",
	}
	if p.Name != "notes.txt" {
		t.Errorf("Name = %q, want %q", p.Name, "notes.txt")
	}
	if p.MimeType != "text/plain" {
		t.Errorf("MimeType = %q, want %q", p.MimeType, "text/plain")
	}
	if string(p.Data) != "file content" {
		t.Errorf("Data = %q, want %q", string(p.Data), "file content")
	}
}

func TestImagePart_EmptyFields(t *testing.T) {
	p := ImagePart{}
	if p.Data != nil {
		t.Errorf("Data = %v, want nil", p.Data)
	}
	if p.MimeType != "" {
		t.Errorf("MimeType = %q, want empty", p.MimeType)
	}
	if p.URL != "" {
		t.Errorf("URL = %q, want empty", p.URL)
	}
	if p.PartType() != ContentImage {
		t.Errorf("PartType() = %q, want %q", p.PartType(), ContentImage)
	}
}
