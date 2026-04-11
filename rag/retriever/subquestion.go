package retriever

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// Decomposer breaks a complex query into sub-questions. Each sub-question
// includes a routing hint indicating which named retriever should handle it.
type Decomposer interface {
	// Decompose splits query into sub-questions, each tagged with a retriever
	// name from the available set.
	Decompose(ctx context.Context, query string, available []string) ([]SubQuestion, error)
}

// SubQuestion represents a single decomposed question with its routing target.
type SubQuestion struct {
	// Question is the decomposed sub-question text.
	Question string
	// Retriever is the name of the retriever to route this sub-question to.
	Retriever string
}

// LLMDecomposer uses an LLM to decompose queries into sub-questions.
type LLMDecomposer struct {
	llm llm.ChatModel
}

// Compile-time interface check.
var _ Decomposer = (*LLMDecomposer)(nil)

// NewLLMDecomposer creates a Decomposer backed by the given LLM.
func NewLLMDecomposer(model llm.ChatModel) *LLMDecomposer {
	return &LLMDecomposer{llm: model}
}

// Decompose uses the LLM to break the query into sub-questions, each tagged
// with a retriever name from the available set.
func (d *LLMDecomposer) Decompose(ctx context.Context, query string, available []string) ([]SubQuestion, error) {
	prompt := fmt.Sprintf(
		"Break the following complex query into simpler sub-questions.\n"+
			"For each sub-question, specify which retriever should handle it.\n\n"+
			"Available retrievers: %s\n\n"+
			"Format each line as: retriever_name|sub-question\n"+
			"Return ONLY the formatted lines, nothing else.\n\n"+
			"Query: %s",
		strings.Join(available, ", "), query,
	)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := d.llm.Generate(ctx, msgs)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "decompose query: %w", err)
	}

	return parseSubQuestions(resp.Text(), available), nil
}

// parseSubQuestions extracts SubQuestion values from LLM output lines of the
// form "retriever_name|question". Lines that don't match the format or
// reference unknown retrievers fall back to the first available retriever.
func parseSubQuestions(text string, available []string) []SubQuestion {
	lines := strings.Split(strings.TrimSpace(text), "\n")

	availSet := make(map[string]struct{}, len(available))
	for _, a := range available {
		availSet[a] = struct{}{}
	}

	fallback := ""
	if len(available) > 0 {
		fallback = available[0]
	}

	var questions []SubQuestion
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			q := strings.TrimSpace(parts[1])
			if _, ok := availSet[name]; !ok {
				name = fallback
			}
			if q != "" {
				questions = append(questions, SubQuestion{Question: q, Retriever: name})
			}
		} else {
			// No pipe — treat whole line as a question routed to fallback.
			if fallback != "" {
				questions = append(questions, SubQuestion{Question: line, Retriever: fallback})
			}
		}
	}

	return questions
}

// SubQuestionRetriever decomposes complex queries into sub-questions, routes
// each to a named retriever, retrieves results, and aggregates them into a
// deduplicated result set.
type SubQuestionRetriever struct {
	decomposer      Decomposer
	retrievers      map[string]Retriever
	maxSubQuestions int
	hooks           Hooks
}

// Compile-time interface check.
var _ Retriever = (*SubQuestionRetriever)(nil)

// SubQuestionOption configures a SubQuestionRetriever.
type SubQuestionOption func(*SubQuestionRetriever)

// WithDecomposer sets the Decomposer used to split queries. If not set,
// a default LLMDecomposer must be provided at construction time.
func WithDecomposer(d Decomposer) SubQuestionOption {
	return func(r *SubQuestionRetriever) {
		r.decomposer = d
	}
}

// WithRetrievers sets the named retrievers map for sub-question routing.
func WithRetrievers(m map[string]Retriever) SubQuestionOption {
	return func(r *SubQuestionRetriever) {
		r.retrievers = m
	}
}

// WithMaxSubQuestions limits the number of sub-questions generated.
// Defaults to 5.
func WithMaxSubQuestions(n int) SubQuestionOption {
	return func(r *SubQuestionRetriever) {
		if n > 0 {
			r.maxSubQuestions = n
		}
	}
}

// WithSubQuestionHooks sets hooks on the SubQuestionRetriever.
func WithSubQuestionHooks(h Hooks) SubQuestionOption {
	return func(r *SubQuestionRetriever) {
		r.hooks = h
	}
}

// NewSubQuestionRetriever creates a SubQuestionRetriever with the given
// options. A Decomposer and at least one named retriever must be provided
// via options.
func NewSubQuestionRetriever(opts ...SubQuestionOption) *SubQuestionRetriever {
	r := &SubQuestionRetriever{
		maxSubQuestions: 5,
		retrievers:      make(map[string]Retriever),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve decomposes the query into sub-questions, routes each to the
// appropriate named retriever, and returns aggregated, deduplicated results.
func (r *SubQuestionRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	if r.decomposer == nil {
		return nil, core.Errorf(core.ErrInvalidInput, "retriever: subquestion: decomposer not configured")
	}

	if len(r.retrievers) == 0 {
		return nil, core.Errorf(core.ErrInvalidInput, "retriever: subquestion: no retrievers configured")
	}

	available := make([]string, 0, len(r.retrievers))
	for name := range r.retrievers {
		available = append(available, name)
	}
	// Map iteration order is randomized; sort for deterministic prompts
	// and reproducible routing decisions.
	sort.Strings(available)

	subQuestions, err := r.decomposer.Decompose(ctx, query, available)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "retriever: subquestion decompose: %w", err)
	}

	// Limit sub-questions.
	if len(subQuestions) > r.maxSubQuestions {
		subQuestions = subQuestions[:r.maxSubQuestions]
	}

	seen := make(map[string]struct{})
	var results []schema.Document

	for _, sq := range subQuestions {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ret, ok := r.retrievers[sq.Retriever]
		if !ok {
			// Skip sub-questions targeting unknown retrievers.
			continue
		}

		docs, err := ret.Retrieve(ctx, sq.Question, opts...)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "retriever: subquestion retrieve %q via %q: %w", sq.Question, sq.Retriever, err)
		}

		for _, doc := range docs {
			key := doc.ID
			if key == "" {
				// Use a bounded content hash as the dedup key so that very
				// large documents do not allocate oversized map keys.
				sum := sha256.Sum256([]byte(doc.Content))
				key = hex.EncodeToString(sum[:])
			}
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				results = append(results, doc)
			}
		}
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, results, nil)
	}

	return results, nil
}
