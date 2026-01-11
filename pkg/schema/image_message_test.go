// Package schema provides tests for ImageMessage type.
package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageMessage(t *testing.T) {
	tests := []struct {
		name       string
		imageURL   string
		textContent string
		validate   func(t *testing.T, msg Message)
	}{
		{
			name:        "basic_image_message",
			imageURL:    "https://example.com/image.jpg",
			textContent: "Look at this image",
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, RoleHuman, msg.GetType())
				assert.Contains(t, msg.GetContent(), "Look at this image")
				
				imgMsg, ok := AsImageMessage(msg)
				require.True(t, ok, "Message should be ImageMessage")
				assert.Equal(t, "https://example.com/image.jpg", imgMsg.ImageURL)
				assert.True(t, imgMsg.HasImageData())
			},
		},
		{
			name:        "image_message_no_text",
			imageURL:    "https://example.com/image.png",
			textContent: "",
			validate: func(t *testing.T, msg Message) {
				imgMsg, ok := AsImageMessage(msg)
				require.True(t, ok)
				assert.Equal(t, "https://example.com/image.png", imgMsg.ImageURL)
				assert.Contains(t, msg.GetContent(), "[Image:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewImageMessage(tt.imageURL, tt.textContent)
			tt.validate(t, msg)
		})
	}
}

func TestNewImageMessageWithData(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
	format := "jpeg"
	textContent := "JPEG image"

	msg := NewImageMessageWithData(imageData, format, textContent)
	
	imgMsg, ok := AsImageMessage(msg)
	require.True(t, ok)
	assert.Equal(t, imageData, imgMsg.ImageData)
	assert.Equal(t, format, imgMsg.ImageFormat)
	assert.Equal(t, textContent, imgMsg.GetContent())
	assert.True(t, imgMsg.HasImageData())
}

func TestNewImageMessageWithContext(t *testing.T) {
	ctx := context.Background()
	imageURL := "https://example.com/image.jpg"
	textContent := "Test image"

	msg := NewImageMessageWithContext(ctx, imageURL, textContent)
	
	imgMsg, ok := AsImageMessage(msg)
	require.True(t, ok)
	assert.Equal(t, imageURL, imgMsg.ImageURL)
	assert.Equal(t, textContent, imgMsg.GetContent())
}

func TestImageMessageAdditionalArgs(t *testing.T) {
	imageURL := "https://example.com/image.jpg"
	msg := NewImageMessage(imageURL, "Test")
	
	args := msg.AdditionalArgs()
	assert.Equal(t, imageURL, args["image_url"])
}

func TestIsImageMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      Message
		expected bool
	}{
		{
			name:     "image_message",
			msg:      NewImageMessage("https://example.com/image.jpg", "Test"),
			expected: true,
		},
		{
			name:     "regular_message",
			msg:      NewHumanMessage("Test"),
			expected: false,
		},
		{
			name:     "video_message",
			msg:      NewVideoMessage("https://example.com/video.mp4", "Test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsImageMessage(tt.msg))
		})
	}
}

func TestImageMessageMethods(t *testing.T) {
	imageURL := "https://example.com/image.jpg"
	msg := NewImageMessage(imageURL, "Test content")
	
	imgMsg, ok := AsImageMessage(msg)
	require.True(t, ok)
	
	assert.Equal(t, imageURL, imgMsg.GetImageURL())
	assert.Equal(t, "auto", imgMsg.GetImageFormat())
	assert.True(t, imgMsg.HasImageData())
}
