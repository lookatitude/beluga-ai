// Package schema provides multimodal message types for images.
package schema

import (
	"context"
	"log/slog"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ImageMessage represents a message containing an image.
// It implements the Message interface and extends BaseMessage with image-specific data.
type ImageMessage struct {
	internal.BaseMessage
	ImageURL    string            `json:"image_url,omitempty"`    // URL to the image
	ImageData   []byte            `json:"image_data,omitempty"`   // Base64-encoded image data
	ImageFormat string            `json:"image_format,omitempty"` // Format: "png", "jpeg", "webp", etc.
	Metadata    map[string]string `json:"metadata,omitempty"`     // Additional image metadata
}

// GetType returns the message type, which is RoleHuman for ImageMessage.
func (m *ImageMessage) GetType() iface.MessageType {
	return RoleHuman
}

// GetContent returns a text description or placeholder for the image.
// For multimodal messages, the actual content is in ImageURL or ImageData.
func (m *ImageMessage) GetContent() string {
	if m.BaseMessage.Content != "" {
		return m.BaseMessage.Content
	}
	if m.ImageURL != "" {
		return "[Image: " + m.ImageURL + "]"
	}
	if len(m.ImageData) > 0 {
		return "[Image: " + m.ImageFormat + " data]"
	}
	return "[Image]"
}

// ToolCalls returns an empty slice for ImageMessage.
func (m *ImageMessage) ToolCalls() []iface.ToolCall {
	return []iface.ToolCall{}
}

// AdditionalArgs returns additional arguments including image data.
func (m *ImageMessage) AdditionalArgs() map[string]any {
	args := make(map[string]any)
	if m.ImageURL != "" {
		args["image_url"] = m.ImageURL
	}
	if len(m.ImageData) > 0 {
		args["image_data"] = m.ImageData
		args["image_format"] = m.ImageFormat
	}
	if m.Metadata != nil {
		args["metadata"] = m.Metadata
	}
	return args
}

// HasImageData returns true if the message contains image data.
func (m *ImageMessage) HasImageData() bool {
	return len(m.ImageData) > 0 || m.ImageURL != ""
}

// GetImageURL returns the image URL if present.
func (m *ImageMessage) GetImageURL() string {
	return m.ImageURL
}

// GetImageData returns the image data if present.
func (m *ImageMessage) GetImageData() []byte {
	return m.ImageData
}

// GetImageFormat returns the image format.
func (m *ImageMessage) GetImageFormat() string {
	return m.ImageFormat
}

// NewImageMessage creates a new ImageMessage with a URL.
func NewImageMessage(imageURL string, textContent string) iface.Message {
	return &ImageMessage{
		BaseMessage: internal.BaseMessage{Content: textContent},
		ImageURL:    imageURL,
		ImageFormat: "auto", // Will be detected from URL if possible
	}
}

// NewImageMessageWithData creates a new ImageMessage with image data.
func NewImageMessageWithData(imageData []byte, format string, textContent string) iface.Message {
	return &ImageMessage{
		BaseMessage: internal.BaseMessage{Content: textContent},
		ImageData:   imageData,
		ImageFormat: format,
	}
}

// NewImageMessageWithContext creates a new ImageMessage with OTEL tracing context.
func NewImageMessageWithContext(ctx context.Context, imageURL string, textContent string) iface.Message {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/schema")
	ctx, span := tracer.Start(ctx, "schema.NewImageMessage",
		trace.WithAttributes(
			attribute.String("message.type", "image"),
			attribute.String("image_url", imageURL),
			attribute.Int("content.length", len(textContent)),
		))
	defer span.End()

	// Record metrics
	RecordMessageCreated(ctx, RoleHuman)

	// Structured logging with OTEL context
	logWithOTELContext(ctx, slog.LevelInfo, "Creating image message",
		"message_type", "image",
		"image_url", imageURL,
		"content_length", len(textContent))

	msg := NewImageMessage(imageURL, textContent)
	span.SetStatus(codes.Ok, "")
	return msg
}

// Ensure ImageMessage implements the Message interface.
var _ iface.Message = (*ImageMessage)(nil)
