# Missing Test Files Analysis

**Generated**: $(date -u +%Y-%m-%dT%H:%M:%SZ)
**Purpose**: Identify packages missing test_utils.go and advanced_test.go

## Packages Missing test_utils.go

Based on analysis of all packages in `pkg/`, the following packages are missing `test_utils.go`:

### Top-Level Packages
- None - All top-level packages have test_utils.go

### Sub-Packages Analysis
- `pkg/agents/tools` - Missing test_utils.go
- `pkg/agents/tools/api` - Missing test_utils.go
- `pkg/agents/tools/gofunc` - Missing test_utils.go
- `pkg/agents/tools/mcp` - Missing test_utils.go
- `pkg/agents/tools/shell` - Missing test_utils.go
- `pkg/chatmodels/iface` - Missing test_utils.go (interface-only package, may not need)
- `pkg/chatmodels/registry` - Missing test_utils.go
- `pkg/config/iface` - Missing test_utils.go (interface-only package, may not need)
- `pkg/core/iface` - Missing test_utils.go (interface-only package, may not need)
- `pkg/core/model` - Missing test_utils.go
- `pkg/core/utils` - Missing test_utils.go
- `pkg/documentloaders/iface` - Missing test_utils.go (interface-only package, may not need)
- `pkg/embeddings/iface` - Missing test_utils.go (interface-only package, may not need)
- `pkg/embeddings/registry` - Missing test_utils.go
- `pkg/llms/iface` - Missing test_utils.go (interface-only package, may not need)
- `pkg/schema/iface` - Missing test_utils.go (interface-only package, may not need)

**Note**: Interface-only packages (`iface/` directories) may not require test_utils.go as they typically contain only interface definitions.

## Packages Missing advanced_test.go

Based on analysis of all packages in `pkg/`, the following packages are missing `advanced_test.go`:

### Top-Level Packages
- None - All top-level packages have advanced_test.go

### Sub-Packages Analysis
- `pkg/agents/tools` - Missing advanced_test.go
- `pkg/agents/tools/api` - Missing advanced_test.go
- `pkg/agents/tools/gofunc` - Missing advanced_test.go
- `pkg/agents/tools/mcp` - Missing advanced_test.go
- `pkg/agents/tools/shell` - Missing advanced_test.go
- `pkg/chatmodels/iface` - Missing advanced_test.go (interface-only package, may not need)
- `pkg/chatmodels/registry` - Missing advanced_test.go
- `pkg/config/iface` - Missing advanced_test.go (interface-only package, may not need)
- `pkg/core/iface` - Missing advanced_test.go (interface-only package, may not need)
- `pkg/core/model` - Missing advanced_test.go
- `pkg/core/utils` - Missing advanced_test.go
- `pkg/documentloaders/iface` - Missing advanced_test.go (interface-only package, may not need)
- `pkg/embeddings/iface` - Missing advanced_test.go (interface-only package, may not need)
- `pkg/embeddings/registry` - Missing advanced_test.go
- `pkg/llms/iface` - Missing advanced_test.go (interface-only package, may not need)
- `pkg/schema/iface` - Missing advanced_test.go (interface-only package, may not need)

**Note**: Interface-only packages (`iface/` directories) may not require advanced_test.go as they typically contain only interface definitions.

## Summary

- **Total top-level packages**: 19
- **Packages with test_utils.go**: 19 (100%)
- **Packages with advanced_test.go**: 19 (100%)
- **Sub-packages requiring test files**: See list above
- **Interface-only packages (excluded)**: Multiple iface/ directories

## Action Items

1. Create test_utils.go for sub-packages that have implementation code
2. Create advanced_test.go for sub-packages that have implementation code
3. Document exclusions for interface-only packages
4. Focus on top-level packages first (all have required files)
