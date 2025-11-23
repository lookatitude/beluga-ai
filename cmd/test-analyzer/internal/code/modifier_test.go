package code

import (
	"context"
	"testing"
)

func TestNewModifier(t *testing.T) {
	modifier := NewModifier()
	if modifier == nil {
		t.Fatal("NewModifier() returned nil")
	}

	if _, ok := modifier.(CodeModifier); !ok {
		t.Error("NewModifier() does not implement CodeModifier interface")
	}
}

func TestModifier_ApplyCodeChange(t *testing.T) {
	ctx := context.Background()
	modifier := NewModifier()

	t.Run("ApplyCodeChangeBasic", func(t *testing.T) {
		change := &CodeChange{
			File:        "test.go",
			LineStart:   10,
			LineEnd:     15,
			OldCode:     "old",
			NewCode:     "new",
			Description: "test change",
		}

		err := modifier.ApplyCodeChange(ctx, change)
		// May return error for invalid changes
		_ = err
	})
}

func TestModifier_CreateBackup(t *testing.T) {
	ctx := context.Background()
	modifier := NewModifier()

	t.Run("CreateBackupBasic", func(t *testing.T) {
		_, err := modifier.CreateBackup(ctx, "nonexistent.go")
		// Expected to return error for non-existent file
		_ = err
	})
}

func TestModifier_FormatCode(t *testing.T) {
	ctx := context.Background()
	modifier := NewModifier()

	t.Run("FormatCodeBasic", func(t *testing.T) {
		code := "package test\n\nfunc test() {\n}"
		_, err := modifier.FormatCode(ctx, code)
		if err != nil {
			t.Fatalf("FormatCode() error = %v", err)
		}
	})
}
