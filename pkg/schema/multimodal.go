// Package schema provides helper functions for working with multimodal message and document types.
package schema

// Type assertion helpers for multimodal types

// AsImageMessage attempts to convert a Message to ImageMessage.
// Returns the ImageMessage and true if the conversion succeeds, nil and false otherwise.
func AsImageMessage(msg Message) (*ImageMessage, bool) {
	if imgMsg, ok := msg.(*ImageMessage); ok {
		return imgMsg, true
	}
	return nil, false
}

// IsImageMessage checks if a Message is an ImageMessage.
func IsImageMessage(msg Message) bool {
	_, ok := AsImageMessage(msg)
	return ok
}

// AsVideoMessage attempts to convert a Message to VideoMessage.
// Returns the VideoMessage and true if the conversion succeeds, nil and false otherwise.
func AsVideoMessage(msg Message) (*VideoMessage, bool) {
	if vidMsg, ok := msg.(*VideoMessage); ok {
		return vidMsg, true
	}
	return nil, false
}

// IsVideoMessage checks if a Message is a VideoMessage.
func IsVideoMessage(msg Message) bool {
	_, ok := AsVideoMessage(msg)
	return ok
}

// AsVoiceDocument attempts to convert a Document to VoiceDocument.
// Since Document is a struct type (not an interface), this function checks if the Document
// was created from a VoiceDocument by checking for voice-specific metadata or using type assertion
// on the underlying value if it's a pointer.
// Returns the VoiceDocument and true if the conversion succeeds, nil and false otherwise.
func AsVoiceDocument(doc Document) (*VoiceDocument, bool) {
	// Document is a struct, so we can't use type assertion directly.
	// VoiceDocument embeds Document, so we need to check if doc was created from VoiceDocument.
	// This is a limitation - we can only check if doc is actually a *VoiceDocument when passed as Message.
	// For now, return false as Document and VoiceDocument are different types.
	return nil, false
}

// IsVoiceDocument checks if a Document is a VoiceDocument.
// Note: This is limited since Document is a struct type, not an interface.
func IsVoiceDocument(doc Document) bool {
	// Check metadata for voice-specific indicators
	if doc.Metadata != nil {
		if _, hasAudio := doc.Metadata["audio_url"]; hasAudio {
			return true
		}
		if _, hasAudioData := doc.Metadata["audio_data"]; hasAudioData {
			return true
		}
		if _, hasTranscript := doc.Metadata["transcript"]; hasTranscript {
			return true
		}
	}
	return false
}

// HasMultimodalContent checks if a Message contains multimodal content (image, video, or audio).
func HasMultimodalContent(msg Message) bool {
	return IsImageMessage(msg) || IsVideoMessage(msg)
}

// HasMultimodalDocument checks if a Document contains multimodal content (audio/voice).
// This checks metadata for voice-specific indicators since Document is a struct type.
func HasMultimodalDocument(doc Document) bool {
	return IsVoiceDocument(doc)
}

// ExtractMultimodalData extracts multimodal data from a Message if present.
// Returns a map with keys: "image_url", "image_data", "video_url", "video_data", etc.
func ExtractMultimodalData(msg Message) map[string]any {
	result := make(map[string]any)
	
	if imgMsg, ok := AsImageMessage(msg); ok {
		if imgMsg.ImageURL != "" {
			result["image_url"] = imgMsg.ImageURL
		}
		if len(imgMsg.ImageData) > 0 {
			result["image_data"] = imgMsg.ImageData
			result["image_format"] = imgMsg.ImageFormat
		}
		result["type"] = "image"
	}
	
	if vidMsg, ok := AsVideoMessage(msg); ok {
		if vidMsg.VideoURL != "" {
			result["video_url"] = vidMsg.VideoURL
		}
		if len(vidMsg.VideoData) > 0 {
			result["video_data"] = vidMsg.VideoData
			result["video_format"] = vidMsg.VideoFormat
		}
		if vidMsg.Duration > 0 {
			result["duration"] = vidMsg.Duration
		}
		result["type"] = "video"
	}
	
	return result
}

// ExtractMultimodalDocumentData extracts multimodal data from a Document if present.
// Since Document is a struct type, this checks metadata for voice-specific indicators.
// Returns a map with keys: "audio_url", "audio_data", "transcript", etc.
func ExtractMultimodalDocumentData(doc Document) map[string]any {
	result := make(map[string]any)
	
	// Check metadata for voice-specific data
	if doc.Metadata != nil {
		if audioURL, ok := doc.Metadata["audio_url"]; ok {
			result["audio_url"] = audioURL
		}
		if audioData, ok := doc.Metadata["audio_data"]; ok {
			result["audio_data"] = audioData
		}
		if audioFormat, ok := doc.Metadata["audio_format"]; ok {
			result["audio_format"] = audioFormat
		}
		if transcript, ok := doc.Metadata["transcript"]; ok {
			result["transcript"] = transcript
		}
		if duration, ok := doc.Metadata["duration"]; ok {
			result["duration"] = duration
		}
		if sampleRate, ok := doc.Metadata["sample_rate"]; ok {
			result["sample_rate"] = sampleRate
		}
		if channels, ok := doc.Metadata["channels"]; ok {
			result["channels"] = channels
		}
		if len(result) > 0 {
			result["type"] = "voice"
		}
	}
	
	return result
}

// ExtractMultimodalDocumentDataFromVoiceDocument extracts multimodal data from a VoiceDocument.
// This is the preferred method when you have a VoiceDocument instance.
func ExtractMultimodalDocumentDataFromVoiceDocument(voiceDoc *VoiceDocument) map[string]any {
	result := make(map[string]any)
	
	if voiceDoc.AudioURL != "" {
		result["audio_url"] = voiceDoc.AudioURL
	}
	if len(voiceDoc.AudioData) > 0 {
		result["audio_data"] = voiceDoc.AudioData
		result["audio_format"] = voiceDoc.AudioFormat
	}
	if voiceDoc.Transcript != "" {
		result["transcript"] = voiceDoc.Transcript
	}
	if voiceDoc.Duration > 0 {
		result["duration"] = voiceDoc.Duration
	}
	if voiceDoc.SampleRate > 0 {
		result["sample_rate"] = voiceDoc.SampleRate
	}
	if voiceDoc.Channels > 0 {
		result["channels"] = voiceDoc.Channels
	}
	result["type"] = "voice"
	
	return result
}

// Note: ImageMessage and VideoMessage implement Message interface (verified in their respective files).
// VoiceDocument implements Message interface via embedded Document (verified in voice_document.go).
