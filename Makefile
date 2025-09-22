.PHONY: tidy fmt test run

GOCACHE := $(PWD)/.cache/go-build
GOENV := $(PWD)/.config/go/env

export GOCACHE
export GOENV

fmt:
	go fmt ./...

tidy:
	go mod tidy

test:
	go test ./...

run:
	go run ./cmd/server --config-dir=./config
