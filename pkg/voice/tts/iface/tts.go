// Package iface defines the package-specific interfaces for the TTS package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// TTSProvider is an alias to the shared TTSProvider interface.
// This allows package-specific extensions if needed in the future.
type TTSProvider = voiceiface.TTSProvider
