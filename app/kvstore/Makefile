#!/usr/bin/make -f

# Makefile for kv store application

BINARY_NAME=kvstore
GOLANGCI_VERS=v1.54.1

all: build

.PHONY: build
build:
	@echo Building kvstore
	go build -o ${BINARY_NAME} ./...

.PHONY: lint
lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERS)
	golangci-lint run  ./... --config ../../.golanglint.yml

.PHONY: format
format:
	find . -name '*.go' -type f | xargs gofumpt -w -l
	golangci-lint run --fix --config ../../.golanglint.yml

.PHONY: install
install:
	@go install ./
	