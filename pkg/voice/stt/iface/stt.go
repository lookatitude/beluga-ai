// Package iface defines the package-specific interfaces for the STT package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// STTProvider is an alias to the shared STTProvider interface.
// This allows package-specific extensions if needed in the future.
type STTProvider = voiceiface.STTProvider

// StreamingSession is an alias to the shared StreamingSession interface.
type StreamingSession = voiceiface.StreamingSession

// TranscriptResult is an alias to the shared TranscriptResult type.
type TranscriptResult = voiceiface.TranscriptResult
