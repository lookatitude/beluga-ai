# Cache Directory Structure

All build artifacts go to `.cache/` with organized subdirectories.

```makefile
CACHE_DIR := .cache
TEST_BIN_DIR := $(CACHE_DIR)/test-binaries
GO_CACHE_DIR := $(CACHE_DIR)/go-build

test-build:
    @mkdir -p $(TEST_BIN_DIR)
    @for pkg in $$(go list ./pkg/...); do \
        name=$$(basename $$pkg); \
        go test -c -o $(TEST_BIN_DIR)/$$name.test $$pkg; \
    done

build-examples:
    @mkdir -p $(CACHE_DIR)/bin
    @for dir in examples/*/*/; do \
        name=$$(basename $$(dirname $$dir))_$$(basename $$dir); \
        go build -o $(CACHE_DIR)/bin/$$name $$dir; \
    done
```

## Directory Layout
```
.cache/
├── bin/              # Example binaries (category_name format)
├── test-binaries/    # Pre-compiled test binaries
├── go-build/         # Go build cache (GOCACHE)
└── benchmarks/       # Benchmark results
```

## Naming Conventions
| Type | Pattern | Example |
|------|---------|---------|
| Test binary | `{package}.test` | `agents.test` |
| Example binary | `{category}_{name}` | `rag_simple` |
| Benchmark | `bench.txt` | - |

## Clean Target
```makefile
clean:
    @rm -rf $(CACHE_DIR)/bin
    @rm -rf $(CACHE_DIR)/test-binaries
    # Note: go-build cache preserved for speed
```

## .gitignore
Add `.cache/` to gitignore.
