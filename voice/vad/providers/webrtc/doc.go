// Package webrtc provides a pure Go WebRTC-style VAD (Voice Activity Detection)
// provider for the Beluga AI voice pipeline. It uses energy and zero-crossing
// rate (ZCR) analysis on 16-bit PCM audio to detect speech, distinguishing
// voiced content from noise.
//
// # Registration
//
// This package registers itself as "webrtc" with the voice VAD registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"
//
// # Usage
//
//	vad, err := voice.NewVAD("webrtc", map[string]any{"threshold": 1500.0})
//	result, err := vad.DetectActivity(ctx, audioPCM)
//
// # Detection Algorithm
//
// Speech is detected when both conditions are met:
//
//   - RMS energy exceeds the energy threshold (filters out silence)
//   - Zero-crossing rate is below the ZCR threshold (filters out noise)
//
// This dual-criteria approach provides better discrimination between speech
// and noise compared to energy-only detection.
//
// # Configuration
//
// Configuration is passed as map[string]any:
//
//   - threshold — RMS energy threshold (default: 1000.0)
//   - zcr_threshold — Zero-crossing rate threshold (default: 0.1)
//
// # Exported Types
//
//   - [VAD] — implements voice.VAD using energy + ZCR analysis
//   - [New] — constructor accepting energy and ZCR thresholds
package webrtc
