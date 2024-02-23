#!/usr/bin/make -f

# Makefile for kv store application

BINARY_NAME=xchanged
GOLANGCI_VERS=v1.54.1

all: build

build:
	@echo Building xchange app
	go build -C ./cmd/xchanged -o ${BINARY_NAME}
.PHONY: build

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERS)
	golangci-lint run  ./... --config ../../.golanglint.yml
.PHONY: lint

format:
	find . -name '*.go' -type f | xargs gofumpt -w -l
	golangci-lint run --fix --config ../../.golanglint.yml
.PHONY: format
