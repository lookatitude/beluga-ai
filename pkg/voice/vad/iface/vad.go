// Package iface defines the package-specific interfaces for the VAD package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// VADProvider is an alias to the shared VADProvider interface.
// This allows package-specific extensions if needed in the future.
type VADProvider = voiceiface.VADProvider
