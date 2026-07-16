# Equiv of former .goreleaser.yaml: linux/amd64 binary + tar.gz + checksums.

APP       := etarantula
DIST      := dist
BINARY    := $(DIST)/$(APP)
GOOS      := linux
GOARCH    := amd64

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
ARCHIVE   := $(DIST)/$(APP)_$(VERSION)_linux_x86_64.tar.gz
CHECKSUMS := $(DIST)/checksums.txt

.PHONY: all build package clean

all: package

build:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY) .

package: build
	@tmpdir=$$(mktemp -d) && \
	cp $(BINARY) $$tmpdir/$(APP) && \
	cp config.yaml README.md $$tmpdir/ && \
	if [ -d js ]; then cp -R js $$tmpdir/; fi && \
	tar -C $$tmpdir -czf $(ARCHIVE) . && \
	rm -rf $$tmpdir
	cd $(DIST) && shasum -a 256 $$(basename $(ARCHIVE)) > $$(basename $(CHECKSUMS))
	@echo "built $(ARCHIVE)"
	@echo "checksums -> $(CHECKSUMS)"

clean:
	rm -rf $(DIST)
