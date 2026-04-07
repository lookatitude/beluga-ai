package cost

import (
	"context"
	"time"
)

// Usage represents a single LLM call's token consumption and associated cost.
type Usage struct {
	// InputTokens is the number of tokens in the prompt/input.
	InputTokens int

	// OutputTokens is the number of tokens in the completion/output.
	OutputTokens int

	// CachedTokens is the number of tokens served from the provider's cache.
	CachedTokens int

	// TotalTokens is the sum of input and output tokens. Callers should set
	// this explicitly; it is not computed automatically.
	TotalTokens int

	// Cost is the monetary cost in USD for this call.
	Cost float64

	// Model is the model identifier (e.g. "gpt-4o", "claude-3-5-sonnet").
	Model string

	// Provider is the provider name (e.g. "openai", "anthropic").
	Provider string

	// TenantID scopes this usage record to a specific tenant. Empty means
	// the default/global tenant.
	TenantID string

	// Timestamp is when the LLM call occurred.
	Timestamp time.Time
}

// Filter constrains which usage records are included in a Query result.
// Zero-value fields are ignored (i.e. they match any value).
type Filter struct {
	// TenantID restricts results to a specific tenant. Empty matches all tenants.
	TenantID string

	// Model restricts results to a specific model. Empty matches all models.
	Model string

	// Provider restricts results to a specific provider. Empty matches all providers.
	Provider string

	// Since, when non-zero, excludes records with a timestamp before this time.
	Since time.Time

	// Until, when non-zero, excludes records with a timestamp at or after this time.
	Until time.Time
}

// Summary is the aggregated result of a Query call.
type Summary struct {
	// TotalInputTokens is the sum of all InputTokens across matching records.
	TotalInputTokens int64

	// TotalOutputTokens is the sum of all OutputTokens across matching records.
	TotalOutputTokens int64

	// TotalCost is the sum of all Cost values across matching records.
	TotalCost float64

	// EntryCount is the number of usage records that matched the filter.
	EntryCount int64
}

// Tracker records LLM usage and answers aggregate queries about it.
// Implementations must be safe for concurrent use.
type Tracker interface {
	// Record stores a single usage entry. The context is used for
	// cancellation and tenant propagation.
	Record(ctx context.Context, usage Usage) error

	// Query returns an aggregated Summary for all records matching the given
	// filter. An empty filter returns a summary over all records.
	Query(ctx context.Context, filter Filter) (*Summary, error)
}
