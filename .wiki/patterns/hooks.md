# Pattern: Hooks

Status: stub. Populate with /wiki-learn.

## Contract

Hooks are optional function fields on a Hooks struct. A nil field means skip. Hooks are composable via ComposeHooks(). Hooks observe and augment — they never replace core logic.

Example sketch:

    type Hooks struct {
        OnStart  func(context.Context, Input) error
        OnFinish func(context.Context, Output, error)
    }

    func ComposeHooks(hs ...Hooks) Hooks { /* ... */ }

## Canonical example

Populate via /wiki-learn.

## Anti-patterns

- Non-optional hook fields break the "nil equals skip" contract.
- Swallowing hook errors instead of propagating them.
- Hooks that replace core logic — that is middleware territory.

## Related

- patterns/middleware.md
