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

.PHONY: test-ut
test-ut:
	go test ./cosmux

test-e2e: build
	go clean -testcache
	go test -v ./tests/e2e/...
.PHONY: test-e2e

install:
	$(MAKE) -C ./cosmux install
	$(MAKE) -C ./app/sdk-chain-a install
	${MAKE} -C ./app/kvstore install
.PHONY: install

.PHONY: clean
clean:
	killall cosmux;\
	killall minid; \
	killall kvstore; \
	rm  -f /tmp/kvapp.sock; \
	rm  -f /tmp/mind.sock;
