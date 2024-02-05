SUBDIRS := $(wildcard ./app/*/.)

# builds all applications in ./app
build: $(SUBDIRS)
$(SUBDIRS):
	$(MAKE) -C $@ build

.PHONY: build $(SUBDIRS)

