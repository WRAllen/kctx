.PHONY: build install test

build:
	go build -o bin/kctx ./cmd/kctx

install:
	go install ./cmd/kctx

test:
	go test ./...
