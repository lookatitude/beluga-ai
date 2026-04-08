package agentic

import "github.com/lookatitude/beluga-ai/guard"

// AgenticRisk identifies one of the OWASP Top 10 risks for agentic applications.
type AgenticRisk string

const (
	// RiskPromptInjection corresponds to AG01 -- prompt injection attacks.
	RiskPromptInjection AgenticRisk = "AG01_PromptInjection"

	// RiskInsecureOutput corresponds to AG02 -- insecure output handling.
	RiskInsecureOutput AgenticRisk = "AG02_InsecureOutput"

	// RiskToolMisuse corresponds to AG03 -- tool misuse or abuse.
	RiskToolMisuse AgenticRisk = "AG03_ToolMisuse"

	// RiskPrivilegeEscalation corresponds to AG04 -- privilege escalation
	// through handoff chains or delegation.
	RiskPrivilegeEscalation AgenticRisk = "AG04_PrivilegeEscalation"

	// RiskMemoryPoisoning corresponds to AG05 -- poisoning of agent memory.
	RiskMemoryPoisoning AgenticRisk = "AG05_MemoryPoisoning"

	// RiskDataExfiltration corresponds to AG06 -- exfiltration of sensitive
	// data via tool arguments or outbound payloads.
	RiskDataExfiltration AgenticRisk = "AG06_DataExfiltration"

	// RiskSupplyChain corresponds to AG07 -- supply chain attacks on tools
	// or plugins.
	RiskSupplyChain AgenticRisk = "AG07_SupplyChain"

	// RiskCascadingFailure corresponds to AG08 -- cascading failures through
	// recursive agent chains.
	RiskCascadingFailure AgenticRisk = "AG08_CascadingFailure"

	// RiskInsufficientLogging corresponds to AG09 -- insufficient logging
	// and monitoring.
	RiskInsufficientLogging AgenticRisk = "AG09_InsufficientLogging"

	// RiskExcessiveAutonomy corresponds to AG10 -- excessive agent autonomy
	// without human oversight.
	RiskExcessiveAutonomy AgenticRisk = "AG10_ExcessiveAutonomy"
)

// AllRisks returns a slice of all ten OWASP agentic risks in order.
func AllRisks() []AgenticRisk {
	return []AgenticRisk{
		RiskPromptInjection,
		RiskInsecureOutput,
		RiskToolMisuse,
		RiskPrivilegeEscalation,
		RiskMemoryPoisoning,
		RiskDataExfiltration,
		RiskSupplyChain,
		RiskCascadingFailure,
		RiskInsufficientLogging,
		RiskExcessiveAutonomy,
	}
}

// RiskAssessment records the outcome of evaluating a single agentic risk.
type RiskAssessment struct {
	// Risk is the OWASP agentic risk that was evaluated.
	Risk AgenticRisk

	// Blocked is true when the guard determined the input violates this risk.
	Blocked bool

	// Reason explains why the input was blocked or flagged. Empty when not
	// blocked.
	Reason string

	// Severity indicates the impact level: "low", "medium", "high", or
	// "critical".
	Severity string

	// GuardName is the name of the guard that produced this assessment.
	GuardName string
}

// AgenticGuardResult aggregates results from multiple agentic guards run
// through an AgenticPipeline.
type AgenticGuardResult struct {
	// GuardResult is the overall outcome following the guard.GuardResult
	// contract: Allowed is false if any guard blocked, Reason contains the
	// first blocking reason.
	guard.GuardResult

	// Assessments contains per-risk assessment details for every guard that
	// was evaluated.
	Assessments []RiskAssessment
}
