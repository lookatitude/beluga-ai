package heuristic

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

// HeuristicConfig extends the base Turn Detection config with Heuristic-specific settings
type HeuristicConfig struct {
	*turndetection.Config

	// SentenceEndMarkers specifies characters that indicate sentence endings
	SentenceEndMarkers string `mapstructure:"sentence_end_markers" yaml:"sentence_end_markers" default:".!?"`

	// QuestionMarkers specifies phrases that indicate questions
	QuestionMarkers []string `mapstructure:"question_markers" yaml:"question_markers" default:"what,where,when,why,how,who,which"`

	// MinSilenceDuration specifies the minimum duration of silence to detect a turn end (ms)
	MinSilenceDuration time.Duration `mapstructure:"min_silence_duration" yaml:"min_silence_duration" default:"500ms"`

	// MinTurnLength specifies the minimum length of a turn in characters
	MinTurnLength int `mapstructure:"min_turn_length" yaml:"min_turn_length" default:"10"`

	// MaxTurnLength specifies the maximum length of a turn in characters
	MaxTurnLength int `mapstructure:"max_turn_length" yaml:"max_turn_length" default:"5000"`
}

// DefaultHeuristicConfig returns a default Heuristic Turn Detection configuration
func DefaultHeuristicConfig() *HeuristicConfig {
	return &HeuristicConfig{
		Config:             turndetection.DefaultConfig(),
		SentenceEndMarkers: ".!?",
		QuestionMarkers:    []string{"what", "where", "when", "why", "how", "who", "which"},
		MinSilenceDuration: 500 * time.Millisecond,
		MinTurnLength:      10,
		MaxTurnLength:      5000,
	}
}
