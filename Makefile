BIN := bin/rekord
VERSION ?= dev
LDFLAGS := -s -w -X github.com/Omotolani98/rekord/internal/cli.version=$(VERSION)
GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: build test lint fmt run clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/rekord

test:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run

fmt:
	gofmt -w $(GOFILES)

run:
	go run ./cmd/rekord $(ARGS)

clean:
	rm -rf bin dist
