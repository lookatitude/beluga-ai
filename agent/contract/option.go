package contract

import "github.com/lookatitude/beluga-ai/schema"

// Option configures a Contract during construction.
type Option func(*schema.Contract)

// WithDescription sets the contract's human-readable description.
func WithDescription(desc string) Option {
	return func(c *schema.Contract) {
		c.Description = desc
	}
}

// WithInputSchema sets the JSON Schema for the contract's expected input.
func WithInputSchema(s map[string]any) Option {
	return func(c *schema.Contract) {
		c.InputSchema = s
	}
}

// WithOutputSchema sets the JSON Schema for the contract's promised output.
func WithOutputSchema(s map[string]any) Option {
	return func(c *schema.Contract) {
		c.OutputSchema = s
	}
}

// WithStrict enables strict mode where additional properties are rejected.
func WithStrict(strict bool) Option {
	return func(c *schema.Contract) {
		c.Strict = strict
	}
}

// WithVersion sets the contract's version string.
func WithVersion(version string) Option {
	return func(c *schema.Contract) {
		c.Version = version
	}
}
