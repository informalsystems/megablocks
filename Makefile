SUBDIRS := $(wildcard ./app/*/. ./cosmux/.)

# builds all applications in ./app
build: $(SUBDIRS)
$(SUBDIRS):
	$(MAKE) -C $@ build

.PHONY: build $(SUBDIRS)

test-e2e:
	go clean -testcache
	go test -v ./tests/e2e/...
.PHONY: test-e2e
