BINARY_NAME := "minid"

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# Update the ldflags with the app, client & server names
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=mini \
	-X github.com/cosmos/cosmos-sdk/version.AppName=${BINARY_NAME} \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

###########
# Install #
###########

all: install

.PHONY: install
install:
	@echo "--> ensure dependencies have not been modified"
	@go mod verify
	@echo "--> installing minid"
	@go install $(BUILD_FLAGS) -mod=readonly ./cmd/minid

.PHONY: init
init:
	./scripts/init.sh

.PHONY: build
build:
	@echo Building xchange app
	go build -C ./cmd/minid -o ${BINARY_NAME}
