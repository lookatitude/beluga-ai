// Package iface defines the package-specific interfaces for the Turn Detection package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// TurnDetector is an alias to the shared TurnDetector interface.
// This allows package-specific extensions if needed in the future.
type TurnDetector = voiceiface.TurnDetector
