# Validation Contract: Mock File Package Declarations

## Contract Overview
All Go source files in the beluga-ai package must start with a valid `package` declaration.

## Validation Rules

### Rule 1: Package Declaration Required
**Requirement**: Every `.go` file MUST start with `package <name>` declaration.

**Validation**:
```bash
# Check all .go files have package declaration
for file in $(find pkg -name "*.go"); do
  first_line=$(head -n 1 "$file")
  if [[ ! "$first_line" =~ ^package\  ]]; then
    echo "ERROR: $file missing package declaration"
    exit 1
  fi
done
```

**Expected Result**: All files pass validation.

### Rule 2: Package Name Matches Directory
**Requirement**: Package name MUST match the directory structure.

**Validation**:
```bash
# Verify package names match directories
pkg/core/*.go → package core
pkg/prompts/*.go → package prompts
pkg/memory/*.go → package memory
pkg/vectorstores/*.go → package vectorstores
pkg/vectorstores/iface/*.go → package vectorstores
```

**Expected Result**: All package names match their directory structure.

### Rule 3: Files Compile Successfully
**Requirement**: All packages MUST compile with `go build`.

**Validation**:
```bash
go build ./pkg/...
```

**Expected Result**: Build succeeds with exit code 0.

### Rule 4: Tests Pass
**Requirement**: All existing tests MUST continue to pass.

**Validation**:
```bash
go test ./pkg/...
```

**Expected Result**: All tests pass.

## Contract Tests

### Test: Package Declaration Presence
```go
// Contract: All .go files must have package declaration
func TestAllGoFilesHavePackageDeclaration(t *testing.T) {
    // Implementation: Check first line of each .go file
}
```

### Test: Package Name Correctness
```go
// Contract: Package names match directory structure
func TestPackageNamesMatchDirectories(t *testing.T) {
    // Implementation: Verify package name matches directory
}
```

### Test: Compilation Success
```go
// Contract: All packages compile successfully
func TestPackagesCompile(t *testing.T) {
    // Implementation: Run go build
}
```

## Success Criteria
- ✅ All 5 mock files have correct package declarations
- ✅ `go build ./pkg/...` succeeds
- ✅ `go test ./pkg/...` passes
- ✅ `go mod verify` passes

