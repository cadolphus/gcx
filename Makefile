BINARY_NAME=gcx
BIN_DIR=bin
INSTALL_DIR?=$(shell go env GOPATH)/bin

VERSION?=1.0.0
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "HEAD")
DATE?=$(shell date -u +%Y-%m-%d)
LDFLAGS=-s -w -X github.com/cadolphus/gcx/cmd.Version=$(VERSION) -X github.com/cadolphus/gcx/cmd.GitCommit=$(COMMIT) -X github.com/cadolphus/gcx/cmd.BuildDate=$(DATE)

.PHONY: all build install test clean fmt vet

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .

install:
	go install -ldflags="$(LDFLAGS)" .

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)
