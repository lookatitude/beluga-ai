# Quickstart: Fix Corrupted Mock Files

## Overview
This quickstart validates that the corrupted mock files have been fixed and the beluga-ai package compiles successfully.

## Prerequisites
- Go 1.24.0 or later
- Repository cloned and checked out to `002-beluga-ai-dependency` branch

## Validation Steps

### Step 1: Verify Package Declarations
Check that all mock files have package declarations:

```bash
# Check each file starts with package declaration
head -n 1 pkg/core/di_mock.go | grep -q "^package core$" && echo "✓ core/di_mock.go" || echo "✗ core/di_mock.go"
head -n 1 pkg/prompts/advanced_mock.go | grep -q "^package prompts$" && echo "✓ prompts/advanced_mock.go" || echo "✗ prompts/advanced_mock.go"
head -n 1 pkg/memory/advanced_mock.go | grep -q "^package memory$" && echo "✓ memory/advanced_mock.go" || echo "✗ memory/advanced_mock.go"
head -n 1 pkg/vectorstores/advanced_mock.go | grep -q "^package vectorstores$" && echo "✓ vectorstores/advanced_mock.go" || echo "✗ vectorstores/advanced_mock.go"
head -n 1 pkg/vectorstores/iface/iface_mock.go | grep -q "^package vectorstores$" && echo "✓ vectorstores/iface/iface_mock.go" || echo "✗ vectorstores/iface/iface_mock.go"
```

**Expected**: All files show ✓

### Step 2: Build All Packages
Verify all packages compile successfully:

```bash
go build ./pkg/...
```

**Expected**: Build succeeds with no errors

### Step 3: Run Tests
Verify all tests pass:

```bash
go test ./pkg/...
```

**Expected**: All tests pass

### Step 4: Verify Module Integrity
Check module checksums:

```bash
go mod verify
```

**Expected**: "all modules verified"

### Step 5: Test Module Import (Optional)
If you have a test project that depends on beluga-ai:

```bash
cd /path/to/test-project
go mod download
go build
```

**Expected**: Test project compiles without errors related to missing package declarations

## Success Criteria

✅ **All validation steps pass**
- All 5 mock files have correct package declarations
- `go build ./pkg/...` succeeds
- `go test ./pkg/...` passes
- `go mod verify` passes
- Dependent projects can compile successfully

## Troubleshooting

### Issue: Build still fails
**Solution**: Verify package declarations were added correctly:
```bash
head -n 1 pkg/core/di_mock.go
# Should show: package core
```

### Issue: Tests fail
**Solution**: Check if tests were affected by the fix. The fix should not change test behavior, only add package declarations.

### Issue: Module verification fails
**Solution**: This should not happen as we're only adding package declarations. If it does, check for accidental changes to other files.

## Next Steps

After validation passes:
1. Commit the changes
2. Create a pull request
3. Tag a new release (e.g., v1.3.1)
4. Publish to Go module registry

