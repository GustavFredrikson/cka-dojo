BINARY := bin/dojo
PKG    := ./...
PREFIX ?= $(HOME)/.local

.PHONY: all build install test vet validate clean fmt

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/dojo

install: build
	@mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/dojo
	@echo "installed $(DESTDIR)$(PREFIX)/bin/dojo"

test:
	go test $(PKG)

vet:
	go vet $(PKG)

fmt:
	gofmt -l -w .

# Content linting uses the freshly built binary against the in-repo content.
validate: build
	DOJO_CONTENT=$(CURDIR) $(BINARY) content validate

check: vet test validate

clean:
	rm -rf bin
