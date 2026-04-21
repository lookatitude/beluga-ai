.PHONY: build test test-ci lint check security eval-ci

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

# eval-ci runs the smoke evaluation with the mock LLM provider and a
# deterministic fixture dataset so CI gets pass/fail signal without
# paid API calls. The env vars mirror test-ci plus OTEL_SDK_DISABLED
# to keep the eval child's stdout clean (the IPC payload is JSON on
# stdout; a stdout OTel exporter would corrupt it). See framework
# docs/reference/cli.md for the canonical eval env contract.
#
# Requires the `beluga` binary on PATH. Install once (versioned to your
# go.mod via the current module):
#   go install github.com/lookatitude/beluga-ai/v2/cmd/beluga
# `go run github.com/.../cmd/beluga` would force a resolved go.sum entry
# for every transitive dep of the full CLI (fsnotify, cobra, every
# provider) — which a scaffolded project's tidy'd go.sum does not carry.
eval-ci:
	BELUGA_LLM_PROVIDER=mock \
	BELUGA_DETERMINISTIC=1 \
	BELUGA_SEED=42 \
	OTEL_SDK_DISABLED=true \
	beluga eval .beluga/eval.smoke.json

# security runs gosec and govulncheck. They are not in `check` by default
# because gosec can be chatty on new projects; wire them into CI when you
# are ready. Install once:
#   go install github.com/securego/gosec/v2/cmd/gosec@latest
#   go install golang.org/x/vuln/cmd/govulncheck@latest
# security:
# 	gosec -quiet ./...
# 	govulncheck ./...
