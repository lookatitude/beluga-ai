package spectral

import (
	"math"
)

// WindowFunction applies a window function to a signal
type WindowFunction func([]float64)

// ApplyHann applies a Hann window
func ApplyHann(signal []float64) {
	n := len(signal)
	for i := 0; i < n; i++ {
		signal[i] *= 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
}

// ApplyHamming applies a Hamming window
func ApplyHamming(signal []float64) {
	n := len(signal)
	for i := 0; i < n; i++ {
		signal[i] *= 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(n-1))
	}
}

// ApplyBlackman applies a Blackman window
func ApplyBlackman(signal []float64) {
	n := len(signal)
	for i := 0; i < n; i++ {
		a0 := 0.42
		a1 := 0.5
		a2 := 0.08
		signal[i] *= a0 - a1*math.Cos(2*math.Pi*float64(i)/float64(n-1)) + a2*math.Cos(4*math.Pi*float64(i)/float64(n-1))
	}
}

// GetWindowFunction returns the appropriate window function
func GetWindowFunction(windowType string) WindowFunction {
	switch windowType {
	case "hamming":
		return ApplyHamming
	case "blackman":
		return ApplyBlackman
	default:
		return ApplyHann
	}
}

// FFT performs a simple FFT (placeholder - in production, use a proper FFT library)
// This is a simplified version for demonstration purposes
func FFT(signal []float64) []complex128 {
	n := len(signal)
	if n == 0 {
		return []complex128{}
	}

	// Simple DFT implementation (not optimized)
	result := make([]complex128, n)
	for k := 0; k < n; k++ {
		var sum complex128
		for j := 0; j < n; j++ {
			angle := -2.0 * math.Pi * float64(k) * float64(j) / float64(n)
			sum += complex(signal[j], 0) * complex(math.Cos(angle), math.Sin(angle))
		}
		result[k] = sum
	}
	return result
}

// IFFT performs an inverse FFT (placeholder - in production, use a proper FFT library)
func IFFT(spectrum []complex128) []float64 {
	n := len(spectrum)
	if n == 0 {
		return []float64{}
	}

	// Simple IDFT implementation (not optimized)
	result := make([]float64, n)
	for j := 0; j < n; j++ {
		var sum complex128
		for k := 0; k < n; k++ {
			angle := 2.0 * math.Pi * float64(k) * float64(j) / float64(n)
			sum += spectrum[k] * complex(math.Cos(angle), math.Sin(angle))
		}
		result[j] = real(sum) / float64(n)
	}
	return result
}

// MagnitudeSpectrum computes the magnitude spectrum from complex FFT output
func MagnitudeSpectrum(fft []complex128) []float64 {
	magnitude := make([]float64, len(fft))
	for i, val := range fft {
		magnitude[i] = math.Sqrt(real(val)*real(val) + imag(val)*imag(val))
	}
	return magnitude
}

// PhaseSpectrum computes the phase spectrum from complex FFT output
func PhaseSpectrum(fft []complex128) []float64 {
	phase := make([]float64, len(fft))
	for i, val := range fft {
		phase[i] = math.Atan2(imag(val), real(val))
	}
	return phase
}

// ReconstructFromMagnitudePhase reconstructs complex FFT from magnitude and phase
func ReconstructFromMagnitudePhase(magnitude, phase []float64) []complex128 {
	result := make([]complex128, len(magnitude))
	for i := range magnitude {
		result[i] = complex(magnitude[i]*math.Cos(phase[i]), magnitude[i]*math.Sin(phase[i]))
	}
	return result
}
