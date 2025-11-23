package spectral

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

// SpectralConfig extends the base Noise Cancellation config with Spectral Subtraction-specific settings
type SpectralConfig struct {
	*noise.Config

	// Alpha specifies the over-subtraction factor (default: 2.0)
	Alpha float64 `mapstructure:"alpha" yaml:"alpha" default:"2.0" validate:"gte=1.0,lte=5.0"`

	// Beta specifies the spectral floor factor (default: 0.1)
	Beta float64 `mapstructure:"beta" yaml:"beta" default:"0.1" validate:"gte=0.0,lte=1.0"`

	// FFTSize specifies the FFT size (default: 512)
	FFTSize int `mapstructure:"fft_size" yaml:"fft_size" default:"512" validate:"oneof=256 512 1024 2048"`

	// WindowType specifies the window function type ("hann", "hamming", "blackman")
	WindowType string `mapstructure:"window_type" yaml:"window_type" default:"hann" validate:"oneof=hann hamming blackman"`

	// Overlap specifies the overlap ratio (0.0-0.9, default: 0.5)
	Overlap float64 `mapstructure:"overlap" yaml:"overlap" default:"0.5" validate:"gte=0.0,lte=0.9"`

	// NoiseProfileUpdateRate specifies how often to update the noise profile (in frames)
	NoiseProfileUpdateRate int `mapstructure:"noise_profile_update_rate" yaml:"noise_profile_update_rate" default:"100" validate:"min=1,max=1000"`
}

// DefaultSpectralConfig returns a default Spectral Subtraction configuration
func DefaultSpectralConfig() *SpectralConfig {
	return &SpectralConfig{
		Config:                 noise.DefaultConfig(),
		Alpha:                  2.0,
		Beta:                   0.1,
		FFTSize:                512,
		WindowType:             "hann",
		Overlap:                0.5,
		NoiseProfileUpdateRate: 100,
	}
}
