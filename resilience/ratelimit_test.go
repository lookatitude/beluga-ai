package resilience

import (
	"context"
	"testing"
	"time"
)

func TestNewRateLimiter_Defaults(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{})
	if rl.limits.RPM != 0 {
		t.Errorf("RPM = %d, want 0 (unlimited)", rl.limits.RPM)
	}
	if rl.limits.TPM != 0 {
		t.Errorf("TPM = %d, want 0 (unlimited)", rl.limits.TPM)
	}
	if rl.limits.MaxConcurrent != 0 {
		t.Errorf("MaxConcurrent = %d, want 0 (unlimited)", rl.limits.MaxConcurrent)
	}
}

func TestNewRateLimiter_WithRPM(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{RPM: 60})
	if rl.rpmTokens != 60.0 {
		t.Errorf("rpmTokens = %f, want 60.0", rl.rpmTokens)
	}
}

func TestNewRateLimiter_WithTPM(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{TPM: 10000})
	if rl.tpmTokens != 10000.0 {
		t.Errorf("tpmTokens = %f, want 10000.0", rl.tpmTokens)
	}
}

func TestRateLimiter_Allow_Unlimited(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{})
	for i := 0; i < 100; i++ {
		if err := rl.Allow(context.Background()); err != nil {
			t.Fatalf("Allow() error = %v on iteration %d", err, i)
		}
	}
}

func TestRateLimiter_Allow_RPMLimited(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{RPM: 2})

	// First two should succeed immediately.
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		err := rl.Allow(ctx)
		cancel()
		if err != nil {
			t.Fatalf("Allow() call %d error = %v", i, err)
		}
	}

	// Third should block and timeout (no tokens available).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := rl.Allow(ctx)
	if err == nil {
		t.Error("expected timeout error when RPM exhausted")
	}
}

func TestRateLimiter_Allow_ConcurrencyLimited(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{MaxConcurrent: 1})

	// First call succeeds.
	if err := rl.Allow(context.Background()); err != nil {
		t.Fatalf("Allow() first call error = %v", err)
	}

	// Second call should block because max concurrency reached.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := rl.Allow(ctx)
	if err == nil {
		t.Error("expected timeout error when concurrency limit reached")
	}
}

func TestRateLimiter_Release(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{MaxConcurrent: 1})

	// Acquire slot.
	if err := rl.Allow(context.Background()); err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	// Release slot.
	rl.Release()

	// Should be able to acquire again.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := rl.Allow(ctx); err != nil {
		t.Fatalf("Allow() after Release() error = %v", err)
	}
}

func TestRateLimiter_Release_NeverNegative(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{MaxConcurrent: 1})

	// Release without prior Allow should not panic or go negative.
	rl.Release()

	rl.mu.Lock()
	if rl.concurrent < 0 {
		t.Errorf("concurrent = %d, should never go below 0", rl.concurrent)
	}
	rl.mu.Unlock()
}

func TestRateLimiter_Wait_NoCooldown(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{CooldownOnRetry: 0})
	err := rl.Wait(context.Background())
	if err != nil {
		t.Errorf("Wait() error = %v, want nil (no cooldown)", err)
	}
}

func TestRateLimiter_Wait_WithCooldown(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{CooldownOnRetry: 10 * time.Millisecond})
	start := time.Now()
	err := rl.Wait(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if elapsed < 5*time.Millisecond {
		t.Errorf("Wait() returned too fast: %v", elapsed)
	}
}

func TestRateLimiter_Wait_ContextCancelled(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{CooldownOnRetry: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("Wait() expected context cancellation error")
	}
}

func TestRateLimiter_ConsumeTokens_NoTPM(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{TPM: 0})
	err := rl.ConsumeTokens(context.Background(), 100)
	if err != nil {
		t.Errorf("ConsumeTokens() error = %v, want nil (unlimited TPM)", err)
	}
}

func TestRateLimiter_ConsumeTokens_ZeroCount(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{TPM: 100})
	err := rl.ConsumeTokens(context.Background(), 0)
	if err != nil {
		t.Errorf("ConsumeTokens(0) error = %v, want nil", err)
	}
}

func TestRateLimiter_ConsumeTokens_WithinBudget(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{TPM: 1000})
	err := rl.ConsumeTokens(context.Background(), 100)
	if err != nil {
		t.Fatalf("ConsumeTokens() error = %v", err)
	}

	rl.mu.Lock()
	remaining := rl.tpmTokens
	rl.mu.Unlock()

	// Should have consumed 100 tokens, leaving ~900.
	if remaining > 910 || remaining < 890 {
		t.Errorf("remaining tokens = %f, expected ~900", remaining)
	}
}

func TestRateLimiter_ConsumeTokens_ExceedsBudget_Blocks(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{TPM: 100})

	// Consume all available tokens.
	err := rl.ConsumeTokens(context.Background(), 100)
	if err != nil {
		t.Fatalf("first ConsumeTokens() error = %v", err)
	}

	// Next call should block and timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = rl.ConsumeTokens(ctx, 100)
	if err == nil {
		t.Error("expected timeout when tokens exhausted")
	}
}

func TestRateLimiter_ConsumeTokens_ContextCancelled(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{TPM: 1})
	// Exhaust tokens.
	_ = rl.ConsumeTokens(context.Background(), 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.ConsumeTokens(ctx, 1)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestRateLimiter_Allow_ContextCancelled(t *testing.T) {
	rl := NewRateLimiter(ProviderLimits{RPM: 1})
	// Exhaust RPM.
	_ = rl.Allow(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.Allow(ctx)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestProviderLimits_Fields(t *testing.T) {
	limits := ProviderLimits{
		RPM:             60,
		TPM:             100000,
		MaxConcurrent:   10,
		CooldownOnRetry: 5 * time.Second,
	}
	if limits.RPM != 60 {
		t.Errorf("RPM = %d, want 60", limits.RPM)
	}
	if limits.TPM != 100000 {
		t.Errorf("TPM = %d, want 100000", limits.TPM)
	}
	if limits.MaxConcurrent != 10 {
		t.Errorf("MaxConcurrent = %d, want 10", limits.MaxConcurrent)
	}
	if limits.CooldownOnRetry != 5*time.Second {
		t.Errorf("CooldownOnRetry = %v, want 5s", limits.CooldownOnRetry)
	}
}

func TestRateLimiter_RPM_RefillsOverTime(t *testing.T) {
	// High RPM so refill is fast.
	rl := NewRateLimiter(ProviderLimits{RPM: 6000}) // 100/sec

	// Consume all tokens.
	for i := 0; i < 6000; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		err := rl.Allow(ctx)
		cancel()
		if err != nil {
			break
		}
		rl.Release()
	}

	// Wait a bit for refill.
	time.Sleep(50 * time.Millisecond)

	// Should have some tokens now.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := rl.Allow(ctx)
	if err != nil {
		t.Errorf("Allow() after refill error = %v", err)
	}
}
