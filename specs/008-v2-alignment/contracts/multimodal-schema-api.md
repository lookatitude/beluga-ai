# Multimodal Schema API Contract

**Feature**: V2 Framework Alignment  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This contract defines the multimodal schema extensions for the schema package. These extensions add support for image, audio, and video data types while maintaining backward compatibility with existing text-only message types.

---

## Schema Extensions

### 1. ImageMessage

**Type**: Extends `Message` interface

**Purpose**: Represents a message containing image data.

**Structure**:
```go
type ImageMessage struct {
    // Base Message fields (embedded)
    Message
    
    // Image-specific fields
    ImageData    []byte
    ImageFormat  string  // "jpeg", "png", "webp", etc.
    ImageURL     string  // Optional: URL to image
    Width        int     // Optional: Image width in pixels
    Height       int     // Optional: Image height in pixels
    Caption      string  // Optional: Image caption/description
    Metadata     map[string]interface{} // Optional: Additional metadata
}
```

**Behavior**:
- Extends base Message interface
- Can be used anywhere Message is expected
- Maintains backward compatibility
- Supports both embedded image data and image URLs

**Validation**:
- ImageData or ImageURL must be provided
- ImageFormat must be valid if ImageData provided
- Width and Height must be positive if provided

---

### 2. VoiceDocument

**Type**: Extends `Document` interface

**Purpose**: Represents a document containing audio/voice data.

**Structure**:
```go
type VoiceDocument struct {
    // Base Document fields (embedded)
    Document
    
    // Voice-specific fields
    AudioData    []byte
    AudioFormat  string  // "wav", "mp3", "ogg", etc.
    AudioURL     string  // Optional: URL to audio
    Duration     time.Duration // Optional: Audio duration
    SampleRate   int     // Optional: Audio sample rate
    Transcript   string  // Optional: Text transcript of audio
    Language     string  // Optional: Language code (e.g., "en-US")
    Metadata     map[string]interface{} // Optional: Additional metadata
}
```

**Behavior**:
- Extends base Document interface
- Can be used anywhere Document is expected
- Maintains backward compatibility
- Supports both embedded audio data and audio URLs

**Validation**:
- AudioData or AudioURL must be provided
- AudioFormat must be valid if AudioData provided
- Duration must be positive if provided
- SampleRate must be positive if provided

---

### 3. VideoMessage

**Type**: Extends `Message` interface

**Purpose**: Represents a message containing video data.

**Structure**:
```go
type VideoMessage struct {
    // Base Message fields (embedded)
    Message
    
    // Video-specific fields
    VideoData    []byte
    VideoFormat  string  // "mp4", "webm", "mov", etc.
    VideoURL     string  // Optional: URL to video
    Width        int     // Optional: Video width in pixels
    Height       int     // Optional: Video height in pixels
    Duration     time.Duration // Optional: Video duration
    Thumbnail    []byte  // Optional: Video thumbnail image
    Caption      string  // Optional: Video caption/description
    Metadata     map[string]interface{} // Optional: Additional metadata
}
```

**Behavior**:
- Extends base Message interface
- Can be used anywhere Message is expected
- Maintains backward compatibility
- Supports both embedded video data and video URLs

**Validation**:
- VideoData or VideoURL must be provided
- VideoFormat must be valid if VideoData provided
- Width and Height must be positive if provided
- Duration must be positive if provided

---

## Type Conversion Utilities

### 1. Convert to Base Type

**Operation**: `ToMessage(msg ImageMessage) Message`

**Purpose**: Convert multimodal message to base Message type.

**Behavior**:
- Returns base Message interface
- Maintains all base Message functionality
- Multimodal-specific data preserved in message

---

### 2. Type Assertion

**Operation**: `AssertImageMessage(msg Message) (ImageMessage, bool)`

**Purpose**: Assert that a Message is an ImageMessage.

**Behavior**:
- Returns ImageMessage and true if assertion succeeds
- Returns zero value and false if assertion fails
- Safe type checking without panics

---

## Backward Compatibility

### Text-Only Workflows

All existing text-only workflows continue to work without changes:

```go
// Existing code continues to work
msg := schema.NewTextMessage("Hello, world!")
// ... use msg as before
```

### Multimodal Workflows

New multimodal workflows can be used alongside text-only:

```go
// New multimodal message
imgMsg := schema.NewImageMessage(imageData, "jpeg")
// Can be used anywhere Message is expected
```

---

## Validation Rules

1. **Required Fields**: ImageData/ImageURL, AudioData/AudioURL, or VideoData/VideoURL must be provided
2. **Format Validation**: Data formats must be valid and supported
3. **Size Limits**: Data sizes should be within reasonable limits (configurable)
4. **URL Validation**: URLs must be valid and accessible (if provided)
5. **Metadata Validation**: Metadata must be valid JSON-serializable data

---

## Error Codes

- `ErrInvalidImageFormat`: Image format is not supported
- `ErrInvalidAudioFormat`: Audio format is not supported
- `ErrInvalidVideoFormat`: Video format is not supported
- `ErrMissingImageData`: Image data or URL is required
- `ErrMissingAudioData`: Audio data or URL is required
- `ErrMissingVideoData`: Video data or URL is required
- `ErrInvalidImageDimensions`: Image dimensions are invalid
- `ErrInvalidAudioParameters`: Audio parameters are invalid
- `ErrInvalidVideoParameters`: Video parameters are invalid

---

## Example Usage

### Creating an Image Message

```go
import "github.com/lookatitude/beluga-ai/pkg/schema"

// From image data
imageData := []byte{...} // JPEG image bytes
imgMsg := schema.NewImageMessage(imageData, "jpeg")
imgMsg.Caption = "A beautiful sunset"

// From image URL
imgMsg := schema.NewImageMessageFromURL("https://example.com/image.jpg")
```

### Creating a Voice Document

```go
import "github.com/lookatitude/beluga-ai/pkg/schema"

// From audio data
audioData := []byte{...} // WAV audio bytes
voiceDoc := schema.NewVoiceDocument(audioData, "wav")
voiceDoc.Transcript = "Hello, this is a voice message"
voiceDoc.Language = "en-US"
```

### Using with Existing Code

```go
// Multimodal message can be used where Message is expected
func processMessage(msg schema.Message) {
    // Works with both text and multimodal messages
    if imgMsg, ok := schema.AssertImageMessage(msg); ok {
        // Handle image message
    } else {
        // Handle other message types
    }
}
```

---

**Status**: Contract complete, ready for implementation.
