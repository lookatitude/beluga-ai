//go:build cgo

// Package silero registers the "silero" VAD (Voice Activity Detection) name
// for the Beluga AI voice pipeline.
//
// Despite the name, this provider does NOT run the Silero neural model. It is
// an RMS-energy detector on 16-bit little-endian PCM audio whose default
// threshold is calibrated to approximate Silero VAD sensitivity. Loading the
// Silero ONNX model is not implemented: setting model_path (Config.ModelPath)
// makes construction fail instead of being silently ignored.
//
// This package requires CGO and is only compiled when the cgo build tag is set.
//
// # Registration
//
// This package registers itself as "silero" with the voice VAD registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/voice/vad/providers/silero"
//
// # Usage
//
//	vad, err := voice.NewVAD("silero", map[string]any{"threshold": 0.5})
//	if err != nil {
//	    return err
//	}
//	result, err := vad.DetectActivity(ctx, audioPCM)
//
// # Configuration
//
// Configuration is passed as map[string]any:
//
//   - threshold — Speech probability threshold, 0.0 to 1.0 (default: 0.5)
//   - sample_rate — Audio sample rate, 8000 or 16000 (default: 16000)
//   - model_path — not supported; returns an error when set
//
// # Exported Types
//
//   - [VAD] — energy-based voice.ActivityDetector
//   - [Config] — configuration struct
//   - [New] — constructor accepting Config
package silero
