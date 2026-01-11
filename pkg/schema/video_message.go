// Package schema provides multimodal message types for video.
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

// VideoMessage represents a message containing a video.
// It implements the Message interface and extends BaseMessage with video-specific data.
type VideoMessage struct {
	internal.BaseMessage
	VideoURL    string            `json:"video_url,omitempty"`    // URL to the video
	VideoData   []byte            `json:"video_data,omitempty"`   // Base64-encoded video data
	VideoFormat string            `json:"video_format,omitempty"` // Format: "mp4", "webm", "mov", etc.
	Duration    float64            `json:"duration,omitempty"`   // Duration in seconds
	Metadata    map[string]string `json:"metadata,omitempty"`     // Additional video metadata
}

// GetType returns the message type, which is RoleHuman for VideoMessage.
func (m *VideoMessage) GetType() iface.MessageType {
	return RoleHuman
}

// GetContent returns a text description or placeholder for the video.
// For multimodal messages, the actual content is in VideoURL or VideoData.
func (m *VideoMessage) GetContent() string {
	if m.BaseMessage.Content != "" {
		return m.BaseMessage.Content
	}
	if m.VideoURL != "" {
		return "[Video: " + m.VideoURL + "]"
	}
	if len(m.VideoData) > 0 {
		return "[Video: " + m.VideoFormat + " data]"
	}
	return "[Video]"
}

// ToolCalls returns an empty slice for VideoMessage.
func (m *VideoMessage) ToolCalls() []iface.ToolCall {
	return []iface.ToolCall{}
}

// AdditionalArgs returns additional arguments including video data.
func (m *VideoMessage) AdditionalArgs() map[string]any {
	args := make(map[string]any)
	if m.VideoURL != "" {
		args["video_url"] = m.VideoURL
	}
	if len(m.VideoData) > 0 {
		args["video_data"] = m.VideoData
		args["video_format"] = m.VideoFormat
	}
	if m.Duration > 0 {
		args["duration"] = m.Duration
	}
	if m.Metadata != nil {
		args["metadata"] = m.Metadata
	}
	return args
}

// HasVideoData returns true if the message contains video data.
func (m *VideoMessage) HasVideoData() bool {
	return len(m.VideoData) > 0 || m.VideoURL != ""
}

// GetVideoURL returns the video URL if present.
func (m *VideoMessage) GetVideoURL() string {
	return m.VideoURL
}

// GetVideoData returns the video data if present.
func (m *VideoMessage) GetVideoData() []byte {
	return m.VideoData
}

// GetVideoFormat returns the video format.
func (m *VideoMessage) GetVideoFormat() string {
	return m.VideoFormat
}

// GetDuration returns the video duration in seconds.
func (m *VideoMessage) GetDuration() float64 {
	return m.Duration
}

// NewVideoMessage creates a new VideoMessage with a URL.
func NewVideoMessage(videoURL string, textContent string) iface.Message {
	return &VideoMessage{
		BaseMessage: internal.BaseMessage{Content: textContent},
		VideoURL:    videoURL,
		VideoFormat: "auto", // Will be detected from URL if possible
	}
}

// NewVideoMessageWithData creates a new VideoMessage with video data.
func NewVideoMessageWithData(videoData []byte, format string, textContent string) iface.Message {
	return &VideoMessage{
		BaseMessage: internal.BaseMessage{Content: textContent},
		VideoData:   videoData,
		VideoFormat: format,
	}
}

// NewVideoMessageWithContext creates a new VideoMessage with OTEL tracing context.
func NewVideoMessageWithContext(ctx context.Context, videoURL string, textContent string) iface.Message {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/schema")
	ctx, span := tracer.Start(ctx, "schema.NewVideoMessage",
		trace.WithAttributes(
			attribute.String("message.type", "video"),
			attribute.String("video_url", videoURL),
			attribute.Int("content.length", len(textContent)),
		))
	defer span.End()

	// Record metrics
	RecordMessageCreated(ctx, RoleHuman)

	// Structured logging with OTEL context
	logWithOTELContext(ctx, slog.LevelInfo, "Creating video message",
		"message_type", "video",
		"video_url", videoURL,
		"content_length", len(textContent))

	msg := NewVideoMessage(videoURL, textContent)
	span.SetStatus(codes.Ok, "")
	return msg
}

// Ensure VideoMessage implements the Message interface.
var _ iface.Message = (*VideoMessage)(nil)
