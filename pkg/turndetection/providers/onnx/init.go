package onnx

import (
	"github.com/lookatitude/beluga-ai/pkg/turndetection"
)

func init() {
	// Register ONNX provider with the global registry
	turndetection.GetRegistry().Register("onnx", NewONNXProvider)
}
