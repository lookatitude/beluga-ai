package redteam

import "time"

// AttackCategory identifies the type of adversarial attack.
type AttackCategory string

const (
	// CategoryPromptInjection represents attempts to override system instructions.
	CategoryPromptInjection AttackCategory = "prompt_injection"

	// CategoryJailbreak represents attempts to bypass safety guidelines.
	CategoryJailbreak AttackCategory = "jailbreak"

	// CategoryObfuscation represents attacks that encode malicious content
	// using Base64, ROT13, leetspeak, or similar transformations.
	CategoryObfuscation AttackCategory = "obfuscation"

	// CategoryToolMisuse represents attempts to abuse tool capabilities.
	CategoryToolMisuse AttackCategory = "tool_misuse"

	// CategoryDataExfiltration represents attempts to extract sensitive data.
	CategoryDataExfiltration AttackCategory = "data_exfiltration"

	// CategoryRolePlay represents attempts to trick the agent via role-playing.
	CategoryRolePlay AttackCategory = "role_play"

	// CategoryMultiTurn represents multi-turn escalation attacks.
	CategoryMultiTurn AttackCategory = "multi_turn"
)

// AllCategories returns all defined attack categories.
func AllCategories() []AttackCategory {
	return []AttackCategory{
		CategoryPromptInjection,
		CategoryJailbreak,
		CategoryObfuscation,
		CategoryToolMisuse,
		CategoryDataExfiltration,
		CategoryRolePlay,
		CategoryMultiTurn,
	}
}

// Severity indicates the impact level of a successful attack.
type Severity string

const (
	// SeverityLow indicates a minor vulnerability with limited impact.
	SeverityLow Severity = "low"

	// SeverityMedium indicates a moderate vulnerability.
	SeverityMedium Severity = "medium"

	// SeverityHigh indicates a serious vulnerability.
	SeverityHigh Severity = "high"

	// SeverityCritical indicates a critical vulnerability requiring immediate attention.
	SeverityCritical Severity = "critical"
)

// AttackResult captures the outcome of a single attack attempt.
type AttackResult struct {
	// Category is the type of attack that was attempted.
	Category AttackCategory

	// Prompt is the adversarial input sent to the target agent.
	Prompt string

	// Response is the agent's response to the adversarial prompt.
	Response string

	// Success indicates whether the attack bypassed the agent's defenses.
	Success bool

	// Severity indicates the impact level if the attack succeeded.
	Severity Severity

	// Details provides additional information about the attack outcome.
	Details string
}

// RedTeamReport is the aggregate result of a red team exercise.
type RedTeamReport struct {
	// Results contains the outcome of each individual attack.
	Results []AttackResult

	// CategoryScores maps each tested category to a defense score in [0, 1].
	// A score of 1.0 means the agent defended against all attacks in that category.
	CategoryScores map[AttackCategory]float64

	// OverallScore is the weighted average defense score across all categories.
	// A score of 1.0 means the agent defended against all attacks.
	OverallScore float64

	// Timestamp records when the red team exercise completed.
	Timestamp time.Time

	// Duration is the total wall-clock time of the exercise.
	Duration time.Duration

	// TotalAttacks is the total number of attacks executed.
	TotalAttacks int

	// SuccessfulAttacks is the number of attacks that bypassed defenses.
	SuccessfulAttacks int
}
