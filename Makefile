BINARY := bin/dojo
PKG    := ./...
PREFIX ?= $(HOME)/.local

.PHONY: all build install test vet validate clean fmt dogfood study

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

# Dogfooding a lab records the same attempt a learner's own work records, and
# recommend, learn and readiness cannot tell them apart. The marker sends this
# checkout's attempts to progress-dev.json instead; `make study` removes it.
dogfood:
	@touch .dojo-dev
	@echo "dev mode on: attempts from this directory record to ~/.cka-dojo/progress-dev.json"

study:
	@rm -f .dojo-dev
	@echo "dev mode off: attempts record to your study history"

check: vet test validate

clean:
	rm -rf bin
