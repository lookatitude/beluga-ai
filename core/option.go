package core

// Option is a functional option that can be applied to any configurable type.
// The target parameter receives the configuration struct to modify.
type Option interface {
	Apply(target any)
}

// OptionFunc is an adapter that turns a plain function into an Option.
type OptionFunc func(target any)

// Apply calls the underlying function with target.
func (f OptionFunc) Apply(target any) {
	f(target)
}

// ApplyOptions applies a slice of Options to the given target.
func ApplyOptions(target any, opts ...Option) {
	for _, o := range opts {
		o.Apply(target)
	}
}
