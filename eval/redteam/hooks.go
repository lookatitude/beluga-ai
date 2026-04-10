package redteam

import "context"

// Hooks provides optional callback functions invoked during a red team exercise.
// All fields are optional; nil hooks are skipped.
//
// Concurrency: when the runner is configured with WithParallel(n) where n > 1,
// hook functions may be invoked concurrently from multiple goroutines. Any
// shared state touched by a hook must be protected with appropriate
// synchronisation (sync.Mutex, atomic, etc.). The runner itself holds no
// locks while dispatching hooks.
type Hooks struct {
	// BeforeAttack is called before each attack prompt is sent to the target agent.
	BeforeAttack func(ctx context.Context, category AttackCategory, prompt string) error

	// AfterAttack is called after each attack has been scored.
	AfterAttack func(ctx context.Context, result AttackResult)

	// OnVulnerabilityFound is called when an attack successfully bypasses defenses.
	OnVulnerabilityFound func(ctx context.Context, result AttackResult)
}

// ComposeHooks combines multiple Hooks into a single Hooks where each field
// chains the functions in order. A nil function in any input is skipped.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		BeforeAttack:         composeBeforeAttack(hooks),
		AfterAttack:          composeAfterAttack(hooks),
		OnVulnerabilityFound: composeOnVulnerabilityFound(hooks),
	}
}

// composeBeforeAttack chains all BeforeAttack hooks. If any returns an error,
// the chain is short-circuited.
func composeBeforeAttack(hooks []Hooks) func(ctx context.Context, category AttackCategory, prompt string) error {
	var fns []func(context.Context, AttackCategory, string) error
	for _, h := range hooks {
		if h.BeforeAttack != nil {
			fns = append(fns, h.BeforeAttack)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, category AttackCategory, prompt string) error {
		for _, fn := range fns {
			if err := fn(ctx, category, prompt); err != nil {
				return err
			}
		}
		return nil
	}
}

// composeAfterAttack chains all AfterAttack hooks in order.
func composeAfterAttack(hooks []Hooks) func(ctx context.Context, result AttackResult) {
	var fns []func(context.Context, AttackResult)
	for _, h := range hooks {
		if h.AfterAttack != nil {
			fns = append(fns, h.AfterAttack)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, result AttackResult) {
		for _, fn := range fns {
			fn(ctx, result)
		}
	}
}

// composeOnVulnerabilityFound chains all OnVulnerabilityFound hooks in order.
func composeOnVulnerabilityFound(hooks []Hooks) func(ctx context.Context, result AttackResult) {
	var fns []func(context.Context, AttackResult)
	for _, h := range hooks {
		if h.OnVulnerabilityFound != nil {
			fns = append(fns, h.OnVulnerabilityFound)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, result AttackResult) {
		for _, fn := range fns {
			fn(ctx, result)
		}
	}
}
