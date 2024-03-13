#!/usr/bin/make -f

# Makefile for cosmux application

BINARY_NAME=cosmux
GOLANGCI_VERS=v1.54.1

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=cosmux \
	-X github.com/cosmos/cosmos-sdk/version.AppName=cosmux \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'


all: build test

build:
	  go build -o ${BINARY_NAME} ./...
.PHONY: build

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERS)
	golangci-lint run  ./... --config ../.golanglint.yml
.PHONY: lint

format:
	find . -name '*.go' -type f | xargs gofumpt -w -l
	golangci-lint run --fix --config ../.golanglint.yml
.PHONY: format

install:
	@echo "--> ensure dependencies have not been modified"
	@go mod verify
	@echo "--> installing cosmux"
	@go install $(BUILD_FLAGS) -mod=readonly ./
.PHONY: install

init:
	./scripts/init.sh
.PHONY: init