package replay

import (
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// DivergenceType identifies the kind of divergence between original and
// replayed events.
type DivergenceType string

const (
	// DivergenceExtra indicates an event present in replay but not in the original.
	DivergenceExtra DivergenceType = "extra"

	// DivergenceMissing indicates an event present in the original but not in replay.
	DivergenceMissing DivergenceType = "missing"

	// DivergenceTypeMismatch indicates events at the same index have different types.
	DivergenceTypeMismatch DivergenceType = "type_mismatch"

	// DivergenceAgentMismatch indicates events at the same index have different agent IDs.
	DivergenceAgentMismatch DivergenceType = "agent_mismatch"
)

// Divergence describes a single difference between original and replayed
// event sequences.
type Divergence struct {
	// Type identifies the kind of divergence.
	Type DivergenceType

	// Index is the position in the event sequence where the divergence was detected.
	Index int

	// Original is the original event at this index, if present.
	Original *schema.AgentEvent

	// Replayed is the replayed event at this index, if present.
	Replayed *schema.AgentEvent

	// Description is a human-readable explanation of the divergence.
	Description string
}

// DivergenceDetector compares original agent events against replayed events
// to identify behavioral differences.
type DivergenceDetector struct{}

// NewDivergenceDetector creates a new DivergenceDetector.
func NewDivergenceDetector() *DivergenceDetector {
	return &DivergenceDetector{}
}

// Detect compares the original and replayed event sequences and returns all
// divergences found. Events are compared positionally by index.
func (d *DivergenceDetector) Detect(original, replayed []schema.AgentEvent) []Divergence {
	var divergences []Divergence

	minLen := len(original)
	if len(replayed) < minLen {
		minLen = len(replayed)
	}

	// Compare events that exist in both sequences.
	for i := 0; i < minLen; i++ {
		orig := original[i]
		rep := replayed[i]

		if orig.Type != rep.Type {
			divergences = append(divergences, Divergence{
				Type:        DivergenceTypeMismatch,
				Index:       i,
				Original:    &orig,
				Replayed:    &rep,
				Description: "event type mismatch: original=" + orig.Type + " replayed=" + rep.Type,
			})
			continue
		}

		if orig.AgentID != rep.AgentID {
			divergences = append(divergences, Divergence{
				Type:        DivergenceAgentMismatch,
				Index:       i,
				Original:    &orig,
				Replayed:    &rep,
				Description: "agent ID mismatch: original=" + orig.AgentID + " replayed=" + rep.AgentID,
			})
		}
	}

	// Events only in the original.
	for i := minLen; i < len(original); i++ {
		orig := original[i]
		divergences = append(divergences, Divergence{
			Type:        DivergenceMissing,
			Index:       i,
			Original:    &orig,
			Description: "event missing from replay at index " + intToStr(i),
		})
	}

	// Events only in the replay.
	for i := minLen; i < len(replayed); i++ {
		rep := replayed[i]
		divergences = append(divergences, Divergence{
			Type:        DivergenceExtra,
			Index:       i,
			Replayed:    &rep,
			Description: "extra event in replay at index " + intToStr(i),
		})
	}

	return divergences
}

// intToStr converts a small integer to its string representation without
// importing strconv for this simple case.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
