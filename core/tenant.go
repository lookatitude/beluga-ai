package core

import "context"

// tenantKey is the context key for tenant isolation.
type tenantKey struct{}

// TenantID uniquely identifies a tenant in a multi-tenant deployment.
type TenantID string

// WithTenant returns a copy of ctx carrying the given tenant ID.
// All downstream operations will be scoped to this tenant.
func WithTenant(ctx context.Context, id TenantID) context.Context {
	return context.WithValue(ctx, tenantKey{}, id)
}

// GetTenant extracts the tenant ID from ctx. It returns an empty TenantID
// if no tenant is present.
func GetTenant(ctx context.Context) TenantID {
	id, _ := ctx.Value(tenantKey{}).(TenantID)
	return id
}
