// Package voice provides backward-compatible shims for the voice processing packages.
// All types and functions in this file are deprecated and will be removed in v2.0.
// Please update your imports to use the new package locations.
//
// Deprecated: This entire package has been split into multiple specialized packages.
// Please update your imports:
//   - pkg/stt - Speech-to-Text functionality
//   - pkg/tts - Text-to-Speech functionality
//   - pkg/vad - Voice Activity Detection
//   - pkg/s2s - Speech-to-Speech functionality
//   - pkg/audiotransport - Audio transport (WebRTC, WebSocket)
//   - pkg/noisereduction - Noise cancellation
//   - pkg/turndetection - Turn detection
//   - pkg/voicebackend - Voice backend providers
//   - pkg/voicesession - Voice session management
//   - pkg/voiceutils - Shared utilities and buffer pool
//
// This package will be removed in v2.0.
package voice

import (
	"github.com/lookatitude/beluga-ai/pkg/audiotransport"
	"github.com/lookatitude/beluga-ai/pkg/noisereduction"
	"github.com/lookatitude/beluga-ai/pkg/s2s"
	"github.com/lookatitude/beluga-ai/pkg/stt"
	"github.com/lookatitude/beluga-ai/pkg/tts"
	"github.com/lookatitude/beluga-ai/pkg/turndetection"
	"github.com/lookatitude/beluga-ai/pkg/vad"
	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
	"github.com/lookatitude/beluga-ai/pkg/voicesession"
)

// STT package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/stt instead.

// STTConfig is deprecated. Use stt.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/stt.Config.
type STTConfig = stt.Config

// GetSTTRegistry is deprecated. Use stt.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/stt.GetRegistry.
var GetSTTRegistry = stt.GetRegistry

// TTS package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/tts instead.

// TTSConfig is deprecated. Use tts.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/tts.Config.
type TTSConfig = tts.Config

// GetTTSRegistry is deprecated. Use tts.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/tts.GetRegistry.
var GetTTSRegistry = tts.GetRegistry

// VAD package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/vad instead.

// VADConfig is deprecated. Use vad.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/vad.Config.
type VADConfig = vad.Config

// GetVADRegistry is deprecated. Use vad.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/vad.GetRegistry.
var GetVADRegistry = vad.GetRegistry

// S2S package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/s2s instead.

// S2SConfig is deprecated. Use s2s.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/s2s.Config.
type S2SConfig = s2s.Config

// GetS2SRegistry is deprecated. Use s2s.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/s2s.GetRegistry.
var GetS2SRegistry = s2s.GetRegistry

// Transport package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/audiotransport instead.

// TransportConfig is deprecated. Use audiotransport.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/audiotransport.Config.
type TransportConfig = audiotransport.Config

// GetTransportRegistry is deprecated. Use audiotransport.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/audiotransport.GetRegistry.
var GetTransportRegistry = audiotransport.GetRegistry

// Noise package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/noisereduction instead.

// NoiseConfig is deprecated. Use noisereduction.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/noisereduction.Config.
type NoiseConfig = noisereduction.Config

// GetNoiseRegistry is deprecated. Use noisereduction.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/noisereduction.GetRegistry.
var GetNoiseRegistry = noisereduction.GetRegistry

// TurnDetection package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/turndetection instead.

// TurnDetectionConfig is deprecated. Use turndetection.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/turndetection.Config.
type TurnDetectionConfig = turndetection.Config

// GetTurnDetectionRegistry is deprecated. Use turndetection.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/turndetection.GetRegistry.
var GetTurnDetectionRegistry = turndetection.GetRegistry

// Backend package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/voicebackend instead.

// BackendConfig is deprecated. Use voicebackend.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/voicebackend.Config.
type BackendConfig = voicebackend.Config

// GetBackendRegistry is deprecated. Use voicebackend.GetRegistry() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/voicebackend.GetRegistry.
var GetBackendRegistry = voicebackend.GetRegistry

// Session package shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/voicesession instead.

// SessionConfig is deprecated. Use voicesession.Config instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/voicesession.Config.
type SessionConfig = voicesession.Config

// NewVoiceSession is deprecated. Use voicesession.NewVoiceSession() instead.
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/voicesession.NewVoiceSession.
var NewVoiceSession = voicesession.NewVoiceSession

// Buffer pool shims - DEPRECATED
// Use github.com/lookatitude/beluga-ai/pkg/voiceutils instead.
// Note: GetGlobalBufferPool and BufferPool are defined in buffer_pool.go for backward compatibility.
// For new code, use github.com/lookatitude/beluga-ai/pkg/voiceutils.GetGlobalBufferPool() instead.
