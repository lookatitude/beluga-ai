// Package iface defines the package-specific interfaces for the Noise Cancellation package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// NoiseCancellation is an alias to the shared NoiseCancellation interface.
// This allows package-specific extensions if needed in the future.
type NoiseCancellation = voiceiface.NoiseCancellation
