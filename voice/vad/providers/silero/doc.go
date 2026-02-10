//go:build cgo

// Package silero provides the Silero VAD (Voice Activity Detection) provider
// for the Beluga AI voice pipeline. It uses the Silero VAD ONNX model via
// an energy-based approximation for high-accuracy speech detection on 16-bit
// PCM audio.
//
// This package requires CGO and is only compiled when the cgo build tag is set.
//
// # Registration
//
// This package registers itself as "silero" with the voice VAD registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
//
// # Usage
//
//	vad, err := voice.NewVAD("silero", map[string]any{
//	    "threshold":  0.5,
//	    "model_path": "/path/to/silero_vad.onnx",
//	})
//	result, err := vad.DetectActivity(ctx, audioPCM)
//
// # Configuration
//
// Configuration is passed as map[string]any:
//
//   - threshold — Speech probability threshold, 0.0 to 1.0 (default: 0.5)
//   - sample_rate — Audio sample rate, 8000 or 16000 (default: 16000)
//   - model_path — Path to Silero VAD ONNX model file (optional, falls back to energy-based detection)
//
// When the ONNX model is not available, the provider uses an energy-based
// fallback calibrated to approximate Silero's behavior.
//
// # Exported Types
//
//   - [VAD] — implements voice.VAD using Silero
//   - [Config] — configuration struct
//   - [New] — constructor accepting Config
package silero
