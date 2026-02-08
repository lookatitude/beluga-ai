package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	RegisterPlanner("moa", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("moa planner requires an LLM (used as aggregator)")
		}
		var opts []MoAOption
		if layers, ok := cfg.Extra["layers"].([][]llm.ChatModel); ok {
			opts = append(opts, WithLayers(layers))
		}
		if agg, ok := cfg.Extra["aggregator"].(llm.ChatModel); ok {
			opts = append(opts, WithAggregator(agg))
		}
		return NewMoAPlanner(cfg.LLM, opts...), nil
	})
}

// MoAPlanner implements the Mixture of Agents strategy. It organizes multiple
// LLMs into layers. For each layer, all models run in parallel on the input
// (or previous layer's outputs). The final layer uses an aggregator model to
// synthesize all intermediate outputs into a single coherent response.
//
// Reference: "Mixture-of-Agents Enhances Large Language Model Capabilities"
// (Wang et al., 2024)
type MoAPlanner struct {
	// defaultLLM is used as the sole model when no layers are configured,
	// and as the default aggregator.
	defaultLLM llm.ChatModel
	// layers is a list of model layers. Each layer contains models that
	// run in parallel.
	layers [][]llm.ChatModel
	// aggregator is the model used to synthesize the final output.
	aggregator llm.ChatModel
}

// MoAOption configures a MoAPlanner.
type MoAOption func(*MoAPlanner)

// WithLayers sets the model layers. Each inner slice contains models that
// run in parallel within that layer.
func WithLayers(layers [][]llm.ChatModel) MoAOption {
	return func(p *MoAPlanner) {
		p.layers = layers
	}
}

// WithAggregator sets the aggregator model used for final synthesis.
func WithAggregator(model llm.ChatModel) MoAOption {
	return func(p *MoAPlanner) {
		p.aggregator = model
	}
}

// NewMoAPlanner creates a new Mixture of Agents planner. The defaultLLM is used
// as a single-model layer when no layers are configured, and as the aggregator
// when no explicit aggregator is set.
func NewMoAPlanner(defaultLLM llm.ChatModel, opts ...MoAOption) *MoAPlanner {
	p := &MoAPlanner{
		defaultLLM: defaultLLM,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.aggregator == nil {
		p.aggregator = defaultLLM
	}
	return p
}

// Plan executes the Mixture of Agents pipeline: for each layer, fan-out to all
// models in parallel, collect responses, and pass to the next layer. The final
// aggregator synthesizes all outputs into a single response.
func (p *MoAPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	layers := p.layers
	// If no layers configured, use default LLM as a single-model layer
	if len(layers) == 0 {
		layers = [][]llm.ChatModel{{p.defaultLLM}}
	}

	// Start with the user's messages
	currentMessages := buildMessagesFromState(state)

	// Process each layer
	var layerOutputs []string
	for layerIdx, models := range layers {
		outputs, err := p.executeLayer(ctx, state, currentMessages, models, layerOutputs)
		if err != nil {
			return nil, fmt.Errorf("moa layer %d: %w", layerIdx, err)
		}

		layerOutputs = outputs

		// Build messages for the next layer by including previous outputs as context
		currentMessages = p.buildLayerMessages(state, layerOutputs)
	}

	// Aggregate final outputs
	return p.aggregate(ctx, state, layerOutputs)
}

// Replan re-runs the MoA pipeline with observations from previous actions.
func (p *MoAPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.Plan(ctx, state)
}

// executeLayer runs all models in a layer concurrently and collects their outputs.
func (p *MoAPlanner) executeLayer(ctx context.Context, state PlannerState, messages []schema.Message, models []llm.ChatModel, previousOutputs []string) ([]string, error) {
	type result struct {
		idx    int
		output string
		err    error
	}

	results := make([]result, len(models))
	var wg sync.WaitGroup

	for i, model := range models {
		wg.Add(1)
		go func(idx int, m llm.ChatModel) {
			defer wg.Done()

			// Build messages for this model
			msgs := make([]schema.Message, 0, len(messages)+1)

			// If there are previous layer outputs, include them as context
			if len(previousOutputs) > 0 {
				var outputContext strings.Builder
				outputContext.WriteString("Previous analysis from other models:\n\n")
				for j, output := range previousOutputs {
					fmt.Fprintf(&outputContext, "Model %d response:\n%s\n\n", j+1, output)
				}
				msgs = append(msgs, schema.NewSystemMessage(outputContext.String()))
			}

			msgs = append(msgs, messages...)

			// Bind tools
			model := m
			if len(state.Tools) > 0 {
				model = model.BindTools(toolDefinitions(state.Tools))
			}

			resp, err := model.Generate(ctx, msgs)
			if err != nil {
				results[idx] = result{idx: idx, err: err}
				return
			}
			results[idx] = result{idx: idx, output: resp.Text()}
		}(i, model)
	}

	wg.Wait()

	// Collect outputs, skipping errors
	var outputs []string
	var firstErr error
	for _, r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		outputs = append(outputs, r.output)
	}

	// If all models failed, return the first error
	if len(outputs) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return outputs, nil
}

// buildLayerMessages constructs messages that include previous layer outputs
// as context for the next layer.
func (p *MoAPlanner) buildLayerMessages(state PlannerState, outputs []string) []schema.Message {
	messages := buildMessagesFromState(state)

	if len(outputs) == 0 {
		return messages
	}

	var outputContext strings.Builder
	outputContext.WriteString("Responses from the previous layer of analysis:\n\n")
	for i, output := range outputs {
		fmt.Fprintf(&outputContext, "Response %d:\n%s\n\n", i+1, output)
	}

	msgs := make([]schema.Message, 0, len(messages)+1)
	msgs = append(msgs, schema.NewSystemMessage(outputContext.String()))
	msgs = append(msgs, messages...)
	return msgs
}

// aggregate uses the aggregator model to synthesize the final layer outputs.
func (p *MoAPlanner) aggregate(ctx context.Context, state PlannerState, outputs []string) ([]Action, error) {
	messages := buildMessagesFromState(state)

	var outputContext strings.Builder
	outputContext.WriteString("You are the final aggregator. Synthesize the following responses from multiple expert models into a single, comprehensive answer.\n\n")
	for i, output := range outputs {
		fmt.Fprintf(&outputContext, "Expert %d:\n%s\n\n", i+1, output)
	}
	outputContext.WriteString("Synthesize these into a single best response. Take the strongest elements from each expert's analysis.")

	msgs := make([]schema.Message, 0, len(messages)+1)
	msgs = append(msgs, schema.NewSystemMessage(outputContext.String()))
	msgs = append(msgs, messages...)

	model := p.aggregator
	if len(state.Tools) > 0 {
		model = model.BindTools(toolDefinitions(state.Tools))
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("moa aggregate: %w", err)
	}

	return parseAIResponse(resp), nil
}
