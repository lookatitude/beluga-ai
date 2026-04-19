package agentic

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/guard"
)

// AgenticPipeline combines multiple agentic guards into a single validation
// pass. Guards run sequentially; the first blocking guard stops the pipeline.
// Individual risks can be enabled or disabled via options.
type AgenticPipeline struct {
	guards []guard.Guard
}

// PipelineOption configures an AgenticPipeline.
type PipelineOption func(*AgenticPipeline)

// WithToolMisuseGuard adds a ToolMisuseGuard to the pipeline with the given
// options.
func WithToolMisuseGuard(opts ...ToolMisuseOption) PipelineOption {
	return func(p *AgenticPipeline) {
		p.guards = append(p.guards, NewToolMisuseGuard(opts...))
	}
}

// WithEscalationGuard adds a PrivilegeEscalationGuard to the pipeline with
// the given options.
func WithEscalationGuard(opts ...EscalationOption) PipelineOption {
	return func(p *AgenticPipeline) {
		p.guards = append(p.guards, NewPrivilegeEscalationGuard(opts...))
	}
}

// WithExfiltrationGuard adds a DataExfiltrationGuard to the pipeline with
// the given options.
func WithExfiltrationGuard(opts ...ExfiltrationOption) PipelineOption {
	return func(p *AgenticPipeline) {
		p.guards = append(p.guards, NewDataExfiltrationGuard(opts...))
	}
}

// WithCascadeGuard adds a CascadeGuard to the pipeline with the given
// options.
func WithCascadeGuard(opts ...CascadeOption) PipelineOption {
	return func(p *AgenticPipeline) {
		p.guards = append(p.guards, NewCascadeGuard(opts...))
	}
}

// WithGuard adds an arbitrary guard to the pipeline, allowing custom guards
// to participate alongside the built-in agentic guards.
func WithGuard(g guard.Guard) PipelineOption {
	return func(p *AgenticPipeline) {
		p.guards = append(p.guards, g)
	}
}

// NewAgenticPipeline creates an AgenticPipeline configured with the given
// options. When no options are provided, the pipeline has no guards and all
// inputs are allowed.
func NewAgenticPipeline(opts ...PipelineOption) *AgenticPipeline {
	p := &AgenticPipeline{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Validate runs all configured guards sequentially against input. It returns
// an AgenticGuardResult containing the overall outcome and per-guard
// assessments.
func (p *AgenticPipeline) Validate(ctx context.Context, input guard.GuardInput) (AgenticGuardResult, error) {
	var assessments []RiskAssessment

	for _, g := range p.guards {
		select {
		case <-ctx.Done():
			return AgenticGuardResult{}, ctx.Err()
		default:
		}

		result, err := g.Validate(ctx, input)
		if err != nil {
			return AgenticGuardResult{}, err
		}

		risk := guardNameToRisk(g.Name())
		assessment := RiskAssessment{
			Risk:      risk,
			Blocked:   !result.Allowed,
			Reason:    result.Reason,
			Severity:  riskSeverity(risk),
			GuardName: g.Name(),
		}
		assessments = append(assessments, assessment)

		if !result.Allowed {
			return AgenticGuardResult{
				GuardResult: guard.GuardResult{
					Allowed:   false,
					Reason:    result.Reason,
					GuardName: g.Name(),
				},
				Assessments: assessments,
			}, nil
		}
	}

	return AgenticGuardResult{
		GuardResult: guard.GuardResult{Allowed: true},
		Assessments: assessments,
	}, nil
}

// Guards returns the list of guards in the pipeline. This is useful for
// inspection and testing.
func (p *AgenticPipeline) Guards() []guard.Guard {
	out := make([]guard.Guard, len(p.guards))
	copy(out, p.guards)
	return out
}

// guardNameToRisk maps guard names to their corresponding OWASP agentic risk.
func guardNameToRisk(name string) AgenticRisk {
	switch name {
	case "tool_misuse_guard":
		return RiskToolMisuse
	case "privilege_escalation_guard":
		return RiskPrivilegeEscalation
	case "data_exfiltration_guard":
		return RiskDataExfiltration
	case "cascade_guard":
		return RiskCascadingFailure
	case "prompt_injection_detector":
		return RiskPromptInjection
	case "content_filter":
		return RiskInsecureOutput
	default:
		return AgenticRisk("unknown")
	}
}

// riskSeverity returns a default severity for each risk category.
func riskSeverity(risk AgenticRisk) string {
	switch risk {
	case RiskPromptInjection:
		return "critical"
	case RiskInsecureOutput:
		return "high"
	case RiskToolMisuse:
		return "high"
	case RiskPrivilegeEscalation:
		return "critical"
	case RiskMemoryPoisoning:
		return "high"
	case RiskDataExfiltration:
		return "critical"
	case RiskSupplyChain:
		return "high"
	case RiskCascadingFailure:
		return "medium"
	case RiskInsufficientLogging:
		return "medium"
	case RiskExcessiveAutonomy:
		return "high"
	default:
		return "medium"
	}
}
