// Package auth provides capability-based authorization for the Beluga AI
// framework. It implements RBAC, ABAC, and composite policy patterns with a
// default-deny security model. Every authorization check is explicit — if no
// policy grants access, the request is denied.
//
// # Policy Interface
//
// The core Policy interface requires two methods:
//
//   - Name returns a unique identifier for the policy.
//   - Authorize checks whether a subject is allowed to perform a permission
//     on a resource. Returns (false, nil) for a clean deny.
//
// # RBAC
//
// RBACPolicy implements role-based access control. Subjects are assigned one
// or more roles via AssignRole, and authorization checks whether any assigned
// role contains the requested permission.
//
//	rbac := auth.NewRBACPolicy("main")
//	rbac.AddRole(auth.Role{Name: "admin", Permissions: []auth.Permission{auth.PermToolExec}})
//	rbac.AssignRole("alice", "admin")
//	allowed, err := rbac.Authorize(ctx, "alice", auth.PermToolExec, "calculator")
//
// # ABAC
//
// ABACPolicy implements attribute-based access control. Rules with conditions
// and priorities are evaluated in priority order (highest first); the first
// matching rule determines the outcome.
//
//	abac := auth.NewABACPolicy("env-check")
//	abac.AddRule(auth.Rule{
//	    Name:       "prod-deny-write",
//	    Effect:     auth.EffectDeny,
//	    Priority:   100,
//	    Conditions: []auth.Condition{isProdEnv},
//	})
//
// # Composite Policies
//
// CompositePolicy combines multiple policies using configurable modes:
//
//   - AllowIfAny allows access if any child policy allows (logical OR).
//   - AllowIfAll allows access only if all child policies allow (logical AND).
//   - DenyIfAny denies access if any child policy denies (conservative).
//
// # Built-in Permissions
//
// Standard permissions include PermToolExec, PermMemoryRead, PermMemoryWrite,
// PermAgentDelegate, and PermExternalAPI. Custom Permission values can be
// defined as needed.
//
// # Middleware and Hooks
//
// The package supports the standard Beluga middleware and hooks patterns:
//
//   - WithHooks wraps a Policy with lifecycle callbacks for OnAuthorize,
//     OnAllow, OnDeny, and OnError events.
//   - WithAudit wraps a Policy with slog-based audit logging.
//   - ApplyMiddleware composes middlewares in the standard right-to-left order.
//
// # Registry
//
// Policy factories register via the standard Beluga registry pattern with
// Register, New, and List.
package auth
