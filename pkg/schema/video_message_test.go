// Package schema provides tests for VideoMessage type.
package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVideoMessage(t *testing.T) {
	tests := []struct {
		name       string
		videoURL   string
		textContent string
		validate   func(t *testing.T, msg Message)
	}{
		{
			name:        "basic_video_message",
			videoURL:    "https://example.com/video.mp4",
			textContent: "Watch this video",
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, RoleHuman, msg.GetType())
				assert.Contains(t, msg.GetContent(), "Watch this video")
				
				vidMsg, ok := AsVideoMessage(msg)
				require.True(t, ok, "Message should be VideoMessage")
				assert.Equal(t, "https://example.com/video.mp4", vidMsg.VideoURL)
				assert.True(t, vidMsg.HasVideoData())
			},
		},
		{
			name:        "video_message_no_text",
			videoURL:    "https://example.com/video.webm",
			textContent: "",
			validate: func(t *testing.T, msg Message) {
				vidMsg, ok := AsVideoMessage(msg)
				require.True(t, ok)
				assert.Equal(t, "https://example.com/video.webm", vidMsg.VideoURL)
				assert.Contains(t, msg.GetContent(), "[Video:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewVideoMessage(tt.videoURL, tt.textContent)
			tt.validate(t, msg)
		})
	}
}

func TestNewVideoMessageWithData(t *testing.T) {
	videoData := []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70} // MP4 header
	format := "mp4"
	textContent := "MP4 video"

	msg := NewVideoMessageWithData(videoData, format, textContent)
	
	vidMsg, ok := AsVideoMessage(msg)
	require.True(t, ok)
	assert.Equal(t, videoData, vidMsg.VideoData)
	assert.Equal(t, format, vidMsg.VideoFormat)
	assert.Equal(t, textContent, vidMsg.GetContent())
	assert.True(t, vidMsg.HasVideoData())
}

func TestNewVideoMessageWithContext(t *testing.T) {
	ctx := context.Background()
	videoURL := "https://example.com/video.mp4"
	textContent := "Test video"

	msg := NewVideoMessageWithContext(ctx, videoURL, textContent)
	
	vidMsg, ok := AsVideoMessage(msg)
	require.True(t, ok)
	assert.Equal(t, videoURL, vidMsg.VideoURL)
	assert.Equal(t, textContent, vidMsg.GetContent())
}

func TestVideoMessageAdditionalArgs(t *testing.T) {
	videoURL := "https://example.com/video.mp4"
	duration := 120.5
	msg := NewVideoMessage(videoURL, "Test")
	
	vidMsg, ok := AsVideoMessage(msg)
	require.True(t, ok)
	vidMsg.Duration = duration
	
	args := msg.AdditionalArgs()
	assert.Equal(t, videoURL, args["video_url"])
	assert.Equal(t, duration, args["duration"])
}

func TestIsVideoMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      Message
		expected bool
	}{
		{
			name:     "video_message",
			msg:      NewVideoMessage("https://example.com/video.mp4", "Test"),
			expected: true,
		},
		{
			name:     "regular_message",
			msg:      NewHumanMessage("Test"),
			expected: false,
		},
		{
			name:     "image_message",
			msg:      NewImageMessage("https://example.com/image.jpg", "Test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsVideoMessage(tt.msg))
		})
	}
}

func TestVideoMessageMethods(t *testing.T) {
	videoURL := "https://example.com/video.mp4"
	duration := 60.0
	msg := NewVideoMessage(videoURL, "Test content")
	
	vidMsg, ok := AsVideoMessage(msg)
	require.True(t, ok)
	vidMsg.Duration = duration
	
	assert.Equal(t, videoURL, vidMsg.GetVideoURL())
	assert.Equal(t, "auto", vidMsg.GetVideoFormat())
	assert.Equal(t, duration, vidMsg.GetDuration())
	assert.True(t, vidMsg.HasVideoData())
}
