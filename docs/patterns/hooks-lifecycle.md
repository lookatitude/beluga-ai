# Pattern: Lifecycle Hooks

## What it is

A `Hooks` struct with optional function fields. `nil` means "skip this hook". `ComposeHooks(h1, h2, h3)` combines multiple hook sets into one that calls each in order, stopping on the first error.

## Why we use it

Hooks intercept at **specific execution points** — "before plan", "on tool call", "after tool result" — rather than wrapping the entire call like middleware does. They're the right tool when you care about *a moment*, not *the whole operation*.

**Alternatives considered:**
- **Interface-based listener.** Forces every consumer to implement every method, even the ones they don't care about. Optional function fields avoid this.
- **Event bus.** Works but adds async complexity and breaks the sync call model. Overkill for fine-grained interception.
- **Middleware for everything.** Middleware wraps the whole call. You can fake lifecycle interception inside middleware, but you lose fidelity and add complexity.

The optional-function-field pattern is minimal, type-safe, and composes trivially.

## How it works

Canonical code from `tool/hooks.go:11-44` (see [`.wiki/patterns/hooks.md`](../../.wiki/patterns/hooks.md)):

```go
// tool/hooks.go
package tool

import "context"

type Hooks struct {
    // BeforeExecute is called before a tool executes. Returning an error aborts execution.
    BeforeExecute func(ctx context.Context, toolName string, input map[string]any) error

    // AfterExecute is called after a tool executes (success or failure).
    AfterExecute func(ctx context.Context, toolName string, result *Result, err error)

    // OnError is called when tool execution fails. Returning nil suppresses the error.
    OnError func(ctx context.Context, toolName string, err error) error
}

func ComposeHooks(hooks ...Hooks) Hooks {
    h := append([]Hooks{}, hooks...)
    return Hooks{
        BeforeExecute: /* compose in order; stop on first error */,
        AfterExecute:  /* compose in order unconditionally */,
        OnError:       /* compose in order; first non-nil return wins */,
    }
}
```

Key properties:

- Every field is a `func` type. `nil` == no-op.
- `ComposeHooks` returns a `Hooks` whose fields are all non-nil wrapper funcs — so callers don't have to nil-check even when they compose many.
- Each wrapper loops through the component hooks and short-circuits on the first error.

Inside an implementation:

```go
func (h *hookedTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
    name := h.tool.Name()

    // BeforeExecute
    if h.hooks.BeforeExecute != nil {
        if err := h.hooks.BeforeExecute(ctx, name, input); err != nil {
            return nil, err
        }
    }

    // Execute
    result, err := h.tool.Execute(ctx, input)

    // OnError
    if err != nil && h.hooks.OnError != nil {
        err = h.hooks.OnError(ctx, name, err)
    }

    // AfterExecute
    if h.hooks.AfterExecute != nil {
        h.hooks.AfterExecute(ctx, name, result, err)
    }

    return result, err
}
```

## Hooks vs middleware

| Question | Answer |
|---|---|
| Do you want to wrap the whole call with a new behaviour? | **Middleware** |
| Do you want to fire at a specific moment inside the call? | **Hooks** |
| Do you want to modify the input/output? | **Middleware** |
| Do you want to observe without changing behaviour? | **Either**, but hooks are lighter |
| Is the behaviour cross-cutting (retry, log, rate)? | **Middleware** |
| Is the behaviour domain-specific (before-plan, on-tool-call)? | **Hooks** |

## Where it's used

- `tool` — `BeforeExecute`, `AfterExecute`, `OnError`.
- `llm` — `BeforeGenerate`, `AfterGenerate`, `OnStream`, `OnToolCall`, `OnReasoning`, `OnError`.
- `memory` — `BeforeSave`, `AfterSave`, `BeforeLoad`, `AfterLoad`, `BeforeSearch`, `AfterSearch`, `BeforeClear`, `AfterClear`, `OnError`.
- `agent` — `OnStart`, `OnEnd`, `OnError`, `BeforePlan`, `AfterPlan`, `BeforeAct`, `AfterAct`, `OnToolCall`, `OnToolResult`, `OnIteration`, `OnHandoff`, `BeforeGenerate`, `AfterGenerate`.
- `runtime` — (plugin level) `BeforeTurn`, `AfterTurn`. Plugins are a runner-level generalisation of the same pattern.

## Common mistakes

- **Nil-pointer dereference.** Always check `if hooks.OnX != nil` before invoking. `ComposeHooks` does this for you; hand-rolled invocation often forgets.
- **Swallowing hook errors.** A `BeforeExecute` returning an error should abort the operation — treating it as best-effort defeats the whole point of having a hook. Propagate.
- **Modifying shared state inside a hook.** Hooks run in the caller's goroutine, but the data they touch may be shared. Use sync primitives or keep hooks side-effect-free.
- **Unbounded hook chains.** `ComposeHooks(a, b, c, d, e, …)` is O(n) per call. If you're composing dozens of hooks, consider whether a registry of handlers would be more appropriate.
- **Using hooks where middleware belongs.** If your hook has a `func doWithRetry()` inside, that's middleware masquerading as a hook.

## Example: implementing your own

A cost-accounting hook that charges a tenant for each tool invocation:

```go
// myhooks/cost.go
package myhooks

import (
    "context"
    "sync/atomic"
    "github.com/lookatitude/beluga-ai/v2/core"
    "github.com/lookatitude/beluga-ai/v2/tool"
)

type CostTracker struct {
    perCallCost  int64 // in cents
    totalByTenant sync.Map
}

func (c *CostTracker) ToolHooks() tool.Hooks {
    return tool.Hooks{
        AfterExecute: func(ctx context.Context, toolName string, result *tool.Result, err error) {
            tenant := core.GetTenant(ctx)
            if tenant == "" {
                return // no tenant, no charge
            }
            // load-or-store and atomic add
            v, _ := c.totalByTenant.LoadOrStore(tenant, new(int64))
            atomic.AddInt64(v.(*int64), c.perCallCost)
        },
    }
}

func (c *CostTracker) Total(tenant string) int64 {
    v, ok := c.totalByTenant.Load(tenant)
    if !ok {
        return 0
    }
    return atomic.LoadInt64(v.(*int64))
}
```

Usage:

```go
cost := &myhooks.CostTracker{perCallCost: 10}
hooks := tool.ComposeHooks(
    myhooks.AuditHooks(logger),
    cost.ToolHooks(),
)
wrappedTool := tool.WithHooks(myTool, hooks)
```

Only `AfterExecute` is set — `BeforeExecute` and `OnError` remain nil and are skipped. This is the "pay only for what you use" property of the pattern.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md#ring-3--hooks)
- [`patterns/middleware-chain.md`](./middleware-chain.md) — the cross-cutting alternative.
- [`.wiki/patterns/hooks.md`](../../.wiki/patterns/hooks.md) — canonical code references.
