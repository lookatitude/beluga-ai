.PHONY: build test test-ci lint check security

# Release binaries: copy from framework's own .goreleaser.yml when you
# need tagged release automation (https://goreleaser.com).

build:
	go build ./...

test:
	go test -race ./...

# test-ci is the deterministic dev-loop test target. It pins the mock LLM
# provider, a fixed RNG seed, and disables OTel exports so a `beluga test`
# invocation (and CI) produces the same output regardless of environment.
# See framework docs/reference/cli.md for the canonical env contract.
test-ci:
	BELUGA_LLM_PROVIDER=mock \
	BELUGA_DETERMINISTIC=1 \
	BELUGA_SEED=42 \
	OTEL_SDK_DISABLED=true \
	go test -race ./...

lint:
	golangci-lint run ./...

check: build test lint

# security runs gosec and govulncheck. They are not in `check` by default
# because gosec can be chatty on new projects; wire them into CI when you
# are ready. Install once:
#   go install github.com/securego/gosec/v2/cmd/gosec@latest
#   go install golang.org/x/vuln/cmd/govulncheck@latest
# security:
# 	gosec -quiet ./...
# 	govulncheck ./...
