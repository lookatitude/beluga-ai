package core

import (
	"context"
	"testing"
)

func TestWithTenant_GetTenant(t *testing.T) {
	tests := []struct {
		name string
		id   TenantID
	}{
		{name: "normal_id", id: TenantID("tenant-123")},
		{name: "empty_id", id: TenantID("")},
		{name: "special_chars", id: TenantID("org/team:prod")},
		{name: "unicode", id: TenantID("テナント")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithTenant(context.Background(), tt.id)
			got := GetTenant(ctx)
			if got != tt.id {
				t.Errorf("GetTenant() = %q, want %q", got, tt.id)
			}
		})
	}
}

func TestGetTenant_NotSet(t *testing.T) {
	got := GetTenant(context.Background())
	if got != "" {
		t.Errorf("GetTenant() = %q, want empty", got)
	}
}

func TestWithTenant_Overwrite(t *testing.T) {
	ctx := WithTenant(context.Background(), TenantID("first"))
	ctx = WithTenant(ctx, TenantID("second"))

	got := GetTenant(ctx)
	if got != TenantID("second") {
		t.Errorf("GetTenant() = %q, want %q", got, "second")
	}
}

func TestWithTenant_DoesNotAffectParent(t *testing.T) {
	parent := context.Background()
	_ = WithTenant(parent, TenantID("child-tenant"))

	got := GetTenant(parent)
	if got != "" {
		t.Errorf("parent GetTenant() = %q, want empty", got)
	}
}

func TestTenantID_Type(t *testing.T) {
	var id TenantID = "test"
	s := string(id)
	if s != "test" {
		t.Errorf("string(TenantID) = %q, want %q", s, "test")
	}
}
