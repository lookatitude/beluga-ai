.PHONY: build test lint check security

# Release binaries: copy from framework's own .goreleaser.yml when you
# need tagged release automation (https://goreleaser.com).

build:
	go build ./...

test:
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
