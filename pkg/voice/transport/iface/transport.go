// Package iface defines the package-specific interfaces for the Transport package.
package iface

import (
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// Transport is an alias to the shared Transport interface.
// This allows package-specific extensions if needed in the future.
type Transport = voiceiface.Transport
