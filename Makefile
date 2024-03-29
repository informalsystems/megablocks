SUBDIRS := $(wildcard ./app/*/. ./cosmux)

# builds all applications in ./app
build: $(SUBDIRS)
$(SUBDIRS):
	$(MAKE) -C $@ build
.PHONY: build $(SUBDIRS)

init:
	$(MAKE) -C ./cosmux init; \
	$(MAKE) -C ./app/sdk-chain-a init; \


lint:
	for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir lint; \
	done
.PHONY: lint

test-e2e: build
	go clean -testcache
	go test -v ./tests/e2e/...
.PHONY: test-e2e

install:
	$(MAKE) -C ./cosmux install
	$(MAKE) -C ./app/sdk-chain-a install
	${MAKE} -C ./app/kvstore install
.PHONY: install
