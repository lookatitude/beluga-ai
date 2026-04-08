package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllRisks(t *testing.T) {
	risks := AllRisks()
	assert.Len(t, risks, 10)

	// Verify ordering matches OWASP numbering.
	assert.Equal(t, RiskPromptInjection, risks[0])
	assert.Equal(t, RiskInsecureOutput, risks[1])
	assert.Equal(t, RiskToolMisuse, risks[2])
	assert.Equal(t, RiskPrivilegeEscalation, risks[3])
	assert.Equal(t, RiskMemoryPoisoning, risks[4])
	assert.Equal(t, RiskDataExfiltration, risks[5])
	assert.Equal(t, RiskSupplyChain, risks[6])
	assert.Equal(t, RiskCascadingFailure, risks[7])
	assert.Equal(t, RiskInsufficientLogging, risks[8])
	assert.Equal(t, RiskExcessiveAutonomy, risks[9])
}

func TestAgenticRisk_String(t *testing.T) {
	assert.Equal(t, "AG01_PromptInjection", string(RiskPromptInjection))
	assert.Equal(t, "AG10_ExcessiveAutonomy", string(RiskExcessiveAutonomy))
}
