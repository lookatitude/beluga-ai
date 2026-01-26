# CI-Local Steps Order

`make ci-local` runs all CI checks in specific order.

```makefile
ci-local: ## Run all CI checks locally (matches CI workflow)
    @echo "🚀 Running comprehensive CI checks locally..."
    @echo "📋 Step 1: Format check..."
    @$(MAKE) fmt-check
    @echo "🔍 Step 2: Lint & Format (advisory - warnings don't block)..."
    @$(MAKE) lint || (echo "⚠️  Linting issues found (advisory)" && true)
    @echo "🔍 Step 3: Go vet..."
    @$(MAKE) vet
    @echo "🔒 Step 4: Security scans..."
    @$(MAKE) security
    @echo "🧪 Step 5: Unit tests..."
    @$(MAKE) test-unit
    @echo "🔗 Step 6: Integration tests..."
    @$(MAKE) test-integration
    @echo "📈 Step 7: Coverage check..."
    @$(MAKE) test-coverage-threshold
    @echo "🔨 Step 8: Build verification..."
    @$(MAKE) build
    @echo "✅ All CI checks passed!"
```

## Step Order Rationale
1. **Format check** - Fast, catches obvious issues
2. **Lint (advisory)** - Warnings only, don't block
3. **Vet** - Go's built-in static analysis
4. **Security** - gosec, govulncheck, gitleaks
5. **Unit tests** - Fast feedback loop
6. **Integration tests** - Slower, but critical
7. **Coverage** - Advisory metric
8. **Build** - Final verification

## Advisory vs Blocking

| Step | Blocks? | Why |
|------|---------|-----|
| Format | Yes | Must match CI |
| Lint | No | Warnings acceptable |
| Vet | Yes | Catches real bugs |
| Security | Yes | Critical issues |
| Tests | Yes | Must pass |
| Coverage | No | Advisory metric |
| Build | Yes | Must compile |
