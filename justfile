set shell := ["bash", "-cu"]

default:
    @just --list

setup:
    go install golang.org/x/tools/cmd/goimports@latest
    go install github.com/segmentio/golines@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

build:
    go build -o bin/chtop ./cmd/chtop

install:
    #!/usr/bin/env bash
    set -euo pipefail
    target="${GOBIN:-${GOPATH:-$HOME/go}/bin}"
    mkdir -p "$target"
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -extldflags "-static"' \
        -o "$target/chtop" ./cmd/chtop
    echo "installed: $target/chtop"
    case ":$PATH:" in
        *":$target:"*) ;;
        *) echo "warning: $target is not in your PATH" >&2 ;;
    esac

run *ARGS:
    go run ./cmd/chtop {{ARGS}}

test:
    go test ./...

test-race:
    go test -race ./...

lint:
    golangci-lint run ./...

fmt:
    gofmt -s -w .
    goimports -w .
    golines -w --max-len=100 --base-formatter=gofmt .

tidy:
    go mod tidy

clean:
    rm -rf bin/
