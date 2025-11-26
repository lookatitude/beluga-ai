// Package iface defines the package-specific interfaces for the Session package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// VoiceSession is an alias to the shared VoiceSession interface.
// This allows package-specific extensions if needed in the future.
type VoiceSession = voiceiface.VoiceSession

// Re-export types for convenience.
type (
	SessionState = voiceiface.SessionState
	SayOptions   = voiceiface.SayOptions
	SayHandle    = voiceiface.SayHandle
)
